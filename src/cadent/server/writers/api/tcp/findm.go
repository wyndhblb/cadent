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
	sindexer "cadent/server/schemas/indexer"
	"cadent/server/stats"
	"fmt"
	"golang.org/x/net/context"
	"strings"
)

type FINDMCommand struct {
	BaseCommand
}

func (c *FINDMCommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
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
		args.args["format"] = bits[1]
	}
	return args, nil, nil
}

func (c *FINDMCommand) Do(args *CommandArgs) *CommandResponse {
	ret := NewResponseFromArgs(args)
	gots, err := c.GetIndex(args)
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
func (c *FINDMCommand) GetIndex(args *CommandArgs) (sindexer.MetricFindItems, error) {

	pth, ok := args.args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("Invalid path")
	}
	// remove any quotes in case things got spaces in the field
	pth = strings.Replace(pth, `"`, "", -1)
	pth = strings.Replace(pth, `'`, "", -1)

	data, err := args.loop.Indexer.Find(context.Background(), pth, nil)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.tcp.findm.errors", 1)
		return nil, err
	}

	stats.StatsdClientSlow.Incr("reader.tcp.findm.ok", 1)
	return data, nil
}

func init() {
	cmd := new(FINDMCommand)
	cmd.cmd = "FINDM"
	cmd.errcode = ErrBadFINDM
	RegisterCommand(cmd.cmd, cmd)
}
