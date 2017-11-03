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
   Command functions
*/

package tcp

import (
	"bytes"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/stats"
	"cadent/server/writers/metrics"
	"fmt"
	"golang.org/x/net/context"
	"strconv"
	"strings"
)

type GETMCommand struct {
	BaseCommand
}

func (c *GETMCommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
	args := new(CommandArgs)
	args.args = make(map[string]interface{})

	// strip off the 'GETM ' bits
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
	if len(bits) > 3 {
		args.args["step"] = bits[3]
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
	if _, ok := args.args["step"]; !ok {
		args.args["step"] = "1"
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

	step := uint32(0)
	if len(args.args["step"].(string)) > 0 {
		tstep, err := (strconv.ParseUint(args.args["step"].(string), 10, 32))
		if err != nil {
			return args, c.Error(), fmt.Errorf("Invalid resolution")
		}
		step = uint32(tstep)
	}
	args.args["start"] = start
	args.args["end"] = end
	args.args["step"] = step

	return args, nil, nil
}

func (c *GETMCommand) Do(args *CommandArgs) *CommandResponse {
	ret := NewResponseFromArgs(args)
	gots, err := c.GetMetrics(args)
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
func (c *GETMCommand) GetMetrics(args *CommandArgs) (smetrics.RawRenderItems, error) {

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
	step, ok := args.args["step"].(uint32)
	if !ok {
		return nil, fmt.Errorf("Invalid step")
	}
	stp := args.loop.minResolution(start, end, step)

	// return is []*smetrics.RawRenderItem, but we need to cast it to the object for easy encoding
	items, err := args.loop.Metrics.RawRender(context.Background(), pth, start, end, nil, stp)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.tcp.getm.errors", 1)
		return nil, err
	}
	stats.StatsdClientSlow.Incr("reader.tcp.getm.ok", 1)
	return smetrics.RawRenderItems(items), nil
}

func init() {
	cmd := new(GETMCommand)
	cmd.cmd = "GETM"
	cmd.errcode = ErrBadGETCB
	RegisterCommand(cmd.cmd, cmd)
}
