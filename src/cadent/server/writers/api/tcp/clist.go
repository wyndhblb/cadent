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
   Cache List Command functions
*/

package tcp

import (
	"bytes"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"strings"
)

type CLISTCommand struct {
	BaseCommand
}

func (c *CLISTCommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
	args := new(CommandArgs)
	args.args = make(map[string]interface{})

	// strip off the 'CLIST ' bits
	stline := string(bytes.TrimSpace(line[(len(c.Code())):]))

	bits := strings.Fields(stline)

	if len(bits) > 0 {
		args.args["format"] = bits[0]
	}

	return args, nil, nil
}

func (c *CLISTCommand) Do(args *CommandArgs) *CommandResponse {
	ret := NewResponseFromArgs(args)
	gots, err := c.CacheList(args)
	if err != nil {
		ret.errcode = c.Error()
		ret.auxError = err
		return ret
	}

	ret.response = gots
	ret.needTerminator = true
	return ret
}

// GetMetrics :: args better have gone through the ParseCommand otherwise failure
func (c *CLISTCommand) CacheList(args *CommandArgs) (repr.StatNameSlice, error) {

	// return is []*smetrics.RawRenderItem, but we need to cast it to the object for easy encoding
	items, err := args.loop.Metrics.CacheList()
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.tcp.clist.errors", 1)
		return nil, err
	}
	stats.StatsdClientSlow.Incr("reader.tcp.clist.ok", 1)
	return repr.StatNameSlice(items), nil
}

func init() {
	cmd := new(CLISTCommand)
	cmd.cmd = "CLIST"
	cmd.errcode = ErrBadCLIST
	RegisterCommand(cmd.cmd, cmd)
}
