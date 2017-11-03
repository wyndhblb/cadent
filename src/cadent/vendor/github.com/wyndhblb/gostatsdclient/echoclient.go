package gostatsdclient

import (
	"log"
	"time"
)

//StatsdEcho an "echo" statsd for debugging
type StatsdEcho struct {
	Prefix string
}

func (s *StatsdEcho) prit(stat string, t string, val interface{}) error {
	log.Printf("%s%s:%v|%s", s.Prefix, stat, val, t)
	return nil
}

func (s StatsdEcho) String() string                        { return "EchoClient" }
func (s StatsdEcho) CreateSocket() error                   { return nil }
func (s StatsdEcho) Close() error                          { return nil }
func (s StatsdEcho) Incr(stat string, count int64) error   { return s.prit(stat, "c", count) }
func (s StatsdEcho) Decr(stat string, count int64) error   { return s.prit(stat, "c", count) }
func (s StatsdEcho) Timing(stat string, count int64) error { return s.prit(stat, "ms", count) }
func (s StatsdEcho) PrecisionTiming(stat string, delta time.Duration) error {
	return s.prit(stat, "ms", delta)
}
func (s StatsdEcho) Gauge(stat string, value int64) error         { return s.prit(stat, "g", value) }
func (s StatsdEcho) GaugeAbsolute(stat string, value int64) error { return s.prit(stat, "g", value) }
func (s StatsdEcho) GaugeAvg(stat string, value int64) error      { return s.prit(stat, "g", value) }
func (s StatsdEcho) GaugeDelta(stat string, value int64) error    { return s.prit(stat, "+g", value) }
func (s StatsdEcho) Absolute(stat string, value int64) error      { return s.prit(stat, "c", value) }
func (s StatsdEcho) Total(stat string, value int64) error         { return s.prit(stat, "c", value) }
func (s StatsdEcho) FGauge(stat string, value float64) error      { return s.prit(stat, "g", value) }
func (s StatsdEcho) FGaugeDelta(stat string, value float64) error { return s.prit(stat, "g", value) }
func (s StatsdEcho) FAbsolute(stat string, value float64) error   { return s.prit(stat, "g", value) }
