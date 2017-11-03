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
   Incache (INCA) Command functions
*/

package tcp

import (
	"bytes"
	"cadent/server/stats"
	"fmt"
	"strings"
)

type INCACommand struct {
	BaseCommand
}

func (c *INCACommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
	args := new(CommandArgs)
	args.args = make(map[string]interface{})

	// strip off the 'INCA ' bits
	stline := string(bytes.TrimSpace(line[(len(c.Code())):]))

	if len(stline) == 0 {
		return args, c.Error(), nil
	}

	bits := strings.Fields(stline)

	args.args["path"] = bits[0]

	if len(bits) > 1 {
		args.args["format"] = bits[1]
	}

	return args, nil, nil
}

func (c *INCACommand) Do(args *CommandArgs) *CommandResponse {
	ret := NewResponseFromArgs(args)
	gots, err := c.InCache(args)
	if err != nil {
		ret.errcode = c.Error()
		ret.auxError = err
		return ret
	}

	ret.response = 0
	if gots {
		ret.response = 1
	}
	ret.needTerminator = true
	return ret
}

// GetMetrics :: args better have gone through the ParseCommand otherwise failure
func (c *INCACommand) InCache(args *CommandArgs) (bool, error) {

	pth, ok := args.args["path"].(string)
	if !ok {
		return false, fmt.Errorf("Invalid path")
	}

	// return is []*smetrics.RawRenderItem, but we need to cast it to the object for easy encoding
	items, err := args.loop.Metrics.InCache(pth, nil)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.tcp.inca.errors", 1)
		return false, err
	}
	stats.StatsdClientSlow.Incr("reader.tcp.inca.ok", 1)
	return items, nil
}

func init() {
	cmd := new(INCACommand)
	cmd.cmd = "INCA"
	cmd.errcode = ErrBadINCA
	RegisterCommand(cmd.cmd, cmd)
}
