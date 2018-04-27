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

package prereg

/*

Parse a Preliminary Regex (or static) Key to backend mapping

Kinda like the carbon-relay where you could do regex redirects on key values, but farms it
internally to the main server bits that then does ConstHashing

This allows keys coming in to be assigned to backend for the const hasher
so that you can say have something that does

NOTE: the the regex will be over the ENTIRE line for a regex based one
for statsd and graphite the key field for each

regex: <key> # aka everything
graphite: <key> <value> <time> <things>
statsd: <key>:<data>

graphite for diamond based stats from all database that spans 3 machines (and hashed between them)
graphite for diamond based stats for all app servers that spans 3 machines (and hashed between them)

loglines that look like java ones to be hashed to 5 nodes
loglines that look like rails ones to be hashed to 5 nodes
etc


Should be of the form

the backends should match the "server" groups from the const hasher

NOTE:::: the ordering is important, first match = first sent

[graphite-regex-map]
default_backend="graphite-proxy"
listen_server="graphite-proxy" # this needs to be an actual SOCKET based server not a backend

    # another backend
    [[graphite-regex-map.map]]
    #what we're regexed from the input
    regex="""^servers..*"""
    backend="graphite-proxy"

    # anything that starts with the prefix (in lue of a more expensive regex)
    [[graphite-regex-map.map]]
    prefix="""servers.main-"""
    backend="graphite-statsd"

    [[graphite-regex-map.map]]
    regex="^servers.*"
    reject=true  # special "reject me" type

[statsd-regex-map]
default_backend="statsd-servers"
listen_server="statsd-servers" # this needs to be an actual SOCKET based server not a backend

    # another backend
    [[statsd-regex-map.map]]
    regex="""^stats.timers..*"""   #what we're regexed from the input
    backend="statsd-servers"

    # anything that starts with the prefix (in lue of a more expesive regex)
    [[statsd-regex-map.map]]
    prefix="stats"
    backend="statsd-statsd"

    [[statsd-regex-map.map]]
    regex="""^stats_count..*"""
    reject=true  # special "reject me" type

..

*/

import (
	logging "gopkg.in/op/go-logging.v1"

	accumulator "cadent/server/accumulator"
	"cadent/server/utils/envreplace"
	"cadent/server/utils/shared"
	"cadent/server/utils/tomlenv"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

const DEFALT_SECTION_NAME = "prereg"

var log = logging.MustGetLogger("prereg")

type ConfigFilter struct {
	NoOp      string `toml:"noop" json:"noop,omitempty" yaml:"noop,omitempty"`
	Prefix    string `toml:"prefix" json:"prefix,omitempty" yaml:"prefix,omitempty"`
	SubString string `toml:"substring" json:"substring,omitempty" yaml:"substring,omitempty"`
	RegEx     string `toml:"regex" json:"regex,omitempty" yaml:"regex,omitempty"`
	IsReject  bool   `toml:"reject" json:"reject,omitempty" yaml:"reject,omitempty"`
	Backend   string `toml:"backend" json:"backend,omitempty" yaml:"backend,omitempty"`
}

type ConfigMap struct {
	DefaultBackEnd    string                        `toml:"default_backend" json:"default_backend" yaml:"default_backend"`
	ListenServer      string                        `toml:"listen_server" json:"listen_server" yaml:"listen_server"`
	ConfigAccumulator accumulator.ConfigAccumulator `toml:"accumulator" json:"accumulator" yaml:"accumulator"` // the accumulator for a given incoming group

	FilterList []ConfigFilter `toml:"map" json:"map,omitempty" yaml:"map,omitempty"`
}

// list of filters
type ListofConfigMaps map[string]ConfigMap

func (l ListofConfigMaps) ParseConfig() (PreRegMap, error) {
	prs := make(PreRegMap)

	for chunk, cfg := range l {
		pr := new(PreReg)

		pr.Name = chunk
		pr.DefaultBackEnd = cfg.DefaultBackEnd
		pr.ListenServer = cfg.ListenServer

		if len(cfg.DefaultBackEnd) == 0 {
			msg := fmt.Sprintf("Need a default_backend for PreReg filters in `%s`", pr.Name)
			log.Critical(msg)
			return nil, fmt.Errorf(msg)
		}
		if len(cfg.ListenServer) == 0 {
			msg := fmt.Sprintf("Need a listen_server for `%s`", pr.Name)
			log.Critical(msg)
			return nil, fmt.Errorf(msg)
		}

		/*if len(cfg.FilterList) == 0 {
			msg := fmt.Sprintf("Need a Some filters for `%s`", pr.Name)
			log.Critical(msg)
			return nil, fmt.Errorf(msg)
		}*/

		if len(cfg.ConfigAccumulator.InputFormat) > 0 {
			cfg.ConfigAccumulator.Name = chunk
			acc, err := cfg.ConfigAccumulator.GetAccumulator()
			if err != nil {
				log.Critical("%s", err)
				return nil, err
			}

			// the verifcation there is really a "backend" that supports the outgoing is done at the server config
			// manager level

			pr.Accumulator = acc
		}

		//deal with maps
		pr.FilterList = make([]FilterItem, len(cfg.FilterList))
		for idx, cmap := range cfg.FilterList {
			if len(cmap.Prefix) > 0 && len(cmap.RegEx) > 0 {
				panic(fmt.Sprintf("Cannot have BOTH `prefix` and `regex` for `%s`", pr.Name))
			}
			if len(cmap.SubString) > 0 && len(cmap.Prefix) > 0 {
				panic(fmt.Sprintf("Cannot have BOTH `prefix` and `substring` for `%s`", pr.Name))
			}
			if len(cmap.RegEx) > 0 && len(cmap.SubString) > 0 {
				panic(fmt.Sprintf("Cannot have BOTH `regex` and `substring` for `%s`", pr.Name))
			}
			if len(cmap.RegEx) == 0 && len(cmap.SubString) == 0 && len(cmap.Prefix) == 0 && len(cmap.NoOp) == 0 {
				panic(fmt.Sprintf("Need `noop`, `prefix`, `regex`, or `substring` for `%s`", pr.Name))
			}

			if len(cmap.Prefix) > 0 {
				pf := new(PrefixFilter)
				pf.Prefix = cmap.Prefix
				pf.backend = cmap.Backend
				if len(cmap.Backend) == 0 {
					pf.backend = pr.DefaultBackEnd
				}
				pf.IsReject = cmap.IsReject
				pf.Init()
				pr.FilterList[idx] = pf

			} else if len(cmap.SubString) > 0 {
				pf := new(SubStringFilter)
				pf.SubString = cmap.SubString
				pf.backend = cmap.Backend
				if len(cmap.Backend) == 0 {
					pf.backend = pr.DefaultBackEnd
				}
				pf.IsReject = cmap.IsReject
				pf.Init()
				pr.FilterList[idx] = pf
			} else if len(cmap.RegEx) > 0 {
				pf := new(RegexFilter)
				pf.RegexString = cmap.RegEx
				pf.backend = cmap.Backend
				if len(cmap.Backend) == 0 {
					pf.backend = pr.DefaultBackEnd
				}
				pf.IsReject = cmap.IsReject
				pf.Init()
				pr.FilterList[idx] = pf
			} else {
				pf := new(NoOpFilter)
				pf.backend = cmap.Backend
				if len(cmap.Backend) == 0 {
					pf.backend = pr.DefaultBackEnd
				}
				pf.IsReject = cmap.IsReject
				pf.Init()
				pr.FilterList[idx] = pf
			}
		}
		prs[pr.Name] = pr
	}

	shared.Set("prereg", l)

	return prs, nil
}

func parseConfFile(fname string) (cfg ListofConfigMaps, err error) {
	cfg = make(ListofConfigMaps)
	suffix_arr := strings.Split(fname, ".")
	suffix := suffix_arr[len(suffix_arr)-1]
	switch suffix {
	case "yaml", "yml":
		yamlFile, err := ioutil.ReadFile(fname)
		if err != nil {
			log.Critical("config file error: err %v ", err)
			return nil, err
		}
		// replace any $ENV{MY_ENV_VAR:defaultvalue}
		replBytes := envreplace.ReplaceEnv(yamlFile)
		err = yaml.Unmarshal(replBytes, &cfg)

		if err != nil {
			log.Critical("yaml parse error: err %v ", err)
			return nil, err
		}
		return cfg, nil
	case "json", "js":
		jsonFile, err := ioutil.ReadFile(fname)
		if err != nil {
			log.Critical("config file error: err %v ", err)
			return nil, err
		}
		// replace any $ENV{MY_ENV_VAR:defaultvalue}
		replBytes := envreplace.ReplaceEnv(jsonFile)
		err = json.Unmarshal(replBytes, &cfg)
		if err != nil {
			log.Critical("json parse error: err %v ", err)
			return nil, err
		}
		return cfg, nil
	default:
		_, err = tomlenv.DecodeFile(fname, &cfg)
		return cfg, err
	}
}

func ParseConfigFile(filename string) (pr PreRegMap, err error) {
	var lcfg ListofConfigMaps
	if lcfg, err = parseConfFile(filename); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}

	return lcfg.ParseConfig()
}

func ParseConfigString(inconf string) (pr PreRegMap, err error) {

	lcfg := make(ListofConfigMaps)
	if _, err := tomlenv.Decode(inconf, &lcfg); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}

	return lcfg.ParseConfig()
}
