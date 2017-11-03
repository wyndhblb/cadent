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
   Fire up an TCP server for an http interface to the

   metrics/indexer interfaces

   example config
   [graphite-proxy-map.accumulator.tcp.api]
        listen = "0.0.0.0:8083"
            [graphite-proxy-map.accumulator.tcp.api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            [graphite-proxy-map.accumulator.tcp.api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"

   Much like redis or memcache, this supports simple "command -> response" on a socket

   METRIC string from to step format(optional){json,msgpack,protobuf:json}
   FIND string format(optional){json,msgpack,protobuf|default:json}


*/

package tcp

import (
	"fmt"
	"io"
)

const (
	// start at 10
	ErrReadError = 100
	ErrShutdown  = 101

	ErrNotFound    = 404
	ErrBadCommand  = 405
	ErrBadGETM     = 406
	ErrBadFINDM    = 407
	ErrBadGETC     = 408
	ErrBadGETCB    = 409
	ErrBadINCA     = 410
	ErrBadCLIST    = 411
	ErrUnavailable = 499

	ErrInternal             = 500
	ErrNoMsgPack            = 501
	ErrMsgPackMarshalError  = 502
	ErrNoProtobuf           = 503
	ErrProtobufMarshalError = 504

	ErrPanic = 999
	ErrOther = 998
)

type ErrorCode struct {
	Code int
	Msg  string
}

func (e *ErrorCode) Write(buf io.Writer) {
	fmt.Fprintf(buf, "%d %s", e.Code, e.Msg)
}

var ErrorMap map[int]*ErrorCode

func init() {
	ErrorMap = make(map[int]*ErrorCode)
	ErrorMap[ErrBadCommand] = &ErrorCode{
		Code: ErrBadCommand,
		Msg:  "Invalid Command",
	}
	ErrorMap[ErrShutdown] = &ErrorCode{
		Code: ErrShutdown,
		Msg:  "Shutting down, bye bye",
	}
	ErrorMap[ErrBadGETM] = &ErrorCode{
		Code: ErrBadGETM,
		Msg:  "Invalid `GETM <string> [<from?> <to?> <format?>]`",
	}
	ErrorMap[ErrBadFINDM] = &ErrorCode{
		Code: ErrBadFINDM,
		Msg:  "Invalid `FINDM <string> [<format?>]`",
	}
	ErrorMap[ErrBadGETC] = &ErrorCode{
		Code: ErrBadGETC,
		Msg:  "Invalid `GETC <string> [<from?> <to?> <format?>]`",
	}
	ErrorMap[ErrBadGETCB] = &ErrorCode{
		Code: ErrBadGETCB,
		Msg:  "Invalid `GETCB <string>`",
	}
	ErrorMap[ErrBadINCA] = &ErrorCode{
		Code: ErrBadINCA,
		Msg:  "Invalid `INCA <string>`",
	}
	ErrorMap[ErrBadCLIST] = &ErrorCode{
		Code: ErrBadCLIST,
		Msg:  "Invalid `CLIST [<format?>]`",
	}
	ErrorMap[ErrNoMsgPack] = &ErrorCode{
		Code: ErrNoMsgPack,
		Msg:  "Object does not have a msgpack encoder",
	}
	ErrorMap[ErrMsgPackMarshalError] = &ErrorCode{
		Code: ErrMsgPackMarshalError,
		Msg:  "Msgpack MarshalBinary error",
	}
	ErrorMap[ErrInternal] = &ErrorCode{
		Code: ErrInternal,
		Msg:  "Internal Server Error",
	}
	ErrorMap[ErrUnavailable] = &ErrorCode{
		Code: ErrUnavailable,
		Msg:  "Service not available",
	}
	ErrorMap[ErrNotFound] = &ErrorCode{
		Code: ErrNotFound,
		Msg:  "Not found",
	}
	ErrorMap[ErrNoProtobuf] = &ErrorCode{
		Code: ErrNoProtobuf,
		Msg:  "Object does not have a protobuf encoder",
	}
	ErrorMap[ErrProtobufMarshalError] = &ErrorCode{
		Code: ErrProtobufMarshalError,
		Msg:  "Protobuf Marshal error",
	}
	ErrorMap[ErrPanic] = &ErrorCode{
		Code: ErrPanic,
		Msg:  "Server fault, panic",
	}
	ErrorMap[ErrOther] = &ErrorCode{
		Code: ErrOther,
		Msg:  "Other Error",
	}
}

func GetError(code int) *ErrorCode {
	if gt, ok := ErrorMap[code]; ok {
		return gt
	}
	return ErrorMap[ErrOther]
}
