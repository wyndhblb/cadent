package event

import (
	"fmt"
	"sync"
	"time"
)

// PrecisionTiming keeps min/max/avg information about a timer over a certain interval
type PrecisionTiming struct {
	Name  string
	mu    sync.Mutex
	Min   time.Duration
	Max   time.Duration
	Value time.Duration
	Count int64
}

func (e *PrecisionTiming) StatClass() string {
	return "timer"
}

// NewPrecisionTiming is a factory for a Timing event, setting the Count to 1 to prevent div_by_0 errors
func NewPrecisionTiming(k string, delta time.Duration) *PrecisionTiming {
	return &PrecisionTiming{Name: k, Min: delta, Max: delta, Value: delta, Count: 1}
}

// Update the event with metrics coming from a new one of the same type and with the same key
func (e *PrecisionTiming) Update(e2 Event) error {
	if e.Type() != e2.Type() {
		return fmt.Errorf("statsd event type conflict: %s vs %s ", e.String(), e2.String())
	}
	p := e2.Payload().(PrecisionTiming)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Count += p.Count
	e.Value += p.Value
	e.Min = time.Duration(minInt64(int64(e.Min), int64(p.Min)))
	e.Max = time.Duration(maxInt64(int64(e.Max), int64(p.Min)))
	return nil
}

//Reset the value
func (e *PrecisionTiming) Reset() {
	e.Value = 0
	e.Count = 1
	e.Min = time.Duration(0)
	e.Max = time.Duration(0)
}

// Payload returns the aggregated value for this event
func (e PrecisionTiming) Payload() interface{} {
	return e
}

// Stats returns an array of StatsD events as they travel over UDP
func (e PrecisionTiming) Stats(tick time.Duration) []string {
	return []string{
		fmt.Sprintf("%s.count:%d|c", e.Name, int64(e.Count)),
		fmt.Sprintf("%s.avg:%.6f|ms", e.Name, float64(int64(e.Value)/e.Count)), // make sure e.Count != 0
		fmt.Sprintf("%s.min:%.6f|ms", e.Name, float64(e.Min)),
		fmt.Sprintf("%s.max:%.6f|ms", e.Name, float64(e.Max)),
	}
}

// Key returns the name of this metric
func (e PrecisionTiming) Key() string {
	return e.Name
}

// SetKey sets the name of this metric
func (e *PrecisionTiming) SetKey(key string) {
	e.Name = key
}

// Type returns an integer identifier for this type of metric
func (e PrecisionTiming) Type() int {
	return EventPrecisionTiming
}

// TypeString returns a name for this type of metric
func (e PrecisionTiming) TypeString() string {
	return "PrecisionTiming"
}

// String returns a debug-friendly representation of this metric
func (e PrecisionTiming) String() string {
	return fmt.Sprintf("{Type: %s, Key: %s, Value: %+v}", e.TypeString(), e.Name, e.Payload())
}
