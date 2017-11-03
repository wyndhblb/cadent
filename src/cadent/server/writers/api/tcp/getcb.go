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
   Command functions to get series (binary blob) from cache
*/

package tcp

import (
	"bytes"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/stats"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

type GETCBCommand struct {
	BaseCommand
}

func (c *GETCBCommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
	args := new(CommandArgs)
	args.args = make(map[string]interface{})

	// strip off the 'GETCB ' bits
	stline := string(bytes.TrimSpace(line[(len(c.Code())):]))

	if len(stline) == 0 {
		return args, c.Error(), nil
	}

	bits := strings.Fields(stline)

	args.args["path"] = bits[0]

	return args, nil, nil
}

func (c *GETCBCommand) Do(args *CommandArgs) *CommandResponse {
	ret := NewResponseFromArgs(args)
	gots, err := c.GetCacheSeries(args)
	if err != nil {
		ret.errcode = c.Error()
		ret.auxError = err
		return ret
	}
	ret.response = nil         // we dumped it already
	ret.needTerminator = false //need \r\n

	if gots == nil || gots.Series == nil {
		ret.errcode = GetError(ErrNotFound)
		return ret
	}

	// this is just a binary old blob, no "formatting"
	bs, err := gots.Series.MarshalBinary()
	if err != nil {
		ret.errcode = c.Error()
		ret.auxError = err
		return ret
	}
	binary.Write(ret.w, binary.BigEndian, uint32(len(bs)))
	ret.w.Write(bs)

	return ret
}

// GetMetrics :: args better have gone through the ParseCommand otherwise failure
func (c *GETCBCommand) GetCacheSeries(args *CommandArgs) (*smetrics.TotalTimeSeries, error) {

	pth, ok := args.args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("Invalid path")
	}
	// remove any quotes in case things got spaces in the field
	pth = strings.Replace(pth, `"`, "", -1)
	pth = strings.Replace(pth, `'`, "", -1)

	start := time.Now().Add(time.Duration(-1 * time.Minute)).Unix()
	end := time.Now().Unix()

	// return is []*smetrics.RawRenderItem, but we need to cast it to the object for easy encoding
	items, err := args.loop.Metrics.CachedSeries(pth, start, end, nil)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.tcp.getcb.errors", 1)
		return nil, err
	}
	stats.StatsdClientSlow.Incr("reader.tcp.getcb.ok", 1)
	return items, nil
}

func init() {
	cmd := new(GETCBCommand)
	cmd.cmd = "GETCB"
	cmd.errcode = ErrBadGETCB
	RegisterCommand(cmd.cmd, cmd)
}
