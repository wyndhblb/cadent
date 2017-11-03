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
	Encode/Decode bits for Kafka to send metric messages around
*/

package series

import (
	"cadent/server/schemas"
	"cadent/server/schemas/repr"
	"encoding/json"
	"fmt"
	"sync"
)

/** sync pools for ease of ram pressure on fast objects **/
var seriesPool sync.Pool

func getSeries() *KSeriesMetric {
	x := seriesPool.Get()
	if x == nil {
		return new(KSeriesMetric)
	}
	x.(*KSeriesMetric).encoded = []byte{}
	x.(*KSeriesMetric).err = nil
	return x.(*KSeriesMetric)
}

func putSeries(spl *KSeriesMetric) {
	seriesPool.Put(spl)
}

var unpPool sync.Pool

func getUnp() *KUnProcessedMetric {
	x := unpPool.Get()
	if x == nil {
		return new(KUnProcessedMetric)
	}
	x.(*KUnProcessedMetric).encoded = nil
	x.(*KUnProcessedMetric).err = nil
	return x.(*KUnProcessedMetric)
}

func putUnp(spl *KUnProcessedMetric) {
	unpPool.Put(spl)
}

var rawPool sync.Pool

func getRaw() *KRawMetric {
	x := rawPool.Get()
	if x == nil {
		return new(KRawMetric)
	}
	x.(*KRawMetric).encoded = nil
	x.(*KRawMetric).err = nil
	return x.(*KRawMetric)
}

func putRaw(spl *KRawMetric) {
	rawPool.Put(spl)
}

var singPool sync.Pool

func getSing() *KSingleMetric {
	x := singPool.Get()
	if x == nil {
		return new(KSingleMetric)
	}
	x.(*KSingleMetric).encoded = nil
	x.(*KSingleMetric).err = nil
	return x.(*KSingleMetric)
}

func putSing(spl *KSingleMetric) {
	singPool.Put(spl)
}

var writtenPool sync.Pool

func getWritten() *KWrittenMessage {
	x := writtenPool.Get()
	if x == nil {
		return new(KWrittenMessage)
	}
	x.(*KWrittenMessage).encoded = nil
	x.(*KWrittenMessage).err = nil
	x.(*KWrittenMessage).Topic = ""
	x.(*KWrittenMessage).Partition = 0
	x.(*KWrittenMessage).Offset = 0
	x.(*KWrittenMessage).Uid = ""
	x.(*KWrittenMessage).Metric = ""

	return x.(*KWrittenMessage)
}

func putWritten(spl *KWrittenMessage) {
	writtenPool.Put(spl)
}

var anyPool sync.Pool

func getAny() *KMetric {
	x := anyPool.Get()
	if x == nil {
		return new(KMetric)
	}
	x.(*KMetric).encoded = nil
	x.(*KMetric).err = nil
	return x.(*KMetric)
}

func putAny(spl *KMetric) {
	anyPool.Put(spl)
}

func PutPool(ty KMessageBase) {
	switch ty.(type) {
	case *KMetric:
		putAny(ty.(*KMetric))
	case *KSeriesMetric:
		putSeries(ty.(*KSeriesMetric))
	case *KUnProcessedMetric:
		putUnp(ty.(*KUnProcessedMetric))
	case *KRawMetric:
		putRaw(ty.(*KRawMetric))
	case *KSingleMetric:
		putSing(ty.(*KSingleMetric))
	case *KWrittenMessage:
		putWritten(ty.(*KWrittenMessage))
	}
}

func KMetricObjectFromType(ty MessageType) KMessageBase {
	switch ty {
	case MSG_SERIES:
		return getSeries()
	case MSG_UNPROCESSED:
		return getUnp()
	case MSG_RAW:
		return getRaw()
	case MSG_ANY:
		return getAny()
	case MSG_WRITTEN:
		return getWritten()
	default:
		return getSing()
	}
}

// KMessageBase for non list types
type KMessageBase interface {
	SetSendEncoding(enc SendEncoding)
	Id() string
	Length() int
	Encode() ([]byte, error)
	Decode([]byte) error
}

type KMessageSingle interface {
	KMessageBase
	Repr() *repr.StatRepr
}

type KMessageSeries interface {
	KMessageBase
	Reprs() []*repr.StatRepr
}

// KMessageListBase interface for list types
type KMessageListBase interface {
	SetSendEncoding(enc SendEncoding)
	Ids() []string
	Length() int
	Encode() ([]byte, error)
	Decode([]byte) error
	Reprs() repr.StatReprSlice
}

/*************** Envelope message, containing the "type" with it ****/
type KMetric struct {
	AnyMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KMetric) Id() string {
	switch {
	case kp.Raw != nil:
		return kp.Raw.Metric
	case kp.Unprocessed != nil:
		return kp.Unprocessed.Metric
	case kp.Single != nil:
		return kp.Single.Uid
	case kp.UidMetric != nil:
		return kp.UidMetric.Uid
	case kp.Series != nil:
		return kp.Series.Uid
	case kp.Written != nil:
		return kp.Written.Uid
	default:
		return ""
	}
}

// HashKey the key we use to pick the proper partitions in kafka
func (kp *KMetric) HashKey() string {
	switch {
	case kp.Raw != nil:
		return kp.Raw.Metric
	case kp.Unprocessed != nil:
		return kp.Unprocessed.Metric
	case kp.Single != nil:
		return kp.Single.Uid
	case kp.Series != nil:
		return kp.Series.Uid
	case kp.UidMetric != nil:
		return kp.UidMetric.Uid
	case kp.Written != nil:
		return kp.Written.Uid
	default:
		return ""
	}
}

// Repr return the repr.StatRepr from the message
func (kp *KMetric) Repr() *repr.StatRepr {

	switch {
	case kp.Raw != nil:

		return &repr.StatRepr{
			Name: &repr.StatName{
				Key:      kp.Raw.Metric,
				Tags:     kp.Raw.Tags,
				MetaTags: kp.Raw.MetaTags,
			},
			Time:  kp.Raw.Time,
			Sum:   kp.Raw.Value,
			Count: 1,
		}
	case kp.Unprocessed != nil:
		return &repr.StatRepr{
			Name: &repr.StatName{
				Key:      kp.Unprocessed.Metric,
				Tags:     kp.Unprocessed.Tags,
				MetaTags: kp.Unprocessed.MetaTags,
			},
			Time:  kp.Unprocessed.Time,
			Min:   repr.CheckFloat(kp.Unprocessed.Min),
			Max:   repr.CheckFloat(kp.Unprocessed.Max),
			Last:  repr.CheckFloat(kp.Unprocessed.Last),
			Sum:   repr.CheckFloat(kp.Unprocessed.Sum),
			Count: kp.Unprocessed.Count,
		}

	case kp.UidMetric != nil:
		nm := new(repr.StatName)
		nm.SetUidString(kp.UidMetric.Uid)
		return &repr.StatRepr{
			Name:  nm,
			Time:  kp.UidMetric.Time,
			Min:   repr.CheckFloat(kp.UidMetric.Min),
			Max:   repr.CheckFloat(kp.UidMetric.Max),
			Last:  repr.CheckFloat(kp.UidMetric.Last),
			Sum:   repr.CheckFloat(kp.UidMetric.Sum),
			Count: kp.UidMetric.Count,
		}

	case kp.Single != nil:
		return &repr.StatRepr{
			Name: &repr.StatName{
				Key:        kp.Single.Metric,
				Tags:       kp.Single.Tags,
				MetaTags:   kp.Single.MetaTags,
				Resolution: kp.Single.Resolution,
				Ttl:        kp.Single.Ttl,
			},
			Time:  kp.Single.Time,
			Min:   repr.CheckFloat(kp.Single.Min),
			Max:   repr.CheckFloat(kp.Single.Max),
			Last:  repr.CheckFloat(kp.Single.Last),
			Sum:   repr.CheckFloat(kp.Single.Sum),
			Count: kp.Single.Count,
		}
	default:
		return nil
	}
}

// SetSendEncoding set message encoding (json, msgpack, protobuf)
func (kp *KMetric) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc
}

// encodes the 'base' message
func (kp *KMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()

	if kp != nil && kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.AnyMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.AnyMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp)
			return

		}
	}
}

// Length number of bytes in the encoded message
func (kp *KMetric) Length() int {
	if kp == nil {
		return 0
	}
	kp.ensureEncoded()
	return len(kp.encoded)
}

// Encode convert the message to a byte slice
func (kp *KMetric) Encode() ([]byte, error) {
	if kp == nil {
		return nil, schemas.ErrMetricIsNil
	}
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

// Decode the message into the internal go object
func (kp *KMetric) Decode(b []byte) (err error) {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err = kp.AnyMetric.UnmarshalMsg(b)
	case ENCODE_PROTOBUF:
		err = kp.AnyMetric.Unmarshal(b)
	default:
		err = json.Unmarshal(b, &kp.AnyMetric)
	}

	return err
}

/**************** series **********************/
type KSeriesMetric struct {
	SeriesMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

/*
func (kp *SingleMetric) Reprs() *repr.StatReprSlice {

	// need to convert the mighty

	return &repr.StatRepr{
		Name: &repr.StatName{
			Key:        kp.Metric,
			Tags:       kp.Tags,
			MetaTags:   kp.MetaTags,
			Resolution: kp.Resolution,
			Ttl:        kp.Ttl,
		},
		Min:   repr.CheckFloat(kp.Min),
		Max:   repr.CheckFloat(kp.Max),
		Last:  repr.CheckFloat(kp.Last),
		Sum:   repr.CheckFloat(kp.Sum),
		Count: kp.Count,
	}
}*/

func (kp *KSeriesMetric) Id() string {
	return kp.SeriesMetric.Uid
}

func (kp *KSeriesMetric) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc

}

func (kp *KSeriesMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()

	if kp != nil && kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.SeriesMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.SeriesMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp.SeriesMetric)

		}
	}
}

func (kp *KSeriesMetric) Length() int {
	if kp == nil {
		return 0
	}
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KSeriesMetric) Encode() ([]byte, error) {
	if kp == nil {
		return nil, schemas.ErrMetricIsNil
	}
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KSeriesMetric) Decode(b []byte) (err error) {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err = kp.UnmarshalMsg(b)
	case ENCODE_PROTOBUF:
		err = kp.SeriesMetric.Unmarshal(b)
	default:
		err = json.Unmarshal(b, &kp.SeriesMetric)
	}
	return err
}

/**************** Single **********************/

// KSingleMetric kafka single metric
type KSingleMetric struct {
	SingleMetric

	Type       string
	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KSingleMetric) Id() string {
	return kp.SingleMetric.Uid
}

func (kp *KSingleMetric) Repr() *repr.StatRepr {

	return &repr.StatRepr{
		Name: &repr.StatName{
			Key:        kp.SingleMetric.Metric,
			Tags:       kp.SingleMetric.Tags,
			MetaTags:   kp.SingleMetric.MetaTags,
			Resolution: kp.SingleMetric.Resolution,
			Ttl:        kp.SingleMetric.Ttl,
		},
		Time:  kp.SingleMetric.Time,
		Min:   repr.CheckFloat(kp.SingleMetric.Min),
		Max:   repr.CheckFloat(kp.SingleMetric.Max),
		Last:  repr.CheckFloat(kp.SingleMetric.Last),
		Sum:   repr.CheckFloat(kp.SingleMetric.Sum),
		Count: kp.SingleMetric.Count,
	}
}

func (kp *KSingleMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.SingleMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.SingleMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp.SingleMetric)

		}
	}
}

func (kp *KSingleMetric) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc
}

func (kp *KSingleMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KSingleMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KSingleMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.SingleMetric.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.SingleMetric.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.SingleMetric)
	}
}

// KUidMetric

// KSingleMetric kafka single metric
type KUidMetric struct {
	UidMetric

	Type       string
	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KUidMetric) Id() string {
	return kp.UidMetric.Uid
}

func (kp *KUidMetric) Repr() *repr.StatRepr {
	nm := new(repr.StatName)
	nm.SetUidString(kp.UidMetric.Uid)
	return &repr.StatRepr{
		Name:  nm,
		Time:  kp.UidMetric.Time,
		Min:   repr.CheckFloat(kp.UidMetric.Min),
		Max:   repr.CheckFloat(kp.UidMetric.Max),
		Last:  repr.CheckFloat(kp.UidMetric.Last),
		Sum:   repr.CheckFloat(kp.UidMetric.Sum),
		Count: kp.UidMetric.Count,
	}
}

func (kp *KUidMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.UidMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.UidMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp.UidMetric)

		}
	}
}

func (kp *KUidMetric) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc
}

func (kp *KUidMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KUidMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KUidMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.UidMetric.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.UidMetric.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.UidMetric)
	}
}

// KUnProcessedMetric kafka unprocessed metric
type KUnProcessedMetric struct {
	UnProcessedMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KUnProcessedMetric) Id() string {
	return kp.UnProcessedMetric.Metric
}

func (kp *KUnProcessedMetric) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Name: &repr.StatName{
			Key:      kp.UnProcessedMetric.Metric,
			Tags:     kp.UnProcessedMetric.Tags,
			MetaTags: kp.UnProcessedMetric.MetaTags,
		},
		Time:  kp.UnProcessedMetric.Time,
		Min:   repr.CheckFloat(kp.UnProcessedMetric.Min),
		Max:   repr.CheckFloat(kp.UnProcessedMetric.Max),
		Last:  repr.CheckFloat(kp.UnProcessedMetric.Last),
		Sum:   repr.CheckFloat(kp.UnProcessedMetric.Sum),
		Count: kp.UnProcessedMetric.Count,
	}
}

func (kp *KUnProcessedMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.UnProcessedMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.UnProcessedMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp.UnProcessedMetric)

		}
	}
}

func (kp *KUnProcessedMetric) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc

}

func (kp *KUnProcessedMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KUnProcessedMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KUnProcessedMetric) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.UnProcessedMetric.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.UnProcessedMetric.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.UnProcessedMetric)
	}
}

// KRawMetric kafka raw metric
type KRawMetric struct {
	RawMetric

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KRawMetric) Id() string {
	return kp.RawMetric.Metric
}

func (kp *KRawMetric) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.RawMetric.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.RawMetric.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(&kp.RawMetric)

		}
	}

}

func (kp *KRawMetric) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc

}

func (kp *KRawMetric) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KRawMetric) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KRawMetric) Decode(b []byte) error {

	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.RawMetric.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.RawMetric.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.RawMetric)
	}
}

func (kp *KRawMetric) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Name: &repr.StatName{
			Key:      kp.RawMetric.Metric,
			Tags:     kp.RawMetric.Tags,
			MetaTags: kp.RawMetric.MetaTags,
		},
		Time:  kp.RawMetric.Time,
		Sum:   repr.CheckFloat(kp.RawMetric.Value),
		Count: 1,
	}
}

// KWrittenMessage kafka MetricWritten object
type KWrittenMessage struct {
	MetricWritten

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KWrittenMessage) Id() string {
	return kp.MetricWritten.Uid
}

func (kp *KWrittenMessage) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.MetricWritten.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.MetricWritten.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(&kp.MetricWritten)

		}
	}
}

func (kp *KWrittenMessage) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc

}

func (kp *KWrittenMessage) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KWrittenMessage) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KWrittenMessage) Decode(b []byte) error {

	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.MetricWritten.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.MetricWritten.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.MetricWritten)
	}
}

func (kp *KWrittenMessage) Repr() *repr.StatRepr {
	return nil
}

/*****************************************************************************/
/***** LIST types *****/
/*****************************************************************************/

// KMetricList kafka any metric list object
type KMetricList struct {
	AnyMetricList

	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KMetricList) Ids() []string {
	ids := []string{}
	switch {
	case kp.Raw != nil:
		for _, m := range kp.Raw {
			ids = append(ids, m.Metric)
		}
	case kp.Unprocessed != nil:
		for _, m := range kp.Unprocessed {
			ids = append(ids, m.Metric)
		}
	case kp.Single != nil:
		for _, m := range kp.Single {
			ids = append(ids, m.Uid)
		}
	}
	return ids
}

// Reprs return the list of repr.StatRepr from the message
func (kp *KMetricList) Reprs() repr.StatReprSlice {

	stats := repr.StatReprSlice{}

	switch {
	case kp.Raw != nil:
		for _, m := range kp.Raw {
			stats = append(stats, &repr.StatRepr{
				Name: &repr.StatName{
					Key:      m.Metric,
					Tags:     m.Tags,
					MetaTags: m.MetaTags,
				},
				Time:  m.Time,
				Sum:   m.Value,
				Count: 1,
			})
		}

	case kp.Unprocessed != nil:
		for _, m := range kp.Unprocessed {
			stats = append(stats, &repr.StatRepr{
				Name: &repr.StatName{
					Key:      m.Metric,
					Tags:     m.Tags,
					MetaTags: m.MetaTags,
				},
				Time:  m.Time,
				Min:   repr.CheckFloat(m.Min),
				Max:   repr.CheckFloat(m.Max),
				Last:  repr.CheckFloat(m.Last),
				Sum:   repr.CheckFloat(m.Sum),
				Count: m.Count,
			})
		}

	case kp.Single != nil:
		for _, m := range kp.Single {
			stats = append(stats, &repr.StatRepr{
				Name: &repr.StatName{
					Key:        m.Metric,
					Tags:       m.Tags,
					MetaTags:   m.MetaTags,
					Resolution: m.Resolution,
					Ttl:        m.Ttl,
				},
				Time:  m.Time,
				Min:   repr.CheckFloat(m.Min),
				Max:   repr.CheckFloat(m.Max),
				Last:  repr.CheckFloat(m.Last),
				Sum:   repr.CheckFloat(m.Sum),
				Count: m.Count,
			})
		}
	}
	return stats
}

// SetSendEncoding set message encoding (json, msgpack, protobuf)
func (kp *KMetricList) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc

}

// encodes the 'base' message
func (kp *KMetricList) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()

	if kp != nil && kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.AnyMetricList.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.AnyMetricList.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp)
			return

		}
	}
}

// Length number of bytes in the encoded message
func (kp *KMetricList) Length() int {
	if kp == nil {
		return 0
	}
	kp.ensureEncoded()
	return len(kp.encoded)
}

// Encode convert the message to a byte slice
func (kp *KMetricList) Encode() ([]byte, error) {
	if kp == nil {
		return nil, schemas.ErrMetricIsNil
	}
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

// Decode the message into the internal go object
func (kp *KMetricList) Decode(b []byte) (err error) {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err = kp.AnyMetricList.UnmarshalMsg(b)
	case ENCODE_PROTOBUF:
		err = kp.AnyMetricList.Unmarshal(b)
	default:
		err = json.Unmarshal(b, &kp.AnyMetricList)
	}

	return err
}

// KSingleMetricList kafka list of single metric
type KSingleMetricList struct {
	SingleMetricList

	Type       string
	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KSingleMetricList) Ids() []string {
	ids := []string{}
	for _, m := range kp.SingleMetricList.List {
		ids = append(ids, m.Metric)
	}
	return ids
}

func (kp *KSingleMetricList) Reprs() repr.StatReprSlice {

	stats := repr.StatReprSlice{}
	for _, m := range kp.SingleMetricList.List {
		stats = append(stats, &repr.StatRepr{
			Name: &repr.StatName{
				Key:        m.Metric,
				Tags:       m.Tags,
				MetaTags:   m.MetaTags,
				Resolution: m.Resolution,
				Ttl:        m.Ttl,
			},
			Time:  m.Time,
			Min:   repr.CheckFloat(m.Min),
			Max:   repr.CheckFloat(m.Max),
			Last:  repr.CheckFloat(m.Last),
			Sum:   repr.CheckFloat(m.Sum),
			Count: m.Count,
		})
	}
	return stats
}

func (kp *KSingleMetricList) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.SingleMetricList.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.SingleMetricList.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp.SingleMetricList)

		}
	}
}

func (kp *KSingleMetricList) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc
}

func (kp *KSingleMetricList) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KSingleMetricList) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KSingleMetricList) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.SingleMetricList.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.SingleMetricList.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.SingleMetricList)
	}
}

// KRawMetricList kafka list of raw metrics
type KRawMetricList struct {
	RawMetricList

	Type       string
	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KRawMetricList) Ids() []string {
	ids := []string{}
	for _, m := range kp.RawMetricList.List {
		ids = append(ids, m.Metric)
	}
	return ids
}

func (kp *KRawMetricList) Reprs() repr.StatReprSlice {

	stats := repr.StatReprSlice{}
	for _, m := range kp.RawMetricList.List {
		stats = append(stats, &repr.StatRepr{
			Name: &repr.StatName{
				Key:      m.Metric,
				Tags:     m.Tags,
				MetaTags: m.MetaTags,
			},
			Time:  m.Time,
			Sum:   repr.CheckFloat(m.Value),
			Count: 1,
		})
	}
	return stats
}

func (kp *KRawMetricList) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.RawMetricList.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.RawMetricList.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp.RawMetricList)

		}
	}
}

func (kp *KRawMetricList) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc
}

func (kp *KRawMetricList) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KRawMetricList) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KRawMetricList) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.RawMetricList.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.RawMetricList.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.RawMetricList)
	}
}

// KUnProcessedMetricList kafka list of unprocessed metric
type KUnProcessedMetricList struct {
	UnProcessedMetricList

	Type       string
	encodetype SendEncoding
	encoded    []byte
	err        error
}

func (kp *KUnProcessedMetricList) Ids() []string {
	ids := []string{}
	for _, m := range kp.UnProcessedMetricList.List {
		ids = append(ids, m.Metric)
	}
	return ids
}

func (kp *KUnProcessedMetricList) Reprs() repr.StatReprSlice {

	stats := repr.StatReprSlice{}
	for _, m := range kp.UnProcessedMetricList.List {
		stats = append(stats, &repr.StatRepr{
			Name: &repr.StatName{
				Key:      m.Metric,
				Tags:     m.Tags,
				MetaTags: m.MetaTags,
			},
			Time:  m.Time,
			Min:   repr.CheckFloat(m.Min),
			Max:   repr.CheckFloat(m.Max),
			Last:  repr.CheckFloat(m.Last),
			Sum:   repr.CheckFloat(m.Sum),
			Count: m.Count,
		})
	}
	return stats
}

func (kp *KUnProcessedMetricList) ensureEncoded() {
	defer func() {
		if r := recover(); r != nil {
			kp.err = fmt.Errorf("%v", r)
			kp.encoded = nil
		}
	}()
	if kp.encoded == nil && kp.err == nil {
		switch kp.encodetype {
		case ENCODE_MSGP:
			kp.encoded, kp.err = kp.UnProcessedMetricList.MarshalMsg(nil)
		case ENCODE_PROTOBUF:
			kp.encoded, kp.err = kp.UnProcessedMetricList.Marshal()
		default:
			kp.encoded, kp.err = json.Marshal(kp.UnProcessedMetricList)

		}
	}
}

func (kp *KUnProcessedMetricList) SetSendEncoding(enc SendEncoding) {
	if kp.encodetype != enc {
		kp.encoded = nil // have to reset
		kp.err = nil
	}
	kp.encodetype = enc
}

func (kp *KUnProcessedMetricList) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KUnProcessedMetricList) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

func (kp *KUnProcessedMetricList) Decode(b []byte) error {
	switch kp.encodetype {
	case ENCODE_MSGP:
		_, err := kp.UnProcessedMetricList.UnmarshalMsg(b)
		return err
	case ENCODE_PROTOBUF:
		return kp.UnProcessedMetricList.Unmarshal(b)
	default:
		return json.Unmarshal(b, &kp.UnProcessedMetricList)
	}
}
