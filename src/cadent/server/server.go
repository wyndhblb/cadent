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
   Pretty much the main work horse

   ties the lot together

   Listener -> [Accumulator] -> [Prereg] -> Backend -> Hasher -> [Replicator] -> NetPool -> Out

   Optional items are in the `[]`

   If we are using accumulators to basically "be" a graphite aggregator or statsd aggregator the flow is a bit
   Different

   Listener
   	-> PreReg (for rejection processing)
   	-> Accumulator
   		-> `Flush`
   		-> PreReg (again as the accumulator can generate "more" keys, statsd timers for instance)
   			-> Backend
   			-> Hasher
   				-> [Replicator]
   				-> Netpool
   				-> Out

*/

package cadent

import (
	"cadent/server/accumulator"
	"cadent/server/broadcast"
	"cadent/server/config"
	"cadent/server/dispatch"
	"cadent/server/netpool"
	"cadent/server/prereg"
	"cadent/server/splitter"
	"cadent/server/stats"
	sdown "cadent/server/utils/shutdown"
	"encoding/json"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	DEFAULT_WORKERS                    = int64(8)
	DEFAULT_OUTPUT_WORKERS             = int64(32)
	DEFAULT_INTERNAL_QUEUE_LENGTH      = 81920
	DEFAULT_NUM_STATS                  = 256
	DEFAULT_SENDING_CONNECTIONS_METHOD = "bufferedpool"
	DEFAULT_SPLITTER_TIMEOUT           = 5000 * time.Millisecond
	DEFAULT_WRITE_TIMEOUT              = 1000 * time.Millisecond
	DEFAULT_BACKPRESSURE_SLEEP         = 1000 * time.Millisecond
	DEFAULT_READ_BUFFER_SIZE           = 4096
)

type outMessageType int8

const (
	normalMessage outMessageType = 1 << iota // notmal message
)

type OutputMessage struct {
	mType     outMessageType
	outserver string
	param     []byte
	client    Client
	server    *Server
}

var log = logging.MustGetLogger("server")

/****************** SERVERS *********************/

// helper object for the json info about a single "key"
// basically to see "what server" a key will end up going to
type ServerHashCheck struct {
	ToServers []string `json:"to_servers"`
	HashKey   string   `json:"hash_key"`
	HashValue []uint32 `json:"hash_value"`
}

// a server set of stats
type Server struct {
	Name      string
	ListenURL *url.URL

	ValidLineCount       stats.StatCount
	WorkerValidLineCount stats.StatCount
	InvalidLineCount     stats.StatCount
	SuccessSendCount     stats.StatCount
	FailSendCount        stats.StatCount
	UnsendableSendCount  stats.StatCount
	UnknownSendCount     stats.StatCount
	AllLinesCount        stats.StatCount
	RejectedLinesCount   stats.StatCount
	RedirectedLinesCount stats.StatCount
	BytesWrittenCount    stats.StatCount
	BytesReadCount       stats.StatCount
	NumStats             uint
	ShowStats            bool

	// our bound connection if TCP or UnixSocket
	Connection net.Listener     // unix socket (no SO_REUSEPORT allowed)
	TCPConns   []net.Listener   // TCP SO_REUSEPORT
	UDPConns   []net.PacketConn // UDP SO_REUSEPORT

	//if our "buffered" bits exceded this, we're basically out of ram
	// so we "pause" until we can do something
	ClientReadBufferSize int64            //for net read buffers
	MaxReadBufferSize    int64            // the biggest we can make this buffer before "failing"
	CurrentReadBufferRam *stats.AtomicInt // the amount of buffer we're on

	// timeouts for tuning
	WriteTimeout    time.Duration // time out when sending lines
	SplitterTimeout time.Duration // timeout for work queue items

	//Hasher objects (can have multiple for replication of data)
	Hashers []*ConstHasher

	// we can use a "pool" of connections, or single connections per line
	// performance will be depending on the system and work load tcp vs udp, etc
	// "bufferedpool" or "pool" or "single"
	// default is pool
	SendingConnectionMethod string
	//number of connections in the NetPool
	NetPoolConnections int
	//if using the buffer pool, this is the buffer size
	WriteBufferPoolSize int

	//number of replicas to fire data to (i.e. dupes)
	Replicas int

	//if true, we DO NOT really send anything anywhere
	// useful for "writer only" backends not hashers proxies
	DevNullOut bool

	//pool the connections to the outgoing servers
	poolmu  *sync.Mutex //when we make a new pool need to lock the hash below
	Outpool map[string]netpool.NetpoolInterface

	ticker time.Duration

	//input queue for incoming lines
	InputQueue     chan splitter.SplitItem
	ProcessedQueue chan splitter.SplitItem

	//workers and ques sizes
	WorkQueue   chan *OutputMessage
	WorkerHold  chan int64
	InWorkQueue *stats.AtomicInt
	Workers     int64
	OutWorkers  int64

	//Worker Breaker
	// workBreaker *breaker.Breaker

	//the Splitter type to determine the keys to hash on
	SplitterTypeString string
	SplitterConfig     map[string]interface{}
	SplitterProcessor  splitter.Splitter

	// Prereg filters to push to other backends or drop
	PreRegFilter *prereg.PreReg

	//allow us to push backpressure on TCP sockets if we need to
	backPressure      chan bool
	_backPressureOn   bool // makes sure we dont' fire a billion things in the channel
	backPressureSleep time.Duration
	backPressureLock  sync.Mutex

	//trap some signals yo
	StopTicker chan bool
	ShutDown   *broadcast.Broadcaster

	//the push function (polling, direct, etc)
	Writer           OutMessageWriter
	OutputDispatcher *dispatch.Dispatch

	//uptime
	StartTime time.Time

	Stats *stats.HashServerStats

	log *logging.Logger
}

func (server *Server) InitCounters() {
	//pref := fmt.Sprintf("%p", server)
	server.Stats = new(stats.HashServerStats)
	server.Stats.Mu = new(sync.RWMutex)

	server.ValidLineCount = stats.NewStatCount(server.Name + "-ValidLineCount")
	server.WorkerValidLineCount = stats.NewStatCount(server.Name + "-WorkerValidLineCount")
	server.InvalidLineCount = stats.NewStatCount(server.Name + "-InvalidLineCount")
	server.SuccessSendCount = stats.NewStatCount(server.Name + "-SuccessSendCount")
	server.FailSendCount = stats.NewStatCount(server.Name + "-FailSendCount")
	server.UnsendableSendCount = stats.NewStatCount(server.Name + "-UnsendableSendCount")
	server.UnknownSendCount = stats.NewStatCount(server.Name + "-UnknownSendCount")
	server.AllLinesCount = stats.NewStatCount(server.Name + "-AllLinesCount")
	server.RejectedLinesCount = stats.NewStatCount(server.Name + "-RejectedLinesCount")
	server.RedirectedLinesCount = stats.NewStatCount(server.Name + "-RedirectedLinesCount")
	server.BytesWrittenCount = stats.NewStatCount(server.Name + "-BytesWrittenCount")
	server.BytesReadCount = stats.NewStatCount(server.Name + "-BytesReadCount")

	server.CurrentReadBufferRam = stats.NewAtomic(server.Name + "-CurrentReadBufferRam")
	server.InWorkQueue = stats.NewAtomic(server.Name + "-InWorkQueue")

}

func (server *Server) AddToCurrentTotalBufferSize(length int64) int64 {
	return server.CurrentReadBufferRam.Add(length)
}

func (server *Server) GetStats() (stats *stats.HashServerStats) {
	return server.Stats
}

func (server *Server) TrapExit() {
	//trap kills to flush queues and close connections
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func(ins *Server) {
		s := <-sc
		ins.log.Warning("Caught %s: Closing Server `%s` out before quit ", s, ins.Name)

		ins.StopServer()

		signal.Stop(sc)
		close(sc)

		// re-raise it
		//process, _ := os.FindProcess(os.Getpid())
		//process.Signal(s)
		return
	}(server)
}

func (server *Server) StopServer() {
	sdown.AddToShutdown()
	defer sdown.ReleaseFromShutdown()
	server.log.Warning("Stoping Server")

	//broadcast die
	server.ShutDown.Close()

	go func() { close(server.StopTicker) }()

	// need to clen up the socket here otherwise it may not get cleaned
	if server.ListenURL != nil && server.ListenURL.Scheme == "unix" {
		os.Remove("/" + server.ListenURL.Host + server.ListenURL.Path)
	}

	server.log.Warning("Shutting down health checks for `%s`", server.Name)
	for _, hasher := range server.Hashers {
		hasher.ServerPool.Stop()
	}

	//shut this guy down too
	if server.PreRegFilter != nil {
		server.log.Warning("Shutting down Prereg accumulator for `%s`", server.Name)
		//tick := time.NewTimer(2 * time.Second)
		//did := make(chan bool)
		//go func() {
		server.PreRegFilter.Stop()
		//did <- true
		//return
		//}()

		/*select {
		case <-tick.C:
			close(did)
			break
		case <-did:
			break
		}*/

	}
	if server.OutputDispatcher != nil {
		server.OutputDispatcher.Shutdown()
	}
	// bleed the pools
	if server.Outpool != nil {
		for k, outp := range server.Outpool {
			server.log.Warning("Bleeding buffer pool %s", k)
			server.log.Warning("Waiting 2 seconds for pools to empty")
			tick := time.NewTimer(2 * time.Second)
			did := make(chan bool)
			go func() {
				outp.DestroyAll()
				did <- true
				return
			}()
			select {
			case <-tick.C:
				close(did)
				break
			case <-did:
				break
			}

		}
	}

	close(server.ProcessedQueue)
	close(server.WorkQueue)
	// bleed the queues if things get stuck
	for {
		for i := 0; i < len(server.InputQueue); i++ {
			_ = <-server.InputQueue
		}
		// need to keep this open as it's inside other "clients"
		if len(server.InputQueue) == 0 {
			break
		}
	}

	server.log.Warning("Server Terminated .... ")

}

// set the "push" function we are using "pool" or "single"
func (server *Server) SetWriter() OutMessageWriter {
	switch server.SendingConnectionMethod {
	case "single":
		server.Writer = new(SingleWriter)
	default:
		server.Writer = new(PoolWriter)
	}
	return server.Writer
}

func (server *Server) SetSplitterProcessor() (splitter.Splitter, error) {
	gots, err := splitter.NewSplitterItem(server.SplitterTypeString, server.SplitterConfig)
	server.SplitterProcessor = gots
	return server.SplitterProcessor, err

}

func (server *Server) BackPressure() {
	if server._backPressureOn {
		return
	}
	server.backPressure <- server.NeedBackPressure()
	server._backPressureOn = true
}

func (server *Server) NeedBackPressure() bool {
	return server.CurrentReadBufferRam.Get() > server.MaxReadBufferSize || len(server.WorkQueue) == (int)(DEFAULT_INTERNAL_QUEUE_LENGTH)
}

//spins up the queue of go routines to handle outgoing
// each one is wrapped in a breaker
func (server *Server) WorkerOutput() {
	shuts := server.ShutDown.Listen()

	for {
		select {
		case j := <-server.WorkQueue:

			server.OutputDispatcher.JobsQueue() <- &OutputDispatchJob{
				Message: j,
				Writer:  server.Writer,
			}

		/** Does not seem to really work w/o
		res := server.workBreaker.Run(func() error {
			err := server.Writer.Write(j)

			if err != nil {
				server.log.Error("%s", err)
			}
			return err
		})
		if res == breaker.ErrBreakerOpen {
			server.log.Warning("Circuit Breaker is ON")
		}
		*/
		case <-shuts.Ch:
			return

		}
	}
}

func (server *Server) SendtoOutputWorkers(spl splitter.SplitItem, out chan splitter.SplitItem) {
	//direct timer to void leaks (i.e. NOT time.After(...))

	timer := time.NewTimer(server.SplitterTimeout)
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("factory.splitter.process-time-ns"), time.Now())
	defer func() { server.WorkerHold <- -1 }()

	useChan := out

	//IF the item's origin is other, we have to use the "generic" output processor
	// as we've lost the originating socket channel
	if spl.Origin() == splitter.Other {
		useChan = server.ProcessedQueue
	}

	select {
	case useChan <- server.PushLineToBackend(spl):
		timer.Stop()

	case <-timer.C:
		timer.Stop()
		stats.StatsdClient.Incr("failed.splitter-timeout", 1)
		server.FailSendCount.Up(1)
		server.log.Warning("Timeout Queue len: %d, %s", len(server.WorkQueue), spl.Line())
	}
	return
}

//return the ServerHashCheck for a given key (more a utility debugger thing for
// the stats http server)
func (server *Server) HasherCheck(key []byte) ServerHashCheck {

	var out_check ServerHashCheck
	out_check.HashKey = string(key)

	for _, hasher := range server.Hashers {
		// may have replicas inside the pool too that we need to deal with
		servs, err := hasher.GetN(key, server.Replicas)
		if err == nil {
			for _, useme := range servs {
				out_check.ToServers = append(out_check.ToServers, useme)
				out_check.HashValue = append(out_check.HashValue, hasher.Hasher.GetHasherValue(out_check.HashKey))
			}
		}
	}
	return out_check
}

// the "main" hash chooser for a give line, the attaches it to a sender queue
func (server *Server) PushLineToBackend(spl splitter.SplitItem) splitter.SplitItem {

	// if the server(s) are /dev/null we just skip this stuff
	if server.DevNullOut {
		return spl
	}

	//replicate the data across our Lists
	for idx, hasher := range server.Hashers {

		// may have replicas inside the pool too that we need to deal with
		servs, err := hasher.GetN(spl.Key(), server.Replicas)
		if err == nil {
			for nidx, useme := range servs {
				// just log the valid lines "once" total ends stats are WorkerValidLineCount
				if idx == 0 && nidx == 0 {
					// server.ValidLineCount.Up(1) counted on incoming clients
					stats.StatsdClient.Incr("success.valid-lines", 1)
				}
				stats.StatsdClient.Incr("success.valid-lines-sent-to-workers", 1)
				server.WorkerValidLineCount.Up(1)

				sendOut := &OutputMessage{
					mType:     normalMessage,
					outserver: useme,
					server:    server,
					param:     spl.Line(),
				}

				server.WorkQueue <- sendOut
				server.WorkerHold <- 1

			}
			// finally put the splititem back into the pool
			splitter.PutSplitItem(spl)
		} else {

			stats.StatsdClient.Incr("failed.invalid-hash-server", 1)
			server.UnsendableSendCount.Up(1)
		}
	}
	return spl

}

func (server *Server) ResetTickers() {
	server.ValidLineCount.ResetTick()
	server.WorkerValidLineCount.ResetTick()
	server.InvalidLineCount.ResetTick()
	server.SuccessSendCount.ResetTick()
	server.FailSendCount.ResetTick()
	server.UnsendableSendCount.ResetTick()
	server.UnknownSendCount.ResetTick()
	server.AllLinesCount.ResetTick()
	server.RedirectedLinesCount.ResetTick()
	server.RejectedLinesCount.ResetTick()
	server.BytesWrittenCount.ResetTick()
	server.BytesReadCount.ResetTick()
}

func NewServer(cfg *config.HasherConfig) (server *Server, err error) {

	serv := new(Server)
	serv.Name = cfg.Name
	serv.InitCounters()
	serv.ListenURL = cfg.ListenURL
	serv.StartTime = time.Now()
	serv.WorkerHold = make(chan int64, 1024)

	serv.log = logging.MustGetLogger(fmt.Sprintf("server.%s", serv.Name))

	//log.New(os.Stdout, fmt.Sprintf("[Server: %s] ", serv.Name), log.Ldate|log.Ltime)
	if cfg.ListenURL != nil {
		serv.log.Notice("Binding server to %s", serv.ListenURL.String())
	} else {
		serv.log.Notice("Using as a Backend Only to %s", serv.Name)
	}

	//find the runner types
	serv.SplitterTypeString = cfg.MsgType
	serv.SplitterConfig = cfg.MsgConfig
	serv.Replicas = cfg.Replicas

	serv.ShutDown = broadcast.New(2)

	// serv.workBreaker = breaker.New(3, 1, 2*time.Second)

	serv.poolmu = new(sync.Mutex)

	serv.NumStats = DEFAULT_NUM_STATS

	serv.WriteTimeout = DEFAULT_WRITE_TIMEOUT
	if cfg.WriteTimeout != 0 {
		serv.WriteTimeout = cfg.WriteTimeout
	}

	serv.SplitterTimeout = DEFAULT_SPLITTER_TIMEOUT
	if cfg.SplitterTimeout != 0 {
		serv.SplitterTimeout = cfg.SplitterTimeout
	}

	serv.NetPoolConnections = cfg.MaxPoolConnections
	serv.WriteBufferPoolSize = cfg.MaxWritePoolBufferSize
	serv.SendingConnectionMethod = DEFAULT_SENDING_CONNECTIONS_METHOD

	serv.ClientReadBufferSize = cfg.ClientReadBufferSize
	if serv.ClientReadBufferSize <= 0 {
		serv.ClientReadBufferSize = DEFAULT_READ_BUFFER_SIZE
	}
	serv.MaxReadBufferSize = cfg.MaxReadBufferSize

	if serv.MaxReadBufferSize <= 0 {
		serv.MaxReadBufferSize = 1000 * serv.ClientReadBufferSize // reasonable default for max buff
	}

	serv.backPressureSleep = DEFAULT_BACKPRESSURE_SLEEP

	if len(cfg.SendingConnectionMethod) > 0 {
		serv.SendingConnectionMethod = cfg.SendingConnectionMethod
	}

	// if there's a PreReg assign to the server
	if cfg.PreRegFilter != nil {
		// the config should have checked this already, but just in case
		if cfg.ListenStr == "backend_only" {
			return nil, fmt.Errorf("Backend Only cannot have PreReg filters")
		}
		serv.PreRegFilter = cfg.PreRegFilter
	}

	serv.ticker = time.Duration(5) * time.Second

	serv.DevNullOut = false
	if cfg.DevNullOut {
		serv.DevNullOut = true
	}

	return serv, nil

}

func (server *Server) StartListen(url *url.URL) error {
	if server.ListenURL == nil {
		// just a backend, no connections

	} else if url.Scheme == "udp" {

		server.UDPConns = make([]net.PacketConn, server.Workers)
		// use multiple listeners for UDP, so need to have REUSE enabled
		for i := 0; i < int(server.Workers); i++ {
			_, err := net.ResolveUDPAddr(url.Scheme, url.Host)
			if err != nil {
				return fmt.Errorf("Error binding: %s", err)
			}

			//conn, err := net.ListenUDP(url.Scheme, udp_addr)
			// allow multiple connections over the same socket, and let the kernel
			// deal with delgating to which ever listener
			conn, err := GetReusePacketListenerWithBuffer(
				url.Scheme, url.Host, int(server.ClientReadBufferSize), 0,
			)
			if err != nil {
				return fmt.Errorf("Error binding: %s", err)
			}
			//err = SetUDPReuse(conn) // allows for multi listens on the same port!
			if err != nil {
				return fmt.Errorf("Error binding: %s", err)
			}
			server.UDPConns[i] = conn
		}

	} else if url.Scheme == "http" {
		//http is yet another "special" case, client HTTP does the hard work
	} else if url.Scheme == "tcp" {
		server.TCPConns = make([]net.Listener, server.Workers)
		// use multiple listeners for TCP, so need to have REUSE enabled
		for i := 0; i < int(server.Workers); i++ {
			_, err := net.ResolveTCPAddr(url.Scheme, url.Host)
			if err != nil {
				return fmt.Errorf("Error binding: %s", err)
			}

			// allow multiple connections over the same socket, and let the kernel
			// deal with delegating to which ever listener
			conn, err := GetReuseListenerWithBuffer(
				url.Scheme, url.Host, int(server.ClientReadBufferSize), 0,
			)
			if err != nil {
				return fmt.Errorf("Error binding: %s", err)
			}
			//err = SetUDPReuse(conn) // allows for multi listens on the same port!
			if err != nil {
				return fmt.Errorf("Error binding: %s", err)
			}
			server.TCPConns[i] = conn
		}
	} else {
		// unix socket
		var conn net.Listener
		var err error
		conn, err = net.Listen(url.Scheme, url.Host+url.Path)

		if err != nil {
			return fmt.Errorf("Error binding: %s", err)
		}
		server.Connection = conn
	}

	//the queue is consumed by the main output dispatcher
	server.WorkQueue = make(chan *OutputMessage, DEFAULT_INTERNAL_QUEUE_LENGTH)

	//input queue
	server.InputQueue = make(chan splitter.SplitItem, DEFAULT_INTERNAL_QUEUE_LENGTH)

	//input queue
	server.ProcessedQueue = make(chan splitter.SplitItem, DEFAULT_INTERNAL_QUEUE_LENGTH)
	return nil

}

func (server *Server) StatsTick() {

	elapsed := time.Since(server.StartTime)
	elapsedSec := float64(elapsed) / float64(time.Second)
	tStamp := time.Now().UnixNano()

	server.Stats.ValidLineCount = server.ValidLineCount.TotalCount.Get()
	server.Stats.WorkerValidLineCount = server.WorkerValidLineCount.TotalCount.Get()
	server.Stats.InvalidLineCount = server.InvalidLineCount.TotalCount.Get()
	server.Stats.SuccessSendCount = server.SuccessSendCount.TotalCount.Get()
	server.Stats.FailSendCount = server.FailSendCount.TotalCount.Get()
	server.Stats.UnsendableSendCount = server.UnsendableSendCount.TotalCount.Get()
	server.Stats.UnknownSendCount = server.UnknownSendCount.TotalCount.Get()
	server.Stats.AllLinesCount = server.AllLinesCount.TotalCount.Get()
	server.Stats.RedirectedLinesCount = server.RedirectedLinesCount.TotalCount.Get()
	server.Stats.RejectedLinesCount = server.RejectedLinesCount.TotalCount.Get()
	server.Stats.BytesReadCount = server.BytesReadCount.TotalCount.Get()
	server.Stats.BytesWrittenCount = server.BytesWrittenCount.TotalCount.Get()

	server.Stats.CurrentValidLineCount = server.ValidLineCount.TickCount.Get()
	server.Stats.CurrentWorkerValidLineCount = server.WorkerValidLineCount.TickCount.Get()
	server.Stats.CurrentInvalidLineCount = server.InvalidLineCount.TickCount.Get()
	server.Stats.CurrentSuccessSendCount = server.SuccessSendCount.TickCount.Get()
	server.Stats.CurrentFailSendCount = server.FailSendCount.TickCount.Get()
	server.Stats.CurrentUnknownSendCount = server.UnknownSendCount.TickCount.Get()
	server.Stats.CurrentAllLinesCount = server.AllLinesCount.TickCount.Get()
	server.Stats.CurrentRedirectedLinesCount = server.RedirectedLinesCount.TickCount.Get()
	server.Stats.CurrentRejectedLinesCount = server.RejectedLinesCount.TickCount.Get()
	server.Stats.CurrentBytesReadCount = server.BytesReadCount.TickCount.Get()
	server.Stats.CurrentBytesWrittenCount = server.BytesWrittenCount.TickCount.Get()

	server.Stats.ValidLineCountList = append(server.Stats.ValidLineCountList, server.ValidLineCount.TickCount.Get())
	server.Stats.WorkerValidLineCountList = append(server.Stats.WorkerValidLineCountList, server.WorkerValidLineCount.TickCount.Get())
	server.Stats.InvalidLineCountList = append(server.Stats.InvalidLineCountList, server.InvalidLineCount.TickCount.Get())
	server.Stats.SuccessSendCountList = append(server.Stats.SuccessSendCountList, server.SuccessSendCount.TickCount.Get())
	server.Stats.FailSendCountList = append(server.Stats.FailSendCountList, server.FailSendCount.TickCount.Get())
	server.Stats.UnknownSendCountList = append(server.Stats.UnknownSendCountList, server.UnknownSendCount.TickCount.Get())
	server.Stats.UnsendableSendCountList = append(server.Stats.UnsendableSendCountList, server.UnsendableSendCount.TickCount.Get())
	server.Stats.AllLinesCountList = append(server.Stats.AllLinesCountList, server.AllLinesCount.TickCount.Get())
	server.Stats.RejectedCountList = append(server.Stats.RejectedCountList, server.RejectedLinesCount.TickCount.Get())
	server.Stats.RedirectedCountList = append(server.Stats.RedirectedCountList, server.RedirectedLinesCount.TickCount.Get())
	server.Stats.BytesReadCountList = append(server.Stats.BytesReadCountList, server.BytesReadCount.TickCount.Get())
	server.Stats.BytesWrittenCountList = append(server.Stats.BytesWrittenCountList, server.BytesWrittenCount.TickCount.Get())
	// javascript resolution is ms .. not nanos
	server.Stats.TicksList = append(server.Stats.TicksList, int64(tStamp/int64(time.Millisecond)))
	server.Stats.GoRoutinesList = append(server.Stats.GoRoutinesList, runtime.NumGoroutine())

	if uint(len(server.Stats.ValidLineCountList)) > server.NumStats {
		server.Stats.ValidLineCountList = server.Stats.ValidLineCountList[1:server.NumStats]
		server.Stats.WorkerValidLineCountList = server.Stats.WorkerValidLineCountList[1:server.NumStats]
		server.Stats.InvalidLineCountList = server.Stats.InvalidLineCountList[1:server.NumStats]
		server.Stats.SuccessSendCountList = server.Stats.SuccessSendCountList[1:server.NumStats]
		server.Stats.FailSendCountList = server.Stats.FailSendCountList[1:server.NumStats]
		server.Stats.UnknownSendCountList = server.Stats.UnknownSendCountList[1:server.NumStats]
		server.Stats.UnsendableSendCountList = server.Stats.UnsendableSendCountList[1:server.NumStats]
		server.Stats.AllLinesCountList = server.Stats.AllLinesCountList[1:server.NumStats]
		server.Stats.RejectedCountList = server.Stats.RejectedCountList[1:server.NumStats]
		server.Stats.RedirectedCountList = server.Stats.RedirectedCountList[1:server.NumStats]
		server.Stats.TicksList = server.Stats.TicksList[1:server.NumStats]
		server.Stats.GoRoutinesList = server.Stats.GoRoutinesList[1:server.NumStats]
		server.Stats.BytesReadCountList = server.Stats.BytesReadCountList[1:server.NumStats]
		server.Stats.BytesWrittenCountList = server.Stats.BytesWrittenCountList[1:server.NumStats]
	}
	server.Stats.UpTimeSeconds = int64(elapsedSec)
	server.Stats.CurrentReadBufferSize = server.CurrentReadBufferRam.Get()
	server.Stats.MaxReadBufferSize = server.MaxReadBufferSize

	server.Stats.InputQueueSize = len(server.InputQueue)
	server.Stats.WorkQueueSize = len(server.WorkQueue)

	stats.StatsdClientSlow.GaugeAbsolute(fmt.Sprintf("%s.inputqueue.length", server.Name), int64(server.Stats.InputQueueSize))
	stats.StatsdClientSlow.GaugeAbsolute(fmt.Sprintf("%s.workqueue.length", server.Name), int64(server.Stats.WorkQueueSize))
	stats.StatsdClientSlow.GaugeAbsolute(fmt.Sprintf("%s.readbuffer.length", server.Name), int64(server.Stats.CurrentReadBufferSize))

	server.Stats.ValidLineCountPerSec = server.ValidLineCount.TotalRate(elapsed)
	server.Stats.WorkerValidLineCountPerSec = server.WorkerValidLineCount.TotalRate(elapsed)
	server.Stats.InvalidLineCountPerSec = server.InvalidLineCount.TotalRate(elapsed)
	server.Stats.SuccessSendCountPerSec = server.SuccessSendCount.TotalRate(elapsed)
	server.Stats.UnsendableSendCountPerSec = server.UnsendableSendCount.TotalRate(elapsed)
	server.Stats.UnknownSendCountPerSec = server.UnknownSendCount.TotalRate(elapsed)
	server.Stats.AllLinesCountPerSec = server.AllLinesCount.TotalRate(elapsed)
	server.Stats.RedirectedLinesCountPerSec = server.RedirectedLinesCount.TotalRate(elapsed)
	server.Stats.RejectedLinesCountPerSec = server.RejectedLinesCount.TotalRate(elapsed)
	server.Stats.BytesReadCountPerSec = server.BytesReadCount.TotalRate(elapsed)
	server.Stats.BytesWrittenCountPerSec = server.BytesWrittenCount.TotalRate(elapsed)

	if server.ListenURL == nil {
		server.Stats.Listening = "BACKEND-ONLY"
	} else {
		server.Stats.Listening = server.ListenURL.String()
	}

	//XXX TODO FIX ME to look like a multi service line, not one big puddle
	for idx, hasher := range server.Hashers {
		if idx == 0 {
			server.Stats.ServersUp = hasher.Members()
			server.Stats.ServersDown = hasher.DroppedServers()
			server.Stats.ServersChecks = hasher.CheckingServers()
		} else {
			server.Stats.ServersUp = append(server.Stats.ServersUp, hasher.Members()...)
			server.Stats.ServersDown = append(server.Stats.ServersDown, hasher.DroppedServers()...)
			server.Stats.ServersChecks = append(server.Stats.ServersChecks, hasher.CheckingServers()...)
		}
		//tick the cacher stats
		length, size, capacity, _ := hasher.Cache.Stats()
		stats.StatsdClientSlow.GaugeAbsolute(fmt.Sprintf("%s.lrucache.length", server.Name), int64(length))
		stats.StatsdClientSlow.GaugeAbsolute(fmt.Sprintf("%s.lrucache.size", server.Name), int64(size))
		stats.StatsdClientSlow.GaugeAbsolute(fmt.Sprintf("%s.lrucache.capacity", server.Name), int64(capacity))

	}

}

// dump some json data about the stats and server status
func (server *Server) StatsJsonString() string {
	resbytes, _ := json.Marshal(server.Stats)
	return string(resbytes)
}

// a little function to log out some collected stats
func (server *Server) tickDisplay() {

	ticker := time.NewTicker(server.ticker)
	for {
		select {
		case <-ticker.C:
			runtime.GC() // clean things each tick
			server.Stats.Mu.Lock()
			server.StatsTick()
			if server.ShowStats {
				server.log.Info("Server: ValidLineCount: %d", server.ValidLineCount.TotalCount.Get())
				server.log.Info("Server: WorkerValidLineCount: %d", server.WorkerValidLineCount.TotalCount.Get())
				server.log.Info("Server: InvalidLineCount: %d", server.InvalidLineCount.TotalCount.Get())
				server.log.Info("Server: SuccessSendCount: %d", server.SuccessSendCount.TotalCount.Get())
				server.log.Info("Server: FailSendCount: %d", server.FailSendCount.TotalCount.Get())
				server.log.Info("Server: UnsendableSendCount: %d", server.UnsendableSendCount.TotalCount.Get())
				server.log.Info("Server: UnknownSendCount: %d", server.UnknownSendCount.TotalCount.Get())
				server.log.Info("Server: AllLinesCount: %d", server.AllLinesCount.TotalCount.Get())
				server.log.Info("Server: BytesReadCount: %d", server.BytesReadCount.TotalCount.Get())
				server.log.Info("Server: BytesWrittenCount: %d", server.BytesWrittenCount.TotalCount.Get())
				server.log.Info("Server: GO Routines Running: %d", runtime.NumGoroutine())
				server.log.Info("Server: Current Buffer Size: %d/%d bytes", server.CurrentReadBufferRam.Get(), server.MaxReadBufferSize)
				server.log.Info("Server: Current Out Work Queue length: %d", len(server.WorkQueue))
				server.log.Info("Server: Current Input Queue length: %d", len(server.InputQueue))
				server.log.Info("-------")
				server.log.Info("Server Rate: Duration %ds", uint64(server.ticker/time.Second))
				server.log.Info("Server Rate: ValidLineCount: %.2f/s", server.ValidLineCount.Rate(server.ticker))
				server.log.Info("Server Rate: WorkerLineCount: %.2f/s", server.WorkerValidLineCount.Rate(server.ticker))
				server.log.Info("Server Rate: InvalidLineCount: %.2f/s", server.InvalidLineCount.Rate(server.ticker))
				server.log.Info("Server Rate: SuccessSendCount: %.2f/s", server.SuccessSendCount.Rate(server.ticker))
				server.log.Info("Server Rate: FailSendCount: %.2f/s", server.FailSendCount.Rate(server.ticker))
				server.log.Info("Server Rate: UnsendableSendCount: %.2f/s", server.UnsendableSendCount.Rate(server.ticker))
				server.log.Info("Server Rate: UnknownSendCount: %.2f/s", server.UnknownSendCount.Rate(server.ticker))
				server.log.Info("Server Rate: RejectedSendCount: %.2f/s", server.RejectedLinesCount.Rate(server.ticker))
				server.log.Info("Server Rate: RedirectedSendCount: %.2f/s", server.RedirectedLinesCount.Rate(server.ticker))
				server.log.Info("Server Rate: AllLinesCount: %.2f/s", server.AllLinesCount.Rate(server.ticker))
				server.log.Info("Server Rate: BytesReadCount: %.2f/s", server.BytesReadCount.Rate(server.ticker))
				server.log.Info("Server Rate: BytesWrittenCount: %.2f/s", server.BytesWrittenCount.Rate(server.ticker))
				server.log.Info("Server Send Method:: %s", server.SendingConnectionMethod)
				for idx, pool := range server.Outpool {
					server.log.Info("Free Connections in Pools [%s]: %d/%d", idx, pool.NumFree(), pool.GetMaxConnections())
				}
			}
			server.ResetTickers()
			server.Stats.Mu.Unlock()

		//runtime.GC()

		case <-server.StopTicker:
			server.log.Warning("Stopping stats ticker")
			return
		}
	}
}

// Fire up the http server for stats and healthchecks
// do this only if there is not a
func (server *Server) AddStatusHandlers(mux *http.ServeMux) {

	stats := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		fmt.Fprintf(w, server.StatsJsonString())
	}
	status := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		have_s := 0
		if server.DevNullOut {
			fmt.Fprintf(w, "ok (/dev/null)")
			return
		}
		for _, hasher := range server.Hashers {
			have_s += len(hasher.Members())
		}
		if have_s <= 0 {
			http.Error(w, "all servers down", http.StatusServiceUnavailable)
			return
		} else {
			fmt.Fprintf(w, "ok")
		}
	}

	// add a new hasher node to a server dynamically
	addnode := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")

		if server.DevNullOut {
			http.Error(w, "This is a /dev/null server .. cannot add", http.StatusBadRequest)
			return
		}
		r.ParseForm()
		server_str := strings.TrimSpace(r.Form.Get("server"))
		if len(server_str) == 0 {
			http.Error(w, "Invalid server name", http.StatusBadRequest)
			return
		}
		server_url, err := url.Parse(server_str)
		if err != nil {
			http.Error(w, "Not a valid server URL", http.StatusBadRequest)
			return
		}
		chk_server_str := strings.TrimSpace(r.Form.Get("check_server"))
		if len(chk_server_str) == 0 {
			chk_server_str = server_str
		}
		chk_server_url, err := url.Parse(chk_server_str)
		if err != nil {
			http.Error(w, "Not a valid Check server URL", http.StatusBadRequest)
			return
		}
		if chk_server_url.Scheme != "tcp" && chk_server_url.Scheme != "http" {
			http.Error(w, "Check server can only be TCP or HTTP", http.StatusBadRequest)
			return
		}
		// since we can have replicas .. need an index to add it to
		replica_str := r.Form.Get("replica")
		if len(replica_str) == 0 {
			replica_str = "0"
		}
		replica_int, err := strconv.Atoi(replica_str)
		if err != nil {
			http.Error(w, "Replica index is not an int", http.StatusBadRequest)
			return
		}
		if replica_int > len(server.Hashers)-1 {
			http.Error(w, "Replica index Too large", http.StatusBadRequest)
			return
		}
		if replica_int < 0 {
			http.Error(w, "Replica index Too small", http.StatusBadRequest)
			return
		}

		//we can also accept a hash key for that server
		hash_key_str := r.Form.Get("hashkey")
		if len(hash_key_str) == 0 {
			hash_key_str = server_str
		}

		server.Hashers[replica_int].AddServer(server_url, chk_server_url, hash_key_str)
		fmt.Fprintf(w, "Server "+server_str+" Added")
	}

	// add a new hasher node to a server dynamically
	purgenode := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		if server.DevNullOut {
			http.Error(w, "This is a /dev/null server .. cannot purge", http.StatusBadRequest)
			return
		}

		r.ParseForm()
		server_str := strings.TrimSpace(r.Form.Get("server"))
		if len(server_str) == 0 {
			http.Error(w, "Invalid server name", http.StatusBadRequest)
			return
		}
		server_url, err := url.Parse(server_str)
		if err != nil {
			http.Error(w, "Not a valid server URL", http.StatusBadRequest)
			return
		}

		// since we can have replicas .. need an index to add it to
		replica_str := r.Form.Get("replica")
		if len(replica_str) == 0 {
			replica_str = "0"
		}
		replica_int, err := strconv.Atoi(replica_str)
		if err != nil {
			http.Error(w, "Replica index is not an int", http.StatusBadRequest)
			return
		}
		if replica_int > len(server.Hashers)-1 {
			http.Error(w, "Replica index Too large", http.StatusBadRequest)
			return
		}
		if replica_int < 0 {
			http.Error(w, "Replica index Too small", http.StatusBadRequest)
			return
		}

		server.Hashers[replica_int].PurgeServer(server_url)

		// we need to CLOSE and thing in the Outpool
		if serv, ok := server.Outpool[server_str]; ok {
			serv.DestroyAll()
			delete(server.Outpool, server_str)
		}

		fmt.Fprintf(w, "Server "+server_str+" Purged")
	}

	// accumulator pokers
	if server.PreRegFilter == nil || server.PreRegFilter.Accumulator == nil {
		// "nothing to see here"
		nullacc := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
			http.Error(w, "No accumulators defined", http.StatusNotImplemented)
			return
		}
		http.HandleFunc(fmt.Sprintf("/%s/accumulator", server.Name), nullacc)
	} else {
		stats := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
			stats, err := json.Marshal(server.PreRegFilter.Accumulator.CurrentStats())
			if err != nil {
				http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, string(stats))
			return
		}
		http.HandleFunc(fmt.Sprintf("/%s/accumulator", server.Name), stats)
	}

	//stats and status
	mux.HandleFunc(fmt.Sprintf("/%s", server.Name), stats)
	mux.HandleFunc(fmt.Sprintf("/%s/ops/status", server.Name), status)
	mux.HandleFunc(fmt.Sprintf("/%s/ping", server.Name), status)
	mux.HandleFunc(fmt.Sprintf("/%s/ops/status/", server.Name), status)
	mux.HandleFunc(fmt.Sprintf("/%s/status", server.Name), status)
	mux.HandleFunc(fmt.Sprintf("/%s/stats/", server.Name), stats)
	mux.HandleFunc(fmt.Sprintf("/%s/stats", server.Name), stats)

	//admin like functions to add and remove servers to a hashring
	mux.HandleFunc(fmt.Sprintf("/%s/addserver", server.Name), addnode)
	mux.HandleFunc(fmt.Sprintf("/%s/purgeserver", server.Name), purgenode)
}

// ProcessSplitItem takes a split item and processes it, all clients need to call this to actually "do" something
// once things are split from the incoming
func (server *Server) ProcessSplitItem(splitem splitter.SplitItem, out_queue chan splitter.SplitItem) error {

	// based on the Key we get re may need to redirect this to another backend
	// due to the PreReg items
	// so you may ask why Here and not before the InputQueue
	// 1. back pressure
	// 2. input queue buffering
	// 3. Incoming lines gets sucked in faster before (and "return ok") quickly

	// if the split item has "already been parsed" we DO NOT send it here (AccumulatedParsed)
	// instead we send it to the Worker output queues
	if splitem == nil {
		return nil
	}

	// nothing to see here move along
	if server.PreRegFilter == nil {
		//log.Notice("Round Two Item: %s", splitem.Line())
		server.SendtoOutputWorkers(splitem, out_queue)
		return nil
	}

	// we need to reg the incoming keys for re-direction or rejection
	// match on the KEY not the entire string
	useBackend, reject, _ := server.PreRegFilter.FirstMatchBackend(splitem.Key())
	if reject {
		stats.StatsdClient.Incr(fmt.Sprintf("prereg.backend.reject.%s", useBackend), 1)
		server.RejectedLinesCount.Up(1)
		return nil
	}
	accumulate := server.PreRegFilter.Accumulator != nil
	//server.log.Notice("Input %s: Accumulate: %v OutBack: %s : reject: %v %s ", server.Name, accumulate, use_backend, reject, splitem.Line())

	//server.log.Notice("Input %s: %s FROM: %s:: doacc: %v", server.Name, splitem.Line(), splitem.accumulate)
	if accumulate {
		//log.Notice("Round One Item: %s", splitem.Line())
		//log.Debug("Acc:%v Line: %s Phase: %s", server.PreRegFilter.Accumulator.Name, splitem.Line())
		stats.StatsdClient.Incr(fmt.Sprintf("prereg.accumulated.%s", server.Name), 1)
		err := server.PreRegFilter.Accumulator.ProcessSplitItem(splitem)

		// put item back into pool
		splitter.PutSplitItem(splitem)

		if err != nil {
			log.Warning("Could not parse in accumulator Acc:%s Err: %s", server.PreRegFilter.Accumulator.Name, err)
		}
		return err
	}

	//if server.PreRegFilter
	if useBackend != server.Name && accumulate {
		// redirect to another input queue
		stats.StatsdClient.Incr(fmt.Sprintf("prereg.backend.redirect.%s", useBackend), 1)
		server.RedirectedLinesCount.Up(1)
		// send to different backend to "repeat" this process
		// this time we need to dis-avow the fact it came from a socket, as it's no longer pinned to the same
		// socket it came from
		log.Debug("Acc:%v UseBack: %v FromBack: %v Line: %s ", server.PreRegFilter.Accumulator.Name, useBackend, server.Name, splitem.Line())
		splitem.SetOrigin(splitter.Other)
		err := ServerBackends.Send(useBackend, splitem)
		if err != nil {
			server.log.Error("backend send error: %v", err)
		}
	} else {
		// otherwise just farm out as normal
		server.SendtoOutputWorkers(splitem, out_queue)
	}

	return nil

}

// accept incoming TCP connections and push them into the
// a connection channel
func (server *Server) Accepter(listen net.Listener) (<-chan net.Conn, error) {

	conns := make(chan net.Conn, server.Workers)
	go func() {
		defer func() {
			if listen != nil {
				listen.Close()
			}
			if server.ListenURL != nil && server.ListenURL.Scheme == "unix" {
				os.Remove("/" + server.ListenURL.Host + server.ListenURL.Path)
			}
		}()

		shuts := server.ShutDown.Listen()
		defer close(conns)

		statConnFailed := fmt.Sprintf("%s.tcp.incoming.failed.connections", server.Name)
		statConnIncoming := fmt.Sprintf("%s.tcp.incoming.connections", server.Name)

		for {
			select {
			case s := <-server.backPressure:
				if s {
					server.backPressureLock.Lock()
					server.log.Warning("Backpressure triggered pausing connections for : %s", server.backPressureSleep)
					time.Sleep(server.backPressureSleep)
					server.backPressureLock.Unlock()
				}
			case <-shuts.Ch:
				server.log.Warning("TCP Listener: Shutdown gotten .. stopping incoming connections")
				listen.Close()
				shuts.Close()
				return
			default:

			}

			conn, err := listen.Accept()
			if err != nil {
				stats.StatsdClient.Incr(statConnFailed, 1)
				server.log.Warning("Error Accecption Connection: %s", err)
				continue
			}
			stats.StatsdClient.Incr(statConnIncoming, 1)
			//server.log.Debug("Accepted connection from %s", conn.RemoteAddr())

			conns <- conn

		}
	}()
	return conns, nil
}

func (server *Server) startTCPServer(hashers *[]*ConstHasher, done chan Client) {

	// for TCP clients (and unix sockets),
	// since we basically create a client on each connect each time (unlike the UDP case)
	// and we want only one processing Q, set up this "queue" here
	// the main Server input Queue should be larger then the UDP case as we can have many TCP clients
	// and only one UDP client

	// tells the Acceptor to "sleep" incase we need to apply some back pressure
	// when connections overflood the acceptor

	// We are using multi sockets per listen, so each "run" here is pinned to its listener
	server.backPressure = make(chan bool, 1)
	defer close(server.backPressure)

	MakeHandler := func(servv *Server, listen net.Listener) {

		run := func() {

			// consume the input queue of lines
			for {
				select {
				case splitem, more := <-servv.InputQueue:
					if !more {
						return
					}
					//server.log.Notice("INQ: %d Line: %s", len(server.InputQueue), splitem.Line())
					if splitem == nil {
						continue
					}
					l_len := (int64)(len(splitem.Line()))
					servv.AddToCurrentTotalBufferSize(l_len)
					if servv.NeedBackPressure() {
						servv.log.Warning(
							"Error::Max Queue or buffer reached dropping connection (Buffer %v, queue len: %v)",
							servv.CurrentReadBufferRam.Get(),
							len(servv.WorkQueue))
						servv.BackPressure()
					}

					servv.ProcessSplitItem(splitem, servv.ProcessedQueue)
					servv.AddToCurrentTotalBufferSize(-l_len)

				}
			}
		}

		//fire up the workers
		go run()

		// the dispatcher is just one for each tcp server (clients feed into the main incoming line stream
		// unlike UDP which is "one client" effectively)

		accepts_queue, err := servv.Accepter(listen)
		if err != nil {
			panic(err)
		}

		shuts_client := servv.ShutDown.Listen()
		close_client := make(chan bool)
		fullName := fmt.Sprintf("worker.%s.queue.isfull", servv.Name)
		//lenName := fmt.Sprintf("worker.%s.queue.length", servv.Name)
		openName := fmt.Sprintf("worker.%s.tcp.connection.open", servv.Name)
		for {
			select {
			case conn, ok := <-accepts_queue:
				if !ok {
					return
				}
				//tcp_socket_out := make(chan splitter.SplitItem)

				client := NewTCPClient(servv, hashers, conn, nil, done)
				client.SetBufferSize((int)(servv.ClientReadBufferSize))
				log.Debug("Accepted con %v", conn.RemoteAddr())

				stats.StatsdClient.Incr(openName, 1)

				go client.handleRequest(nil, close_client)

			case <-shuts_client.Ch:
				servv.log.Warning("TCP Client: Shutdown gotten .. stopping incoming connections")
				listen.Close()
				close(close_client) // tell all handlers to stop
				return

			case workerUpDown := <-servv.WorkerHold:
				servv.InWorkQueue.Add(workerUpDown)
				ct := (int64)(len(servv.WorkQueue))
				if ct >= servv.Workers {
					stats.StatsdClient.Incr("worker.queue.isfull", 1)
					stats.StatsdClient.Incr(fullName, 1)
					//server.Logger.Printf("Worker Queue Full %d", ct)
				}
				//stats.StatsdClient.GaugeAvg("worker.queue.length", ct)
				//stats.StatsdClient.GaugeAvg(lenName, ct)
			case client := <-done:
				client.Close() //this will close the connection too
				client = nil
			}

			if servv.WorkerHold == nil || accepts_queue == nil {
				break
			}
		}
	}

	// unix sockets don't have the SO_REUSEPORT yet
	if server.Connection != nil {
		go MakeHandler(server, server.Connection)
	} else {
		// handler pre "tcp socket" (yes we can share the sockets thanks to SO_REUSEPORT)
		for _, listen := range server.TCPConns {
			go MakeHandler(server, listen)
		}
	}

	// just need to listen for shutdown at this point
	shuts := server.ShutDown.Listen()
	defer shuts.Close()
	for {
		select {
		case <-shuts.Ch:
			return
		}
	}
}

// different mechanism for UDP servers
func (server *Server) startUDPServer(hashers *[]*ConstHasher, done chan Client) {

	MakeHandler := func(servv *Server, conn net.PacketConn) {
		//just need one "client" here as we simply just pull from the socket
		client := NewUDPClient(servv, hashers, conn, done)
		client.SetBufferSize((int)(servv.ClientReadBufferSize))

		// this queue is just for "real" UDP sockets
		udpSocket := make(chan splitter.SplitItem)
		shuts := servv.ShutDown.Listen()
		defer shuts.Close()

		// close chan is not needed as the TCP one is as there is
		// "one" handler for UDP cons
		go client.handleRequest(udpSocket, nil)
		go client.handleSend(udpSocket)

		for {
			select {
			case <-shuts.Ch:
				servv.log.Warning("UDP shutdown acquired ... stopping incoming connections")
				client.ShutDown()
				conn.Close()
				return
			case workerUpDown := <-server.WorkerHold:
				servv.InWorkQueue.Add(workerUpDown)
				work_len := (int64)(len(server.WorkQueue))
				if work_len >= server.Workers {
					stats.StatsdClient.Incr("worker.queue.isfull", 1)
				}
				//stats.StatsdClient.GaugeAvg("worker.queue.length", work_len)

			case client := <-done:
				client.Close()
			}
		}
	}

	// handler pre "socket" (yes we can share the sockets thanks to SO_REUSEPORT)
	for _, conn := range server.UDPConns {
		go MakeHandler(server, conn)
	}

	// just need to listen for shutdown at this point
	shuts := server.ShutDown.Listen()
	defer shuts.Close()
	for {
		select {
		case <-shuts.Ch:
			return
		}
	}
}

// different mechanism for http servers much like UDP,
// client connections are handled by the golang http client bits so it appears
// as "one" client to us
func (server *Server) startHTTPServer(hashers *[]*ConstHasher, done chan Client) {

	client, err := NewHTTPClient(server, hashers, server.ListenURL, done)
	if err != nil {
		panic(err)
	}
	client.SetBufferSize((int)(server.ClientReadBufferSize))

	// this queue is just for "real" TCP sockets
	http_socket_out := make(chan splitter.SplitItem)
	shuts := server.ShutDown.Listen()
	defer shuts.Close()

	go client.handleRequest(http_socket_out, nil)
	go client.handleSend(http_socket_out)

	for {
		select {
		case <-shuts.Ch:
			client.ShutDown()
			return
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			work_len := (int64)(len(server.WorkQueue))
			if work_len >= server.Workers {
				stats.StatsdClient.Incr("worker.queue.isfull", 1)
			}
			//stats.StatsdClient.GaugeAvg("worker.queue.length", work_len)

		case client := <-done:
			client.Close()
		}
	}
}

func (server *Server) startBackendServer(hashers *[]*ConstHasher, done chan Client) {

	// start the "backend only loop"
	statLines := fmt.Sprintf("incoming.backend.%s.lines", server.Name)
	run := func() {
		for {
			select {
			case splitem, more := <-server.InputQueue:
				if !more {
					return
				}
				if splitem == nil {
					continue
				}
				lLen := (int64)(len(splitem.Line()))
				server.AddToCurrentTotalBufferSize(lLen)
				server.ProcessSplitItem(splitem, server.ProcessedQueue)
				server.AllLinesCount.Up(1)
				stats.StatsdClient.Incr("incoming.backend.lines", 1)
				stats.StatsdClient.Incr(statLines, 1)
				server.AddToCurrentTotalBufferSize(-lLen)
			}
		}
	}

	//fire up the workers
	for w := int64(1); w <= server.Workers; w++ {
		go run()
	}

	// "socket-less" consumers will eat the ProcessedQueue
	// just loooop
	shuts := server.ShutDown.Listen()
	select {
	case <-shuts.Ch:
		return
	}

}

// bleed the "non-server-backed" output queues as we need that for
// items that get placed on a server queue, but does not originate
// from a socket (where the output queue is needed for server responses and the like)
// Backend Servers use this exclusively as there are no sockets
func (server *Server) ConsumeProcessedQueue(qu chan splitter.SplitItem) {
	shuts := server.ShutDown.Listen()
	statFullname := fmt.Sprintf("worker.%s.queue.isfull", server.Name)
	//statLengthname := fmt.Sprintf("worker.%s.queue.length", server.Name)
	for {
		select {
		case <-shuts.Ch:
			shuts.Close()
			return
		case _, more := <-qu:
			if !more {
				return
			}
		case workerUpDown := <-server.WorkerHold:
			server.InWorkQueue.Add(workerUpDown)
			ct := (int64)(len(server.WorkQueue))
			if ct >= server.Workers {
				stats.StatsdClient.Incr("worker.queue.isfull", 1)
				stats.StatsdClient.Incr(statFullname, 1)
				//server.log.Warningf("ConsumeProcessedQueue: Worker Queue Full: %d", ct)
			}
			//stats.StatsdClient.GaugeAvg("worker.queue.length", ct)
			//stats.StatsdClient.GaugeAvg(statLengthname, ct)
		}
	}
}

func CreateServer(cfg *config.HasherConfig, hashers []*ConstHasher) (*Server, error) {
	server, err := NewServer(cfg)

	if err != nil {
		panic(err)
	}
	server.Hashers = hashers
	server.Workers = DEFAULT_WORKERS
	if cfg.Workers > 0 {
		server.Workers = int64(cfg.Workers)
	}

	server.OutWorkers = DEFAULT_OUTPUT_WORKERS
	if cfg.OutWorkers > 0 {
		server.OutWorkers = int64(cfg.OutWorkers)
	}

	// fire up the listeners
	err = server.StartListen(cfg.ListenURL)

	if err != nil {
		panic(err)
	}

	server.StopTicker = make(chan bool, 1)

	//start tickin'
	if cfg.StatsTick {
		server.ShowStats = true
	}
	go server.tickDisplay()

	//server.TrapExit() //trappers

	// add it to the list of backends available
	ServerBackends.Add(server.Name, server)

	return server, err
}

func (server *Server) StartServer() {

	server.log.Info("Using %d workers to process output", server.Workers)

	//fire up the send to workers
	done := make(chan Client)

	//set the push method (the WorkerOutput depends on it)
	server.SetWriter()

	// the output dispatcher, just 4 workers here
	for w := int64(1); w <= 4; w++ {
		go server.WorkerOutput() // puts things on the dispatch queue
	}

	server.OutputDispatcher = NewOutputDispatcher(int(server.OutWorkers))
	server.OutputDispatcher.Run()

	//get our line proessor in order
	_, err := server.SetSplitterProcessor()
	if err != nil {
		panic(err)
	}

	//fire off checkers
	for _, hash := range server.Hashers {
		go hash.ServerPool.StartChecks()
	}

	//set the accumulator to this servers input queue
	if server.PreRegFilter != nil && server.PreRegFilter.Accumulator != nil {
		// set where we are from
		server.PreRegFilter.Accumulator.FromBackend = server.Name
		// There is the special "black_hole" Backend that will let us use Writers exclusively
		if server.PreRegFilter.Accumulator.ToBackend == accumulator.BLACK_HOLE_BACKEND {
			log.Notice("NOTE: BlackHole for `%s`", server.PreRegFilter.Accumulator.Name)
		} else {
			to_srv := ServerBackends[server.PreRegFilter.Accumulator.ToBackend]
			log.Notice("Assiging OutQueue for `%s` to backend `%s` ", server.PreRegFilter.Accumulator.Name, to_srv.Name)
			server.PreRegFilter.Accumulator.SetOutputQueue(to_srv.Queue.InputQueue)
			// fire it up
		}

		go server.PreRegFilter.Accumulator.Start()
	}

	// fire up the socket-less consumers for processed items
	for w := int64(1); w <= server.Workers; w++ {
		go server.ConsumeProcessedQueue(server.ProcessedQueue)
	}

	if server.ListenURL == nil {
		// just the little queue listener for pre-reg only redirects
		server.startBackendServer(&server.Hashers, done)
	} else if len(server.UDPConns) > 0 {
		server.startUDPServer(&server.Hashers, done)
	} else if server.ListenURL.Scheme == "http" {
		server.startHTTPServer(&server.Hashers, done)
	} else {
		// we treat the generic TCP and UNIX listeners the same
		server.startTCPServer(&server.Hashers, done)
	}

}
