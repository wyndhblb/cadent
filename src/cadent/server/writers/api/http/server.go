/*
Copyright 2014-2017 Bo Blanton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
   Fire up an HTTP server for an http interface to the

   metrics/indexer interfaces

   example config
   [graphite-proxy-map.accumulator.api]
        base_path = "/graphite/"
        listen = "0.0.0.0:8083"
            [graphite-proxy-map.accumulator.api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

            [graphite-proxy-map.accumulator.api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"
*/

package http

import (
	"cadent/server/broadcast"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils/shutdown"
	"cadent/server/utils/tomlenv"
	"cadent/server/writers/api"
	"cadent/server/writers/api/discovery"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	grpcserver "cadent/server/writers/rpc/server"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/tinylib/msgp/msgp"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"gopkg.in/yaml.v2"
	"io"
	golog "log"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
)

const MAX_METRIC_POINTS uint32 = 20480
const DEFAULT_MIN_RESOLUTION uint32 = 1

type ApiLoop struct {
	Conf         ApiConfig          // our config
	Metrics      metrics.Metrics    // metrics interface
	Indexer      indexer.Indexer    // indexer interface
	Discover     discovery.Discover // discovery registration
	DiscoverList discovery.Lister   // discovery Lister
	CadentHosts  []*url.URL         // hardcoded hosts
	RpcServer    *grpcserver.Server //gRPC server
	Router       *mux.Router        // main router
	WSMux        *http.ServeMux     // websocket mux

	rpcClient *rpcClients

	Tracer      opentracing.Tracer
	TraceCloser io.Closer

	info *api.InfoData
	tags *repr.SortingTags

	shutdown *broadcast.Broadcaster
	log      *logging.Logger

	ReadCache         *metrics.ReadCache // internal readcache, needed for websockets
	activateCacheChan chan *smetrics.RawRenderItem

	started bool
}

func ParseConfigString(inconf string) (rl *ApiLoop, err error) {

	rl = new(ApiLoop)
	if _, err := tomlenv.Decode(inconf, &rl.Conf); err != nil {
		return nil, err
	}

	rl.Metrics, err = rl.Conf.GetMetrics(10.0) // stub "10s" as the res
	if err != nil {
		return nil, err
	}

	rl.Indexer, err = rl.Conf.GetIndexer()
	if err != nil {
		return nil, err
	}
	rl.started = false
	rl.Metrics.SetIndexer(rl.Indexer)
	rl.SetBasePath(rl.Conf.BasePath)
	rl.log = logging.MustGetLogger("reader.http")

	return rl, nil
}

func (re *ApiLoop) Config(conf ApiConfig, resolution float64) (err error) {
	if re.log == nil {
		re.log = logging.MustGetLogger("reader.http")
	}

	if re.shutdown == nil {
		re.shutdown = broadcast.New(1)
	}

	if conf.Logfile == "" {
		conf.Logfile = "stdout"
	}
	re.Conf = conf

	re.Metrics, err = conf.GetMetrics(resolution)
	if err != nil {

		return err
	}

	re.Indexer, err = conf.GetIndexer()
	if err != nil {
		return err
	}
	re.Metrics.SetIndexer(re.Indexer)
	re.SetBasePath(conf.BasePath)

	// discover (ok to be nil)
	re.Discover, err = conf.GetDiscover()
	if err != nil {
		re.log.Critical("Error in Discover module: %v", err)
		return err
	}
	if re.Discover == nil {
		re.log.Noticef("No discovery module set, not registring")
	} else {
		// set up the lister
		re.DiscoverList, err = conf.GetDiscoverList()
		if err != nil {
			re.log.Critical("Error in Discover List module: %v", err)
			return err
		}
	}

	// readcache
	mxRam := metrics.READ_CACHER_TOTAL_RAM
	mxStats := metrics.READ_CACHER_MAX_SERIES_BYTES
	if conf.MaxReadCacheBytes > 0 {
		mxRam = conf.MaxReadCacheBytes
	}
	if conf.MaxReadCacheBytesPerMetric > 0 {
		mxStats = conf.MaxReadCacheBytesPerMetric
	}

	re.ReadCache = metrics.InitReadCache(mxRam, mxStats, metrics.READ_CACHER_MAX_LAST_ACCESS)

	if len(conf.Tags) > 0 {
		re.tags = repr.SortingTagsFromString(conf.Tags)
	}

	// hardcoded clusters addresses
	for _, h := range conf.CadentHosts {
		parsed, err := url.Parse(h)
		if err != nil {
			re.log.Critical("Hardcoded cadent url parse failure: %v", err)
			return err
		}
		re.CadentHosts = append(re.CadentHosts, parsed)
	}

	// set up tracing
	re.Tracer, re.TraceCloser, err = conf.GetTracer()

	return err
}

// GetSpan helper for get a span from the tracer
func (re *ApiLoop) GetSpan(name string, req *http.Request) (opentracing.Span, context.Context) {
	ctx := req.Context()
	var parentCtx opentracing.SpanContext
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan != nil {
		parentCtx = parentSpan.Context()
	}

	span := re.Tracer.StartSpan(name, opentracing.ChildOf(parentCtx))
	span.SetTag("span.kind", "server")
	span.SetTag("component", "cadent-api-server")

	return span, ctx
}

// AddSpanLog for opentracing add a span log field
func (re *ApiLoop) AddSpanLog(span opentracing.Span, tag string, msg string) {
	span.LogFields(otlog.String(tag, msg))
}

// AddSpanEvent for opentracing add a span log field
func (re *ApiLoop) AddSpanEvent(span opentracing.Span, msg string) {
	span.LogEvent(msg)
}

// AddSpanError for opentracing add error log + field
func (re *ApiLoop) AddSpanError(span opentracing.Span, err error) {
	span.SetTag("error", true)
	span.LogFields(
		otlog.String("event", "error"),
		otlog.String("message", err.Error()),
	)
}

// SpanStartEnd a generic "timer" like event for things returns a function to "defer"
// i.e. `defer api.SpanStartEnd(span)()`
func (re *ApiLoop) SpanStartEnd(span opentracing.Span) func() {
	re.AddSpanEvent(span, "started")
	return func() {
		re.AddSpanEvent(span, "ended")
		span.Finish()
	}
}

func (re *ApiLoop) activateCacheLoop() {
	for {
		select {
		case data, more := <-re.activateCacheChan:
			if !more {
				return
			}
			if data == nil {
				continue
			}
			re.ReadCache.ActivateMetricFromRenderData(data)
		}
	}
}

func (re *ApiLoop) AddToCache(data []*smetrics.RawRenderItem) {
	if re.activateCacheChan == nil {
		return
	}
	for _, d := range data {
		// send to activator
		re.activateCacheChan <- d
	}
}

func (re *ApiLoop) SetBasePath(pth string) {
	re.Conf.BasePath = pth
	if len(re.Conf.BasePath) == 0 {
		re.Conf.BasePath = "/"
	}
	if !strings.HasSuffix(re.Conf.BasePath, "/") {
		re.Conf.BasePath += "/"
	}
	if !strings.HasPrefix(re.Conf.BasePath, "/") {
		re.Conf.BasePath = "/" + re.Conf.BasePath
	}
}

func (re *ApiLoop) SetResolutions(res [][]int) {
	re.Metrics.SetResolutions(res)
}

func (re *ApiLoop) withRecover(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			msg := fmt.Sprintf("Panic/Recovered: Error: %s", err)
			re.log.Critical(msg)
			debug.PrintStack()
		}
	}()

	fn()
}

func (re *ApiLoop) OutError(w http.ResponseWriter, msg string, code int) {

	defer stats.StatsdClient.Incr("reader.http.errors", 1)
	w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
	w.Header().Set("Content-Type", "text/plain")
	http.Error(w, msg, code)
	re.log.Error(msg)
}

func (re *ApiLoop) OutOk(w http.ResponseWriter, data interface{}, format string) {
	switch format {
	case "msgpack":
		w.Header().Set("X-Cadent-Compress", "false")
		d, ok := data.(msgp.Encodable)
		if !ok {
			re.OutError(w, "object does not have a msgpack encoder", http.StatusServiceUnavailable)
			return
		}
		re.OutMsgpack(w, d)
	case "protobuf":
		w.Header().Set("X-Cadent-Compress", "false")
		re.OutProtobuf(w, data)
	case "yaml":
		re.OutYaml(w, data)
	default:
		re.OutJson(w, data)
	}
}

// OutJson generic output in json formats
func (re *ApiLoop) OutJson(w http.ResponseWriter, data interface{}) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("Json Out Render Err: %v", r)
			re.log.Critical(msg)
			re.OutError(w, msg, http.StatusInternalServerError)
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.ok", 1)
	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(data)

}

// OutYaml generic output in yaml formats
// not recommended for anything related to metrics values as it's much more expensive writer
// then msgpack or protobuf
func (re *ApiLoop) OutYaml(w http.ResponseWriter, data interface{}) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("Yaml Out Render Err: %v", r)
			re.log.Critical(msg)
			re.OutError(w, msg, http.StatusInternalServerError)
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.ok", 1)
	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/yaml")

	bs, err := yaml.Marshal(data)
	if err != nil {
		msg := fmt.Sprintf("Yaml Out Render Err: %v", err)
		re.log.Critical(msg)
		re.OutError(w, msg, http.StatusInternalServerError)
		return
	}
	w.Write(bs)

}

func (re *ApiLoop) OutMsgpack(w http.ResponseWriter, data msgp.Encodable) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("Msgpack Out Render Err: %v", r)
			re.log.Critical(msg)
			re.OutError(w, msg, http.StatusInternalServerError)
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.msgpack.ok", 1)
	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/x-msgpack")

	wr := msgp.NewWriter(w)
	err := data.EncodeMsg(wr)

	if err != nil {
		msg := fmt.Sprintf("OutMsgpack MarshalBinary error: %s", err)
		re.log.Critical(msg)
		re.OutError(w, msg, http.StatusInternalServerError)
		return
	}
	wr.Flush()
}

// OutProtobuf some items are "streamed" a delimited by their
// [int] length of the chunk.  It's up to the receiver to know what's what
// and deal w/ the chunking for those formats
func (re *ApiLoop) OutProtobuf(w http.ResponseWriter, data interface{}) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("Protobuf Out Render Err: %v", r)
			re.log.Critical(msg)
			re.OutError(w, msg, http.StatusInternalServerError)
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.http.protobuf.ok", 1)
	w.Header().Set("Cache-Control", "public, max-age=60, cache")
	w.Header().Set("Content-Type", "application/x-protobuf")

	writeBytes := func(b []byte) {
		binary.Write(w, binary.BigEndian, uint32(len(b)))
		w.Write(b)
	}

	bytesProto := func(item proto.Message) ([]byte, error) {
		b, err := proto.Marshal(item)
		if err != nil {
			msg := fmt.Sprintf("OutProtobuf Marshal error: %s", err)
			re.log.Critical(msg)
			re.OutError(w, msg, http.StatusInternalServerError)
			return nil, err
		}
		writeBytes(b)
		return b, err
	}

	outProto := func(outdata interface{}) {
		d, ok := outdata.(proto.Message)
		if !ok {
			re.OutError(w, "object does not have a protobuf encoder", http.StatusServiceUnavailable)
			return
		}
		b, err := bytesProto(d)
		if err != nil {
			msg := fmt.Sprintf("OutProtobuf Marshal error: %s", err)
			re.log.Critical(msg)
			re.OutError(w, msg, http.StatusInternalServerError)
			return
		}
		w.Write(b)
	}

	switch t := data.(type) {
	case sindexer.MetricFindItems:
		// convert to MetricFindItemsList
		outProto(&sindexer.MetricFindItemList{Items: t})
		return

	case []*sindexer.MetricFindItem:
		// convert to MetricFindItemsList
		outProto(&sindexer.MetricFindItemList{Items: t})
		return
	case sindexer.MetricTagItems:
		// convert to list proto object
		outProto(&sindexer.MetricTagItemList{Items: t})
		return

	case []*sindexer.MetricTagItem:
		// convert to list proto object
		outProto(&sindexer.MetricTagItemList{Items: t})
		return
	case smetrics.RenderItems:
		// convert to list proto object
		outProto(&smetrics.RenderItemList{Items: t})
		return
	case smetrics.RawRenderItems:
		// convert to list proto object
		outProto(&smetrics.RawRenderItemList{Items: t})
		return
	case smetrics.GraphiteApiItems:
		outProto(&smetrics.GraphiteApiItemList{Items: t})
		return
	}
	outProto(data)
}

func (re *ApiLoop) NoOp(w http.ResponseWriter, r *http.Request) {
	golog.Printf("No handler for this URL %s", r.URL)
	base := re.Conf.BasePath
	http.Error(w,
		fmt.Sprintf(
			"Nothing here .. try %sfind, %srender, %sexpand, %scache, %scached/series",
			base, base, base, base, base,
		),
		http.StatusNotFound,
	)
	return
}

// based on the min resolution, figure out the real min "resample" to match the max points allowed
func (re *ApiLoop) minResolution(start int64, end int64, cur_step uint32) uint32 {

	use_min := cur_step
	if DEFAULT_MIN_RESOLUTION > use_min {
		use_min = DEFAULT_MIN_RESOLUTION
	}
	cur_pts := uint32(end-start) / use_min

	if re.Metrics == nil {
		if cur_pts > MAX_METRIC_POINTS {
			return uint32(end-start) / MAX_METRIC_POINTS
		}
		return use_min
	}
	return use_min
}

// GetInfoData set up our info struct for discovery and info handlers
func (re *ApiLoop) GetInfoData() *api.InfoData {
	if re.info == nil {
		re.info = new(api.InfoData)
	}
	re.info.Get() // grab the base set of info

	driver := re.Metrics.Driver()
	idxname := re.Indexer.Name()

	var res [][]int
	if re.Metrics != nil {
		res = re.Metrics.GetResolutions()
	}

	re.info.MetricDriver = driver
	re.info.IndexDriver = idxname
	re.info.Resolutions = res
	re.info.ClusterName = re.Conf.ClusterName

	if re.Metrics != nil && re.Metrics.Cache() != nil {
		re.info.CachedMetrics = re.Metrics.Cache().Len()
	}

	re.info.Api.BasePath = re.Conf.BasePath

	if len(re.Conf.TLSCertPath) > 0 && len(re.Conf.TLSKeyPath) > 0 {
		re.info.Api.Scheme = "https"
	}
	spl := strings.Split(re.Conf.Listen, ":")
	re.info.Api.Port = spl[len(spl)-1]

	re.info.AdvertiseName = re.info.Api.Host
	if len(re.Conf.AdvertiseName) > 0 {
		re.info.AdvertiseName = re.Conf.AdvertiseName
	}

	re.info.AdvertiseUrl = re.info.Api.Scheme + "://" + re.info.Api.Host + ":" + re.info.Api.Port + re.info.Api.BasePath
	if len(re.Conf.AdvertiseUrl) > 0 {
		re.info.AdvertiseUrl = re.Conf.AdvertiseUrl
	}

	if len(re.Conf.ClusterHost) > 0 {
		re.info.GRPCHost = re.Conf.ClusterHost
	}

	if re.tags != nil {
		re.info.Tags = *re.tags
	}

	rpcHosts := make([]string, 0)
	for h := range re.rpcClient.Clients() {
		rpcHosts = append(rpcHosts, h)
	}
	re.info.RpcClients = rpcHosts

	return re.info
}

// SetDiscoveryData put this servers info into the discovery world if defined
func (re *ApiLoop) SetDiscoveryData() (err error) {

	// fire up discovery module
	if re.Discover != nil {
		err = re.Discover.Start()
		if err != nil {
			return err
		}

		re.GetInfoData()

		err = re.Discover.Register(re.info)
		if err != nil {
			re.log.Critical("Error setting discovery key: %v", err)
			panic(err)
		}
	}

	if re.DiscoverList != nil {
		err = re.DiscoverList.Start()
		if err != nil {
			re.log.Critical("Error setting DiscoverList object: %v", err)
			panic(err)
		}
	}
	return err
}

func (re *ApiLoop) RegisterHandlers(logfile *os.File) (*mux.Router, *http.ServeMux) {
	re.Router = mux.NewRouter()
	base := re.Router.PathPrefix(re.Conf.BasePath).Subrouter()

	// the mess of handlers
	NewFindAPI(re).AddHandlers(base)
	NewCacheAPI(re).AddHandlers(base)
	NewMetricsAPI(re).AddHandlers(base)
	NewTagAPI(re).AddHandlers(base)
	NewPrometheusAPI(re).AddHandlers(base)
	NewInfoAPI(re).AddHandlers(base)

	// websocket routes (need a brand new one lest the middleware get mixed)
	ws := base.PathPrefix("/ws").Subrouter()
	NewMetricsSocket(re).AddHandlers(ws)

	re.WSMux = http.NewServeMux()
	re.WSMux.Handle("/ws", WriteLog(ws, logfile))

	// Compression can cause a lot of GC pressure on high request rates, the
	// default is off due to this fact
	if re.Conf.EnableGzip {
		re.WSMux.Handle("/", WriteLog(CompressHandler(CorsHandler(base)), logfile))
	} else {
		re.WSMux.Handle("/", WriteLog(CorsHandler(base), logfile))
	}

	return re.Router, re.WSMux
}

// GRPCStart fire up the grpc server
func (re *ApiLoop) GRPCStart() (err error) {

	if len(re.Conf.GPRCListen) <= 0 && len(re.Conf.CadentHosts) > 0 {
		err := fmt.Errorf("Cannot have a cluster config with cadent_hosts defined and no GRPC bind defined")
		re.log.Errorf("%v", err)
		return err
	}

	if len(re.Conf.GPRCListen) <= 0 {
		re.log.Noticef("No gRPC listen defined, not starting gRPC server")
		return nil
	}

	re.RpcServer, err = grpcserver.New()
	if err != nil {
		re.log.Critical("Could not start gRPC server at %s", re.Conf.GPRCListen)
		return err
	}
	if len(re.Conf.TLSCertPath) > 0 && len(re.Conf.TLSKeyPath) > 0 {
		re.RpcServer.CertFile = re.Conf.TLSCertPath
		re.RpcServer.CertKeyFile = re.Conf.TLSKeyPath
	}

	re.log.Notice("Starting gRPC server at %s", re.Conf.GPRCListen)
	re.RpcServer.Tracer = re.Tracer
	re.RpcServer.Listen = re.Conf.GPRCListen
	re.RpcServer.Metrics = re.Metrics
	re.RpcServer.Indexer = re.Indexer
	re.RpcServer.Start()
	shuts := re.shutdown.Listen()
	for {
		select {
		case <-shuts.Ch:
			golog.Print("Shutting down gRPC server...")
			re.RpcServer.Stop()
			return nil
		}
	}
}

func (re *ApiLoop) Start() error {
	re.log.Notice("Starting reader http server on %s, base path: %s", re.Conf.Listen, re.Conf.BasePath)

	var outlog *os.File
	var err error
	if re.Conf.Logfile == "stderr" {
		outlog = os.Stderr
	} else if re.Conf.Logfile == "stdout" {
		outlog = os.Stdout
	} else if re.Conf.Logfile != "none" {
		outlog, err = os.OpenFile(re.Conf.Logfile, os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			golog.Panicf("Could not open Logfile %s, setting to stdout", re.Conf.Logfile)
			outlog = os.Stdout

		}
	}

	var conn net.Listener

	// certs if needed
	if len(re.Conf.TLSKeyPath) > 0 && len(re.Conf.TLSCertPath) > 0 {
		cer, err := tls.LoadX509KeyPair(re.Conf.TLSCertPath, re.Conf.TLSKeyPath)
		if err != nil {
			golog.Panicf("Could not start https server: %v", err)
			return err
		}
		// proper TLS settings for modern times
		config := &tls.Config{
			Certificates:             []tls.Certificate{cer},
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		conn, err = tls.Listen("tcp", re.Conf.Listen, config)

		if err != nil {
			return fmt.Errorf("Could not make http socket: %s", err)
		}

	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", re.Conf.Listen)
		if err != nil {
			return fmt.Errorf("Error resolving: %s", err)
		}
		conn, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return fmt.Errorf("Could not make http socket: %s", err)
		}
	}

	re.rpcClient, err = RpcClients.Init(re.CadentHosts, re, re.Tracer)
	if err != nil {
		re.log.Critical("Failed getting clients: %v", err)
		return err
	}

	// upstream our tracer
	re.Indexer.SetTracer(re.Tracer)
	re.Metrics.SetTracer(re.Tracer)

	re.rpcClient.SetLocalMetrics(re.Metrics)
	re.rpcClient.SetLocalIndexer(re.Indexer)

	// advertise_host is required if clustering
	if len(re.Conf.CadentHosts) > 0 && len(re.Conf.ClusterHost) <= 0 {
		return fmt.Errorf("`advertise_host` is required is using `cadent_hosts`")
	}

	// set the local node to our grpc host
	if len(re.Conf.GPRCListen) > 0 {
		re.rpcClient.SetLocalHost(re.Conf.ClusterHost)
		// add ourselves too
		re.rpcClient.AddGrpcHost(re.Conf.ClusterHost)
	}

	re.activateCacheChan = make(chan *smetrics.RawRenderItem, 256)

	// start up the activateCacheLoop
	go re.activateCacheLoop()

	re.RegisterHandlers(outlog)

	go re.SetDiscoveryData() // set the discovery goodies, in case of bad ZK things, background this

	go http.Serve(conn, nethttp.Middleware(re.Tracer, re.WSMux))

	go re.GRPCStart()

	shuts := re.shutdown.Listen()
	re.started = true

	for {
		select {
		case <-shuts.Ch:
			if re.TraceCloser != nil {
				re.TraceCloser.Close()
			}
			conn.Close()
			golog.Print("Shutdown finished for API http server...")
			close(re.activateCacheChan)
			return nil
		}
	}
}

func (re *ApiLoop) Stop() {
	shutdown.AddToShutdown()
	defer shutdown.ReleaseFromShutdown()

	re.log.Warning("Shutting down API servers")

	if re.Discover != nil {
		re.Discover.Stop()
	}
	if re.DiscoverList != nil {
		re.DiscoverList.Stop()
	}

	if re.shutdown != nil {
		re.shutdown.Send(true)
	}
}
