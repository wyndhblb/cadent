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
	Make it easy to get a new time series algo .. not exactly the optimal
	"registry" technique but it works
*/

package series

import (
	"fmt"
	"github.com/golang/snappy"
)

const (
	// GOB format
	GOB uint8 = iota + 1
	// ZIPGOB format
	ZIPGOB
	//JSON format
	JSON
	// PRTOBUG format
	PROTOBUF
	// GORILLA format
	GORILLA
	// MSGPACK format
	MSGPACK
	// BINC format
	BINC
	// CBOR format
	CBOR
	// REPR (native) format
	REPR
)
const (
	// snappy compressed gob
	SNAPPY_GOB uint8 = iota + 20
	// snappy compressed zipgob (rather silly, but here for completeness)
	SNAPPY_ZIPGOB
	// snappy compressed json
	SNAPPY_JSON
	// snappy compressed protobuf
	SNAPPY_PROTOBUF
	// snappy compressed gorilla (rather silly, but here for completeness)
	SNAPPY_GORILLA
	// snappy compressed msgpack
	SNAPPY_MSGPACK
	// snappy compressed binc
	SNAPPY_BINC
	// snappy compressed cbor
	SNAPPY_CBOR
	// snappy compressed (repr == json)
	SNAPPY_REPR
)

// IsCompressed is the series internally compressed (currently not in use)
func IsCompressed(id uint8) bool {
	return id >= 20
}

// NameFromId given the constants, what is the string name for it
func NameFromId(id uint8) string {
	// snappy compressed maybe
	use_id := id
	if id >= 20 {
		use_id = use_id - 20
	}
	switch use_id {
	case GOB:
		return "gob"
	case ZIPGOB:
		return "zipbob"
	case JSON:
		return "json"
	case PROTOBUF:
		return "protobuf"
	case GORILLA:
		return "gorilla"
	case MSGPACK:
		return "msgpack"
	case BINC:
		return "binc"
	case CBOR:
		return "cbor"
	case REPR:
		return "repr"
	default:
		return ""
	}
}

// IdFromName given a string name what is the id for it
func IdFromName(nm string) uint8 {
	switch nm {
	case "gob":
		return GOB
	case "zipgob":
		return ZIPGOB
	case "json":
		return JSON
	case "protobuf":
		return PROTOBUF
	case "gorilla":
		return GORILLA
	case "msgpack":
		return MSGPACK
	case "binc":
		return BINC
	case "cbor":
		return CBOR
	case "repr":
		return REPR
	default:
		return 0
	}
}

// NewTimeSeries create a new timseries from a string name object starting at T0
func NewTimeSeries(name string, t0 int64, options *Options) (TimeSeries, error) {
	if options == nil {
		options = NewDefaultOptions()
	}
	switch {
	case name == "binary" || name == "gob":
		return NewGobTimeSeries(t0, options), nil
	case name == "zipbinary" || name == "zipgob":
		return NewZipGobTimeSeries(t0, options), nil
	case name == "json":
		return NewJsonTimeSeries(t0, options), nil
	case name == "protobuf":
		return NewProtobufTimeSeries(t0, options), nil
	case name == "gorilla":
		return NewGoriallaTimeSeries(t0, options), nil
	case name == "msgpack":
		return NewMsgpackTimeSeries(t0, options), nil
	case name == "binc":
		options.Handler = "binc"
		return NewCodecTimeSeries(t0, options), nil
	case name == "cbor":
		options.Handler = "cbor"
		return NewCodecTimeSeries(t0, options), nil
	case name == "repr":
		return NewReprTimeSeries(t0, options), nil
	default:
		return nil, fmt.Errorf("Invalid time series type `%s`", name)
	}
}

// NewIter create a new time series iterator from the bytes blob
func NewIter(name string, data []byte) (TimeSeriesIter, error) {

	switch {
	case name == "binary" || name == "gob":
		return NewGobIterFromBytes(data)
	case name == "zipbinary" || name == "zipgob":
		return NewZipGobIterFromBytes(data)
	case name == "json":
		return NewJsonIterFromBytes(data)
	case name == "protobuf":
		return NewProtobufIterFromBytes(data)
	case name == "gorilla":
		return NewGorillaIterFromBytes(data)
	case name == "msgpack":
		return NewMsgpackIterFromBytes(data)
	case name == "binc" || name == "cbor":
		return NewCodecIterFromBytes(data)
	case name == "repr":
		return NewReprIterFromBytes(data)
	default:
		return nil, fmt.Errorf("Invalid time series type `%s`", name)
	}
}

func DecompressBytes(seriestype uint8, data []byte) ([]byte, error) {

	// not compressed
	if !IsCompressed(seriestype) {
		return data, nil
	}

	outs := make([]byte, 0)
	return snappy.Decode(outs, data)
}

func CompressBytes(seriestype uint8, data []byte) ([]byte, error) {

	// not to be compressed
	if !IsCompressed(seriestype) {
		return data, nil
	}

	return snappy.Encode(nil, data), nil
}
