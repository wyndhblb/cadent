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

package main

// a little "blaster of reads" to stress test the APIs

import (
	"bufio"

	"bytes"
	"cadent/server/schemas/api"
	"cadent/server/schemas/metrics"
	"cadent/server/utils"
	grcp "cadent/server/writers/rpc/client"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/tinylib/msgp/msgp"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/transport/zipkin"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var tNow time.Time
var startTime int64
var requestsSent int64
var hitStats map[string]int64
var mu sync.RWMutex
var socketList []net.Conn
var tracer opentracing.Tracer

func init() {
	rand.Seed(time.Now().Unix())
	tNow = time.Now()
	startTime = tNow.UnixNano()
	hitStats = make(map[string]int64)
	socketList = make([]net.Conn, 0)
}

func AddHit(code string) {
	mu.Lock()
	defer mu.Unlock()
	if h, ok := hitStats[code]; ok {
		atomic.AddInt64(&h, 1)
		hitStats[code] = h
	} else {
		hitStats[code] = 1
	}
}

// need to up this guy otherwise we quickly run out of sockets
func setUlimits() {

	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println("[System] Error Getting Rlimit: ", err)
	}
	fmt.Println("[System] Current Rlimit: ", rLimit)

	rLimit.Max = 999999
	rLimit.Cur = 999999
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println("[System] Error Setting Rlimit: ", err)
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println("[System] Error Getting Rlimit:  ", err)
	}
	fmt.Println("[System] Final Rlimit Final: ", rLimit)
}

func StatTick(printStats bool) {
	for {
		s_delta := time.Now().UnixNano() - startTime
		tSec := s_delta / 1000000000
		rate := (float64)(requestsSent) / float64(tSec)
		if printStats {
			log.Printf("Sent %d requests in %ds - rate: %0.2f req/s", requestsSent, tSec, rate)
			mu.RLock()
			for k, v := range hitStats {
				log.Printf("%v: %v", k, v)
			}
			mu.RUnlock()
		}
		tickSleep, _ := time.ParseDuration("5s")
		time.Sleep(tickSleep)
	}
}

func SetTracer(zserver string) {
	if len(zserver) == 0 {
		tracer = opentracing.GlobalTracer()
		return
	}

	// Jaeger tracer can be initialized with a transport that will
	// report tracing Spans to a Zipkin backend
	transport, err := zipkin.NewHTTPTransport(
		zserver,
		zipkin.HTTPBatchSize(1),
		zipkin.HTTPLogger(jaeger.StdLogger),
	)
	if err != nil {
		panic(fmt.Errorf("Cannot initialize HTTP transport for tracing: %v", err))
	}

	// 0 means "sample all"
	sampler := jaeger.NewConstSampler(true)

	nm := "cadent-readblast"

	// create Jaeger tracer
	tracer, _ = jaeger.NewTracer(
		nm,
		sampler,
		jaeger.NewRemoteReporter(transport),
	)
}

func SendHttpRequest(server string) {
	c := &http.Client{Transport: &nethttp.Transport{}}
	nm := "cadent-readblast"

	span := tracer.StartSpan(nm)
	span.SetTag(string(ext.Component), nm)
	span.SetTag(string(ext.SpanKind), "client")
	defer span.Finish()

	ctx := opentracing.ContextWithSpan(context.Background(), span)

	req, err := http.NewRequest(
		"GET",
		server,
		nil,
	)
	if err != nil {
		panic(err)
	}
	req = req.WithContext(ctx)

	// wrap the request in nethttp.TraceRequest
	req, ht := nethttp.TraceRequest(tracer, req)
	defer ht.Finish()

	resp, err := c.Do(req)
	if err != nil {
		span.SetTag("error", err)
		return
	}
	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		span.SetTag("error", err)
		AddHit("BODYFAIL")
	}
	AddHit(strconv.Itoa(resp.StatusCode))
}

func RunHttpHitter(server, metrics string, from int, format string) {
	mets := strings.Split(metrics, ",")
	for {
		for _, m := range mets {
			URL := server + fmt.Sprintf("?target=%s&from=%dh&format=%s", m, from, format)
			SendHttpRequest(URL)
			atomic.AddInt64(&requestsSent, 1)
		}
	}
}

func getSockets(hostport string, sockets int) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", hostport)
	if err != nil {
		panic(err)
	}
	for i := 0; i < sockets; i++ {
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			panic(err)
		}
		socketList = append(socketList, conn)
	}
}

func RunTCPHitter(idx int, metric string, from int, format string) {

	head := make([]byte, 4)
	chunk := make([]byte, 4096*2)
	readBinary := func(con net.Conn) []byte {
		var llen uint32

		n, _ := con.Read(head)
		if n != 4 {
			fmt.Println("could not read 4 byte length")
			return nil
		}
		//fmt.Println(head)
		llen = binary.BigEndian.Uint32(head)
		//fmt.Println("LL", llen)

		if llen <= 0 {
			fmt.Println("No bytes left to read")
			return nil
		}
		// 2 extra bytes for the \r\n terminator
		bs := utils.GetBytesBuffer()
		defer utils.PutBytesBuffer(bs)
		curL := uint32(0)
		for curL < llen+2 {
			have, err := con.Read(chunk)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				return nil
			}
			curL += uint32(have)
			bs.Write(chunk[:have])
		}

		if llen+2 != uint32(curL) {
			fmt.Printf("Not enough bytes read... got %d wanted %d\n", curL, llen+2)
			return nil
		}
		//fmt.Println("GO THE GOODS", len(out))
		//fmt.Println("HEAD", out[:10])
		return bs.Bytes()
	}

	readJson := func(buf *bufio.Reader) []byte {
		outBuf := bytes.NewBuffer(nil)
		for {
			bs, err := buf.ReadBytes('\r')
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println(err)
				break
			}
			p, _ := buf.Peek(1)
			if bytes.EqualFold(p, []byte("\n")) {
				outBuf.Write(bs)
				break
			}
			outBuf.Write(bs)

		}
		return outBuf.Bytes()
	}

	conn := socketList[idx]
	for {

		for _, m := range strings.Split(metric, ",") {
			str := fmt.Sprintf("GETM %s %dh now 1 %s\n", m, from, format)

			_, err := conn.Write([]byte(str))
			if err != nil {
				panic(err.Error())
			}
			atomic.AddInt64(&requestsSent, 1)

			mObj := new(metrics.RawRenderItems)
			var b []byte
			switch format {
			case "msgpack":
				b = readBinary(conn)
				e := msgp.Decode(bytes.NewBuffer(b), mObj)
				if e != nil {
					fmt.Println(e)
				}

			case "protobuf":
				b = readBinary(conn)
			default:
				b = readJson(bufio.NewReader(conn))
			}

			//fmt.Print(len(b), " ")

			if err != nil {
				fmt.Println("ERROR: ", err)
				AddHit("BODYFAIL")
			} else {
				AddHit("200")
			}
		}
	}
}

func RunGRPCHitter(server, metrics string, from int) {
	mets := strings.Split(metrics, ",")

	cli, err := grcp.New(server, nil, "", tracer)
	if err != nil {
		panic(err)
	}
	cli.Tracer = tracer

	f := time.Now().Add(time.Duration(int64(time.Hour) * int64(from)))
	for {
		for _, m := range mets {
			span := tracer.StartSpan("cadent-grpc-client")
			span.SetTag(string(ext.Component), "client")

			ctx := opentracing.ContextWithSpan(context.Background(), span)
			m := &api.MetricQuery{
				Target: m,
				Start:  int64(f.Unix()),
				End:    time.Now().Unix(),
				Step:   1,
			}
			_, err := cli.GetMetrics(ctx, m)
			if err != nil {
				fmt.Println("ERROR", err)
				AddHit("BODYFAIL")

			} else {
				atomic.AddInt64(&requestsSent, 1)
				AddHit("200")
			}
			span.Finish()
		}
	}
}

func main() {
	setUlimits()

	server := flag.String("server", "http://127.0.0.1:8083/graphite/rawrender", `server to blast (grpc://127.0.0.1:9089, tcp://127.0.0.1:8084, http://127.0.0.1:8084)`)
	metric := flag.String("metric", "", `list of metrics to read 'my.metric,this.metric,that.metric' (required)`)
	concur := flag.Int("forks", 2, "number of concurrent senders")
	from := flag.Int("from", -1, "number of hours back to get data from")
	format := flag.String("format", "json", "json, msgpack, protobuf")
	toPrint := flag.Bool("tick", true, "print stats")
	zServer := flag.String("zipkin", "", "Tracing to Zipkin (i.e. http://localhost:9411/api/v1/spans)")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if len(*server) == 0 {
		panic("server is required")
	}

	SetTracer(*zServer)

	if len(*metric) == 0 {
		panic("metric is required")
	}

	us, err := url.Parse(*server)
	if err != nil {
		panic(err)
	}

	switch us.Scheme {
	case "http", "https":

		for i := 0; i < *concur; i++ {
			go RunHttpHitter(*server, *metric, *from, *format)
		}
	case "tcp":
		getSockets(us.Host, *concur)
		for i := 0; i < *concur; i++ {
			go RunTCPHitter(i, *metric, *from, *format)
		}
	case "grpc":
		for i := 0; i < *concur; i++ {
			go RunGRPCHitter(us.Host, *metric, *from)
		}
	}

	StatTick(*toPrint)
}
