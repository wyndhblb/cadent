package gostatsdclient

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

    "github.com/wyndhblb/gostatsdclient/event"
	"sync"
	"net/url"
)

// note Hostname is exported so clients can set it to something different than the default
var Hostname string
var ShortHostname string

func init() {
	host, err := os.Hostname()
	if nil == err {
		Hostname = host
		spl := strings.Split(host, ".")
		ShortHostname = spl[0]
	}
}

// StatsdClient is a client library to send events to StatsD
type StatsdClient struct {
	conn            net.Conn
	addr            string
	prefix          string
	Protocol        string
	Logger          *log.Logger
	SampleRate      float32
	TimerSampleRate float32
	BufSize         int
	buffer          []byte
	bufLock         sync.RWMutex
	flushTick       time.Duration
	randChan	chan float32
}

// NewStatsdClient - Factory
func NewStatsdClient(addr string, prefix string, prefhost string) *StatsdClient {
	// allow %HOST% in the prefix string
	prefix = strings.Replace(prefix, "%HOST%", Hostname, 1)
	switch prefhost {
	case "none":
		prefix = strings.Replace(prefix, "%HOST%", "", 1)
	case "short":
		prefix = strings.Replace(prefix, "%HOST%", ShortHostname, 1)
	default:
		prefix = strings.Replace(prefix, "%HOST%", Hostname, 1)
	}


	// see if addr has the protocol in it
	proto := "udp"
	p, err := url.Parse(addr)
	if err == nil && p != nil {
		proto = p.Scheme
		addr = p.Host
	}

	if !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}
	sc := &StatsdClient{
		addr:            addr,
		prefix:          prefix,
		Protocol:        proto,
		Logger:          log.New(os.Stdout, "[StatsdClient] ", log.Ldate|log.Ltime),
		SampleRate:      1.0,
		TimerSampleRate: 1.0,
		flushTick:       time.Duration(time.Second),
		BufSize:         0,
		randChan:        make(chan float32, 2056),

	}
	go sc.genRands()
	return sc
}

// start the local random generator
func (sb *StatsdClient) genRands(){
	nr := rand.New(rand.NewSource(time.Now().UnixNano()))
	for{
		sb.randChan <- nr.Float32()
	}
}

// NewStatsdClientBuffered new client but only send once the buffer is full
// of if the time (1 second) has expired
func NewStatsdClientBuffered(addr string, prefix string, prefhost string, bufSize int) *StatsdClient {
	cli := NewStatsdClient(addr, prefix, prefhost)
	if bufSize > 0 {
		cli.BufSize = bufSize
		go cli.periodicSend()
	}
	go cli.genRands()
	return cli
}

func (sb *StatsdClient) registerStat() bool {
	if sb.SampleRate >= 1.0 {
		return true
	}
	return <-sb.randChan < sb.SampleRate
}

func (sb *StatsdClient) registerTimerStat() bool {
	if sb.SampleRate >= 1.0 {
		return true
	}
	return <-sb.randChan < sb.TimerSampleRate
}

// String returns the StatsD server address
func (c *StatsdClient) String() string {
	return c.addr
}

// CreateSocket creates a UDP connection to a StatsD server
func (c *StatsdClient) CreateSocket() error {
	conn, err := net.DialTimeout(c.Protocol, c.addr, 5*time.Second)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Close the UDP connection
func (c *StatsdClient) Close() error {
	if nil == c.conn {
		return nil
	}
	return c.conn.Close()
}

// See statsd data types here: http://statsd.readthedocs.org/en/latest/types.html
// or also https://github.com/b/statsd_spec

// Incr - Increment a counter metric. Often used to note a particular event
func (c *StatsdClient) Incr(stat string, count int64) error {
	if !c.registerStat() {
		return nil
	}
	if 0 != count {
		return c.send(stat, "%d|c", count)
	}
	return nil
}

// Decr - Decrement a counter metric. Often used to note a particular event
func (c *StatsdClient) Decr(stat string, count int64) error {
	if !c.registerStat() {
		return nil
	}
	if 0 != count {
		return c.send(stat, "%d|c", -count)
	}
	return nil
}

// Timing - Track a duration event
// the time delta must be given in milliseconds
func (c *StatsdClient) Timing(stat string, delta int64) error {
	if !c.registerStat() {
		return nil
	}
	return c.send(stat, "%d|ms", delta)
}

// PrecisionTiming - Track a duration event
// the time delta has to be a duration
func (c *StatsdClient) PrecisionTiming(stat string, delta time.Duration) error {
	if !c.registerTimerStat() {
		return nil
	}
	return c.send(stat, fmt.Sprintf("%.6f%s|ms", float64(delta)/float64(time.Millisecond), "%d"), 0)
}

// Gauge - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again. If you specify
// delta to be true, that specifies that the gauge should be updated, not set. Due to the
// underlying protocol, you can't explicitly set a gauge to a negative number without
// first setting it to zero.
func (c *StatsdClient) Gauge(stat string, value int64) error {
	if !c.registerStat() {
		return nil
	}
	if value < 0 {
		c.send(stat, "%d|g", 0)
		return c.send(stat, "%d|g", value)
	}
	return c.send(stat, "%d|g", value)
}

// the Buffered client has a GaugeAbsolute which DOES NOT "add" values in the buffer, just
// resets the value to the current value, here in the non-buffered client it is
// the same as a Gauge
func (c *StatsdClient) GaugeAbsolute(stat string, value int64) error {
	return c.Gauge(stat, value)
}

// the Buffered client has a GaugeAbsolute which DOES NOT "add" values in the buffer, just
// resets the value to the current value, here in the non-buffered client it is
// the same as a Gauge
func (c *StatsdClient) GaugeAvg(stat string, value int64) error {
	return c.Gauge(stat, value)
}

// GaugeDelta -- Send a change for a gauge
func (c *StatsdClient) GaugeDelta(stat string, value int64) error {
	if !c.registerStat() {
		return nil
	}
	// Gauge Deltas are always sent with a leading '+' or '-'. The '-' takes care of itself but the '+' must added by hand
	if value < 0 {
		return c.send(stat, "%d|g", value)
	}
	return c.send(stat, "+%d|g", value)
}

// FGauge -- Send a floating point value for a gauge
func (c *StatsdClient) FGauge(stat string, value float64) error {
	if !c.registerStat() {
		return nil
	}
	if value < 0 {
		c.send(stat, "%d|g", 0)
		return c.send(stat, "%g|g", value)
	}
	return c.send(stat, "%g|g", value)
}

// FGaugeDelta -- Send a floating point change for a gauge
func (c *StatsdClient) FGaugeDelta(stat string, value float64) error {
	if !c.registerStat() {
		return nil
	}
	if value < 0 {
		return c.send(stat, "%g|g", value)
	}
	return c.send(stat, "+%g|g", value)
}

// Absolute - Send absolute-valued metric (not averaged/aggregated)
func (c *StatsdClient) Absolute(stat string, value int64) error {
	if !c.registerStat() {
		return nil
	}
	return c.send(stat, "%d|c", value)
}

// FAbsolute - Send absolute-valued floating point metric (not averaged/aggregated)
func (c *StatsdClient) FAbsolute(stat string, value float64) error {
	if !c.registerStat() {
		return nil
	}
	return c.send(stat, "%g|c", value)
}

// Total - Send a metric that is continously increasing, e.g. read operations since boot
func (c *StatsdClient) Total(stat string, value int64) error {
	if !c.registerStat() {
		return nil
	}
	return c.send(stat, "%d|c", value)
}

// write a UDP packet with the statsd event
func (c *StatsdClient) send(stat string, format string, value interface{}) error {

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	stat = strings.Replace(stat, "%HOST%", Hostname, 1)
	format = fmt.Sprintf("%s%s:%s", c.prefix, stat, format)
	if c.BufSize <= 0 {
		_, err := fmt.Fprintf(c.conn, format, value)
		return err
	} else {
		c.bufLock.Lock()
		if len(c.buffer) > c.BufSize {
			c.sendBufferToSocket()
			c.buffer = c.buffer[:0]
		}
		c.buffer = append(c.buffer, []byte(fmt.Sprintf(format, value)+"\n")...)
		c.bufLock.Unlock()
	}
	return nil
}

func (c *StatsdClient) periodicSend() {
	tick := time.NewTicker(c.flushTick)
	for {
		<-tick.C
		c.bufLock.Lock()
		if len(c.buffer) > 0 {
			c.sendBufferToSocket()
			c.buffer = c.buffer[:0]
		}
		c.bufLock.Unlock()
	}
}

// write the buffer to the connection
func (c *StatsdClient) sendBufferToSocket() error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err := c.conn.Write(c.buffer)
	return err
}

func (c *StatsdClient) SendRaw(buffer string) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	//log.Printf("SENDING EVENT %s", buffer)

	_, err := fmt.Fprintf(c.conn, buffer)
	return err

}

func (c *StatsdClient) EventStatsdString(e event.Event, tick time.Duration, samplerate float32) (string, error) {
	var out_str = ""

	for _, stat := range e.Stats(tick) {
		str := fmt.Sprintf("%s%s", c.prefix, stat)

		if len(str) > 0 {
			if samplerate < 1.0 {
				str += fmt.Sprintf("|@%f", samplerate)
			}
			out_str += str + "\n"
		}
	}
	return out_str, nil

}

// special case for timers, if we are a buffered client,
// then we DO NOT add the sample rate to the acctuall "timers" parts of the metric lines
// (lower, upper, median, etc...) just to the "count and count_ps"
func (c *StatsdClient) EventStatsdStringTimerSample(e event.Event, tick time.Duration, samplerate float32) (string, error) {
	var out_str = ""

	for _, stat := range e.Stats(tick) {
		str := fmt.Sprintf("%s%s", c.prefix, stat)

		if len(str) > 0 {
			// just count and count ps
			if samplerate < 1.0 && (strings.Contains(stat, "count:") || strings.Contains(stat, "count_ps:")) {
				str += fmt.Sprintf("|@%f", samplerate)
			}
			out_str += str + "\n"
		}
	}
	return out_str, nil

}

// SendEvent - Sends stats from an event object
func (c *StatsdClient) SendEvent(e event.Event, tick time.Duration) error {
	if c.conn == nil {
		return fmt.Errorf("cannot send stats, not connected to StatsD server")
	}
	for _, stat := range e.Stats(tick) {
		//fmt.Printf("SENDING EVENT %s%s\n", c.prefix, stat)
		_, err := fmt.Fprintf(c.conn, "%s%s", c.prefix, stat)
		if nil != err {
			return err
		}
	}
	return nil
}
