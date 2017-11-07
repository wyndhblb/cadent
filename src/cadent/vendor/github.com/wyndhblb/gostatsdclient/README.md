
# GoLang Statsd Client

A little lib for async statsd client to emmit things to statsd in a buffered (or not) fashion

It will also place the metrics on the the `expvar` golang debug stack `/debug/vars` (counters "add" forever, gauges and timers are their last seen value)

```
{
    "counters": {
        "key": v,
        ...
    },
    "timers": {
        "key": v,
        ...
    },
    "gauges": {
        "key": v,
        ...
    }
}
```




## use

```
import (
time
"github.com/wyndhblb/gostatsdclient"
)

var statsd statsd.Statsd = nil
func init(){

    // an "echo to StdOut client"
    // statsd = gostatsdclient.StatsdEcho{}
    
    // a NoOp client (does nothing)
    // statsd = gostatsdclient.StatsdNoop{}

    // a "direct" non-buffered client
    // non-buffered means that metrics will be sent as they come in and not buffered
    // ok for localhost and really fast nets, prefer the buffered client instead in most
    // production cases
    
    prefix := "mystat"
    hostname := "my.host.com"

    // this will prefix all metrics emissions by
    // {hostname}.{prefix}.{metric_name}
    statsdClient = statsd.NewStatsdClient("udp://127.0.0.1:8125", prefix, hostname)
    
    // build a buffered client from the direct client
    
    // if the buffer is not "full" before this interval is up, send the metrics anyway
    interval :=  time.Second * time.Duration(1)
    
    statsd = statsd.NewStatsdBuffer("my-nifty-client", interval, statsdClient)
    statsd.BufferLength = 1024 // bytes before a flush
    statsd.SampleRate = 1.0 // global counter sample rate
    
    // different global rate for timers as they can be called much more
    // frequently for fast function
    statsd.TimerSampleRate = 1.0

}

func main(){

    // fire off some metrics
    statsd.Incr("my.counter.is.nice", 1)    // increment counters
    statsd.Decr("my.counter.is.nice", 1)    // decrement counters
    statsd.Timing("my.timer.is.nice", 0.12) // a timer
    statsd.TimingSampling("my.timer_sampled.is.nice", 0.5, 0.2)  // a timer w/ sampling
    statsd.Gauge("my.gauge.is.nice", 10)  // gauges
    
    statsd.Close() // flush and terminate 
}
```

### Note:

Original based off of this was based off of http://github.com/quipo/statsd
