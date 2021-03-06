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
   Command functions Get data from cache as a list of RawRenderItems
*/

package tcp

import (
	"bytes"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/stats"
	"cadent/server/writers/metrics"
	"fmt"
	"golang.org/x/net/context"
	"strings"
)

var getcByte = []byte("GETC")

type GETCCommand struct {
	BaseCommand
}

func (c *GETCCommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
	args := new(CommandArgs)
	args.args = make(map[string]interface{})

	// strip off the 'GETC ' bits
	stline := string(bytes.TrimSpace(line[(len(c.Code())):]))

	if len(stline) == 0 {
		return args, c.Error(), nil
	}

	bits := strings.Fields(stline)

	args.args["path"] = bits[0]

	if len(bits) > 1 {
		args.args["start"] = bits[1]
	}
	if len(bits) > 2 {
		args.args["end"] = bits[2]
	}
	if len(bits) > 4 {
		args.args["format"] = bits[4]
	}

	if _, ok := args.args["start"]; !ok {
		args.args["start"] = "-1h"
	}
	if _, ok := args.args["end"]; !ok {
		args.args["end"] = "now"
	}

	start, err := metrics.ParseTime(args.args["start"].(string))
	if err != nil {
		return args, c.Error(), fmt.Errorf("Invalid from/start time")
	}

	end, err := metrics.ParseTime(args.args["end"].(string))
	if err != nil {
		return args, c.Error(), fmt.Errorf("Invalid to/end time")
	}

	if end < start {
		start, end = end, start
	}

	args.args["start"] = start
	args.args["end"] = end

	return args, nil, nil
}

func (c *GETCCommand) Do(args *CommandArgs) *CommandResponse {
	ret := NewResponseFromArgs(args)
	gots, err := c.GetCache(args)
	if err != nil {
		ret.errcode = c.Error()
		ret.auxError = err
		return ret
	}

	ret.response = gots
	ret.needTerminator = true //need \r\n
	return ret
}

// GetMetrics :: args better have gone through the ParseCommand otherwise failure
func (c *GETCCommand) GetCache(args *CommandArgs) (smetrics.RawRenderItems, error) {

	pth, ok := args.args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("Invalid path")
	}
	// remove any quotes in case things got spaces in the field
	pth = strings.Replace(pth, `"`, "", -1)
	pth = strings.Replace(pth, `'`, "", -1)

	start, ok := args.args["start"].(int64)
	if !ok {
		return nil, fmt.Errorf("Invalid start")
	}
	end, ok := args.args["end"].(int64)
	if !ok {
		return nil, fmt.Errorf("Invalid end")
	}

	// return is []*smetrics.RawRenderItem, but we need to cast it to the object for easy encoding
	items, err := args.loop.Metrics.CacheRender(context.Background(), pth, start, end, nil)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.tcp.getc.errors", 1)
		return nil, err
	}
	stats.StatsdClientSlow.Incr("reader.tcp.getc.ok", 1)
	return smetrics.RawRenderItems(items), nil
}

func init() {
	cmd := new(GETCCommand)
	cmd.cmd = "GETC"
	cmd.errcode = ErrBadGETC
	RegisterCommand(cmd.cmd, cmd)
}
