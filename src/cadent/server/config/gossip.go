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

/** Gossip config elements **/

package config

import (
	"cadent/server/gossip"
	"fmt"
	"net"
	"os"
)

type GossipConfig struct {
	Enabled       bool   `toml:"enabled" json:"enabled,omitempty"`
	Port          int    `toml:"port" json:"port,omitempty"`                           //
	Mode          string `toml:"mode" json:"mode,omitempty"`                           // local, lan, wan
	Name          string `toml:"name" json:"name,omitempty"`                           // name of this node, otherwise it will pick on (must be unique)
	Bind          string `toml:"bind" json:"bind,omitempty"`                           // bind ip
	AdvertiseBind string `toml:"advertise_address" json:"advertise_address,omitempty"` // bind ip
	Seed          string `toml:"seed" json:"seed,omitempty"`                           // a seed node
}

func getIp() net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var ip net.IP
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
		}
	}
	return ip
}

func getIpFromHost(host string) string {
	addrs, err := net.LookupHost(host)
	if err != nil {
		fmt.Printf("ERRROR: DNS lookup failed for %s: %v\n", host, err)
		return ""
	}
	return addrs[0]
}

func getHostName() string {
	name, err := os.Hostname()
	if err != nil {
		fmt.Printf("ERRROR: Localhost hostname lookup failed: %v\n", err)
		return ""
	}
	return name
}

func (c *GossipConfig) Start() {
	// see if we can join up to the Gossip land
	if c.Enabled {
		if c.Mode == "" {
			c.Mode = "lan"
		}
		if c.Port == 0 {
			c.Port = 8889
		}
		if c.Bind == "" {
			c.Bind = "0.0.0.0"
		}

		if len(c.Name) == 0 {
			c.Name = getHostName()
		}

		// default to the IP if no advertise_address set
		if len(c.AdvertiseBind) == 0 {
			ip := getIpFromHost(c.Name)
			if ip != "" {
				c.AdvertiseBind = ip
			}
		}

		m, err := gossip.Start(c.Mode, c.Port, c.Name, c.Bind, c.AdvertiseBind)
		if err != nil {
			panic("Failed to join gossip: " + err.Error())
		}

		if c.Seed == "" {
			log.Noticef("Starting Gossip (master node) on port:%d seed: master", c.Port)
			gossip.Get().SetJoined(true)
		} else {
			log.Noticef("Joining gossip on port:%d seed: %s", c.Port, c.Seed)
			go m.JoinRetry(c.Seed)
		}
	}
}
