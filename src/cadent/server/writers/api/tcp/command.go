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
	"net"
	"strings"
	"sync"
)

// CommandArgs result after the line has been parsed
type CommandArgs struct {
	loop *TCPLoop
	w    net.Conn
	args map[string]interface{}
}

// CommandResponse response from the command post run
type CommandResponse struct {
	loop           *TCPLoop
	w              net.Conn
	errcode        *ErrorCode
	auxError       error
	args           map[string]interface{}
	response       interface{}
	needTerminator bool
	isBinary       bool // for binary outs, we need to add a mime like ------random------\ndata\n------random------\r\n
}

// NewResponseFromArgs copies the relevant CommandArgs things in to a new CommandResponse
func NewResponseFromArgs(args *CommandArgs) *CommandResponse {
	c := new(CommandResponse)
	c.loop = args.loop
	c.w = args.w
	c.args = args.args
	c.response = nil
	return c
}

// Command interface of a generic command to run
type Command interface {
	Code() []byte
	CodeString() string
	ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error)
	Do(*CommandArgs) *CommandResponse
}

// BaseCommand root command
type BaseCommand struct {
	errcode int
	cmd     string
}

// Code command code in bytes
func (c *BaseCommand) Code() []byte {
	return []byte(c.cmd)
}

// Code command code as string
func (c *BaseCommand) CodeString() string {
	return c.cmd
}

func (c *BaseCommand) ErrorCode() int {
	return c.errcode
}

func (c *BaseCommand) Error() *ErrorCode {
	return GetError(c.ErrorCode())
}

var commandMap map[string]Command
var cmdMu sync.RWMutex

func RegisterCommand(name string, c Command) {
	cmdMu.Lock()
	defer cmdMu.Unlock()
	if commandMap == nil {
		commandMap = make(map[string]Command)
	}
	nm := strings.ToLower(name)
	commandMap[nm] = c
}

// GetCommand get a command obj from the registry
// NOTE: there is no lock here, it is assumed that the registry is populated by init()'s in the
// packages
func GetCommand(name string) (Command, *ErrorCode) {
	nm := strings.ToLower(name) // in case of weird cases
	if cmd, ok := commandMap[nm]; ok {
		return cmd, nil
	}
	return nil, GetError(ErrBadCommand)
}
