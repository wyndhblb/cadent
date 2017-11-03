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

var closeTXT = []byte("Bye Bye...\r\n")

type CLOSECommand struct {
	BaseCommand
}

func (c *CLOSECommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
	return new(CommandArgs), nil, nil
}

func (c *CLOSECommand) Do(args *CommandArgs) *CommandResponse {
	args.w.Write(closeTXT)
	args.w.Close()
	args.w = nil

	return NewResponseFromArgs(args)
}

func init() {
	cmd := new(CLOSECommand)
	cmd.cmd = "CLOSE"
	RegisterCommand(cmd.cmd, cmd)

	// some other common ones
	RegisterCommand("exit", cmd)
	RegisterCommand("quit", cmd)
	RegisterCommand("EXIT", cmd)
	RegisterCommand("QUIT", cmd)
}
