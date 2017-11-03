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
   simple TCP api interface


   Much like redis or memcache, this supports simple "command -> response" on a socket

   for now, it just support these

   GETM string from to step format(optional){json,msgpack,protobuf:json}
   FINDM string format(optional){json,msgpack,protobuf|default:json}
   GETC string from to format(optional){json,msgpack,protobuf:json}

*/

package tcp

import (
	"bufio"
	"bytes"
	"cadent/server/broadcast"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	"cadent/server/utils/tomlenv"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/tinylib/msgp/msgp"
	logging "gopkg.in/op/go-logging.v1"
	"io"
	golog "log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

const (
	MAX_METRIC_POINTS      uint32 = 20480
	DEFAULT_MIN_RESOLUTION uint32 = 1
	MAX_CLIENTS                   = 8092
	NUM_WORKERS                   = 16
	READ_TIMEOUT                  = 5 // in min
)

var SPACE_BYTE = []byte(" ")
var NEW_LINE_BYTE = []byte("\n")
var ERROR_BYTE = []byte("-ERR")
var TERMINATOR = []byte("\r\n")
var heloSequence = []byte(`Cadent: TCP interface\n\n`)

type TCPLoop struct {
	Conf TCPApiConfig

	Metrics metrics.Metrics
	Indexer indexer.Indexer

	Socket     net.Listener
	AcceptChan chan net.Conn

	numWorkers int
	workChan   chan net.Conn

	shutdown *broadcast.Broadcaster
	log      *logging.Logger
	OutLog   *os.File

	started bool
}

func ParseConfigString(inconf string) (rl *TCPLoop, err error) {

	rl = new(TCPLoop)
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
	rl.log = logging.MustGetLogger("reader.tcp")

	return rl, nil
}

func (re *TCPLoop) Config(conf TCPApiConfig, resolution float64) (err error) {
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
	if re.log == nil {
		re.log = logging.MustGetLogger("reader.tcp")
	}
	mxClients := int(conf.MaxClients)
	if mxClients <= 0 {
		mxClients = MAX_CLIENTS
	}
	re.AcceptChan = make(chan net.Conn, mxClients)

	re.shutdown = broadcast.New(1)

	re.numWorkers = NUM_WORKERS
	if conf.NumWorkers > 0 {
		re.numWorkers = int(conf.NumWorkers)
	}
	re.workChan = make(chan net.Conn, re.numWorkers)

	return nil
}

func (re *TCPLoop) Stop() {
	shutdown.AddToShutdown()
	defer shutdown.ReleaseFromShutdown()

	if re.shutdown != nil {
		re.shutdown.Send(true)
	}
}

func (re *TCPLoop) SetResolutions(res [][]int) {
	re.Metrics.SetResolutions(res)
}

func (re *TCPLoop) withRecover(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			msg := fmt.Sprintf("Panic/Recovered: Error: %s", err)
			re.log.Critical(msg)
			debug.PrintStack()
		}
	}()

	fn()
}

func (re *TCPLoop) OutBinaryBegin(w io.Writer) (int, error) {
	return w.Write([]byte("$BG$"))
}

func (re *TCPLoop) OutBinaryEnd(w io.Writer) (int, error) {
	return w.Write([]byte("\r\n$EN$\r\n"))
}

func (re *TCPLoop) OutError(w net.Conn, error *ErrorCode, aux interface{}) {

	defer stats.StatsdClient.Incr("reader.tcp.errors", 1)
	w.Write(ERROR_BYTE)
	w.Write(SPACE_BYTE)
	error.Write(w)
	w.Write(SPACE_BYTE)

	re.log.Errorf(fmt.Sprintf("Error: %d: %s", error.Code, error.Msg))

	if aux != nil {
		fmt.Fprintf(w, " %v", aux)
		re.log.Errorf(fmt.Sprintf("%v", aux))
	}
	w.Write(NEW_LINE_BYTE)

}

func (re *TCPLoop) OutOk(w net.Conn, data interface{}, format string) {
	switch format {
	case "msgpack":
		d, ok := data.(msgp.Encodable)
		if !ok {
			re.OutError(w, GetError(ErrNoMsgPack), "")
			return
		}
		re.OutMsgpack(w, d)
	case "protobuf":
		re.OutProtobuf(w, data)
	default:
		re.OutJson(w, data)

	}
}

// OutJson generic output in json formats
func (re *TCPLoop) OutJson(w net.Conn, data interface{}) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			if w != nil {
				re.OutError(w, GetError(ErrPanic), r)
			}
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.tcp.ok", 1)
	json.NewEncoder(w).Encode(data)
}

func (re *TCPLoop) OutMsgpack(w net.Conn, data msgp.Encodable) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			re.OutError(w, GetError(ErrPanic), r)
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.tcp.msgpack.ok", 1)
	//re.OutBinaryBegin(w)

	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)

	err := msgp.Encode(buf, data)

	if err != nil {
		re.OutError(w, GetError(ErrMsgPackMarshalError), err)
		return
	}
	binary.Write(w, binary.BigEndian, uint32(buf.Len()))
	w.Write(buf.Bytes())

}

// OutProtobuf some items are "streamed" a delimited by their
// [int] length of the chunk.  It's up to the receiver to know what's what
// and deal w/ the chunking for those formats
func (re *TCPLoop) OutProtobuf(w net.Conn, data interface{}) {

	// trap any encoding issues here
	defer func() {
		if r := recover(); r != nil {
			re.OutError(w, GetError(ErrPanic), r)
			debug.PrintStack()
			return
		}
	}()

	// cache theses things for 60 secs
	defer stats.StatsdClient.Incr("reader.tcp.protobuf.ok", 1)

	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)

	writeBytes := func(b []byte) {
		binary.Write(buf, binary.BigEndian, uint32(len(b)))
		buf.Write(b)
	}

	bytesProto := func(item proto.Message) ([]byte, error) {

		b, err := proto.Marshal(item)
		if err != nil {
			re.OutError(w, GetError(ErrProtobufMarshalError), err)
			return nil, err
		}
		writeBytes(b)
		return b, err
	}

	writeAllBytes := func() {
		binary.Write(w, binary.BigEndian, uint32(buf.Len()))
		w.Write(buf.Bytes())
	}
	//re.OutBinaryBegin(w)

	outProto := func(outdata interface{}) {
		// a protobuf message that is fully encoded
		d, ok := outdata.(proto.Message)
		if !ok {
			re.OutError(w, GetError(ErrNoProtobuf), "")
			return
		}
		b, err := bytesProto(d)
		if err != nil {
			re.OutError(w, GetError(ErrProtobufMarshalError), err)
			return
		}

		buf.Write(b)
		writeAllBytes()
	}

	// these are list types and thus we write [len|uint32][protobuf][len|uint32][protobuf]...
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

	// do all the rest
	outProto(data)
}

func (re *TCPLoop) NoOp(w net.Conn, r *http.Request) {
	re.OutError(w, GetError(ErrNotFound), "")
	return
}

// based on the min resolution, figure out the real min "resample" to match the max points allowed
func (re *TCPLoop) minResolution(start int64, end int64, cur_step uint32) uint32 {

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

// so as to not overwhelm the readers DBs, we queue all commands through a worker based channel
// here we have the worker
func (re *TCPLoop) goWorkers() {
	for w := range re.workChan {
		re.handleRequest(w)
	}
}

func (re *TCPLoop) getCommand(line []byte) (Command, *ErrorCode) {
	bits := bytes.Split(line, repr.SPACE_SEPARATOR_BYTE)
	if len(bits) < 1 {
		return nil, GetError(ErrBadCommand)
	}
	gots, err := GetCommand(string(bits[0]))
	if err != nil {
		return nil, err
	}
	return gots, err
}

// accept incoming TCP connections and push them into the
// a connection channel
func (re *TCPLoop) Accepter(listen net.Listener) error {

	shuts := re.shutdown.Listen()
	go func() {
		defer func() {
			if listen != nil {
				listen.Close()
			}
		}()

		for {
			select {
			case <-shuts.Ch:
				re.log.Warning("TCP Listener: Shutdown gotten .. stopping incoming connections")
				listen.Close()
				shuts.Close()
				return
			default:

			}

			conn, err := listen.Accept()
			if err != nil {
				stats.StatsdClient.Incr("reader.tcp.incoming.failed.connections", 1)
				re.log.Warning("Error Accecption Connection: %s", err)
				continue
			}
			stats.StatsdClient.Incr("reader.tcp.incoming.connections", 1)
			//server.log.Debug("Accepted connection from %s", conn.RemoteAddr())

			re.AcceptChan <- conn
		}
	}()
	return nil
}

func (re *TCPLoop) handleRequest(w net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			msg := fmt.Sprintf("Panic/Recovered: Error: %s", err)
			re.log.Critical(msg)
			debug.PrintStack()

			if w != nil {
				re.OutError(w, GetError(ErrPanic), err)
			}
		}
	}()

	w.SetReadDeadline(time.Now().Add(time.Duration(READ_TIMEOUT * time.Minute)))

	buf := bufio.NewReader(w)
	for {
		line, err := buf.ReadBytes(repr.NEWLINE_SEPARATOR_BYTE)

		if err != nil {
			if err == io.EOF {
				re.log.Info("connections closed by %v", w.RemoteAddr())
				return
			}

			re.OutError(w, GetError(ErrReadError), err)

			if strings.Contains(err.Error(), "i/o timeout") {
				return
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		cmd, errcode := re.getCommand(line)
		if errcode != nil {
			re.OutError(w, errcode, "")
			continue
		}
		args, errcode, err := cmd.ParseCommand(line)
		if errcode != nil {
			re.OutError(w, errcode, err)
			continue
		}
		if err != nil {
			re.OutError(w, GetError(ErrOther), err)
			continue
		}

		args.w = w
		args.loop = re
		resp := cmd.Do(args)

		// closed
		if resp.w == nil {
			return
		}

		if resp.errcode != nil {
			re.OutError(w, resp.errcode, resp.auxError)
			continue
		}
		if resp.auxError != nil {
			re.OutError(w, GetError(ErrOther), resp.auxError)
			continue
		}

		fmt := "json"
		if _, ok := args.args["format"]; ok {
			fmt = args.args["format"].(string)
		}
		if resp.response != nil {
			re.OutOk(resp.w, resp.response, fmt)
			if resp.needTerminator {
				w.Write(TERMINATOR)
			}
		}

	}

}

func (re *TCPLoop) Listen() {
	shuts := re.shutdown.Listen()

	re.Accepter(re.Socket)

	for {
		select {
		case conn, ok := <-re.AcceptChan:
			if !ok {
				return
			}
			re.log.Info("Accepted tcp conn %v", conn.RemoteAddr())
			re.workChan <- conn

		case <-shuts.Ch:
			re.log.Warning("TCP Client: Shutdown gotten .. stopping incoming connections")
			return
		}

		if re.AcceptChan == nil {
			return
		}
	}
}

func (re *TCPLoop) Start() error {
	re.log.Notice("Starting reader TCP server on %s", re.Conf.Listen)

	var err error
	if re.Conf.Logfile == "stderr" {
		re.OutLog = os.Stderr
	} else if re.Conf.Logfile == "stdout" {
		re.OutLog = os.Stdout
	} else if re.Conf.Logfile != "none" {
		re.OutLog, err = os.OpenFile(re.Conf.Logfile, os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			golog.Panicf("Could not open Logfile %s, setting to stdout", re.Conf.Logfile)
			re.OutLog = os.Stdout
		}
	}

	// certs if needed
	if len(re.Conf.TLSKeyPath) > 0 && len(re.Conf.TLSCertPath) > 0 {
		cer, err := tls.LoadX509KeyPair(re.Conf.TLSCertPath, re.Conf.TLSKeyPath)
		if err != nil {
			golog.Panicf("Could not start https server: %v", err)
			return err
		}
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
		re.Socket, err = tls.Listen("tcp", re.Conf.Listen, config)
		if err != nil {
			return fmt.Errorf("Could not make tcp socket: %s", err)
		}

	} else {
		tcpAddr, err := net.ResolveTCPAddr("tcp", re.Conf.Listen)
		if err != nil {
			return fmt.Errorf("Error resolving: %s", err)
		}
		re.Socket, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return fmt.Errorf("Could not make tcp socket: %s", err)
		}
	}

	re.started = true

	// fire up the workers
	for i := 0; i < re.numWorkers; i++ {
		go re.goWorkers()
	}

	// listen away
	go re.Listen()
	shuts := re.shutdown.Listen()
	for {
		select {
		case _, more := <-shuts.Ch:
			// already done
			if !more {
				return nil
			}
			re.Socket.Close()
			golog.Print("Shutdown of API TCP server...")
			return nil
		}
	}
}
