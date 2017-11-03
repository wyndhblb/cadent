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

var helpTXT = []byte(`

  Cadent TCP reader interface

    HELP -- this message
    PING -- PONG
    GETM <path> [<from time?> <to time?> <resolution?> <format?>] -- get raw metric data
    FINDM <path> [<format?>] -- find metric names
    GETC <path> [<from time?> <to time?> <format?>] -- get metric data from cache (if supported)
    GETCB <path> -- get metric binary timeseries from cache (if supported)
    INCA <path> -- Is this metric in the cache? (returns 0,1)
    CLIST [<format?>] -- List of stat names ([]repr.StatName) in the cache currently
    EXIT -- terminate connection

     *NOTE: GETCB can only take an exact "match" of

    [...] == optional in order, but the order maters
    <format> == json (default), msgpack, protobuf
    <path> == something like "stats.host.cpu.total"

    <from>, <to> == A parseable date string or time stamp or relative
        i.e :: 1480731100, -1h, now, 2015-07-01T20:10:30.781Z

      default <from> == -1h
      default <to> == now
      default <resolution> == 1 (or the minimum set in the config)
      default <format> == json

    - parameters are space separated

    - End of any stream/output is indicated by "\r\n"

    - Errors are : '-ERR {code} {message} \r\n'

    - BINARY protocols (GETCB, protobuf, and msgpack) have a length prefix
      So when reading in make sure to add 2 extra bytes for the terminator
      [uint32|bigendian][...content...]\r\n

`)

type HELPCommand struct {
	BaseCommand
}

func (c *HELPCommand) ParseCommand(line []byte) (*CommandArgs, *ErrorCode, error) {
	return new(CommandArgs), nil, nil
}

func (c *HELPCommand) Do(args *CommandArgs) *CommandResponse {
	args.w.Write(helpTXT)
	args.w.Write(TERMINATOR)
	return NewResponseFromArgs(args)
}

func init() {
	cmd := new(HELPCommand)
	cmd.cmd = "HELP"
	RegisterCommand(cmd.cmd, cmd)

	// another common one
	RegisterCommand("?", cmd)
}
