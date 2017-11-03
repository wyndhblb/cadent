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
   "Discovery" .. a place to let something know that "we are here"

   A simple Zookeeper Key, or consul key, or "whatever" key
*/

package discovery

import (
	sapi "cadent/server/schemas/api"
	"cadent/server/utils/options"
	"cadent/server/writers/api"
)

// Discover interface
type Discover interface {

	// Config the module
	Config(opts options.Options) error

	// Start ourselves
	Start() error

	// Stop (aka degregister) ourselves
	Stop()

	Register(info *api.InfoData) error
	DeRegister() error
}

// Lister get all the elements "alive" in our registered pile
type Lister interface {

	// Config configure the lister
	Config(options.Options) error

	// Start ourselves
	Start() error

	// Stop ourselves
	Stop()

	// ApiHosts Hosts grab all the hosts in the discover pile
	ApiHosts() *sapi.DiscoverHosts

	// ForcePullHosts force a re-grab of
	ForcePullHosts() (*sapi.DiscoverHosts, error)
}
