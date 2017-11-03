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

/** Internal Health Server config elements **/

package config

import (
	"cadent/server/pages"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
)

type HealthConfig struct {

	// start a little http server for external health checks and stats probes
	Enabled            bool   `toml:"enabled" json:"enabled,omitempty" yaml:"enabled"`
	HealthServerBind   string `toml:"listen" json:"listen,omitempty" yaml:"listen"`
	HealthServerPoints uint   `toml:"points" json:"points,omitempty" yaml:"points"`
	HealthServerPath   string `toml:"path" json:"path,omitempty" yaml:"path"`
	HealthServerKey    string `toml:"key" json:"key,omitempty" yaml:"key"`
	HealthServerCert   string `toml:"cert" json:"cert,omitempty" yaml:"cert"`

	mux *http.ServeMux
}

func (c *HealthConfig) GetMux() *http.ServeMux {
	if c.mux == nil {
		c.mux = new(http.ServeMux)
	}
	return c.mux
}

func (c *HealthConfig) Start(hashers *ConstHashConfig) {
	// Fire up the http server for stats and healthchecks

	if !c.Enabled {
		log.Notice("NOT starting Status server disabled")
		return
	}

	log.Notice("Starting Status server on %s", c.HealthServerBind)

	fileserve := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		fmt.Fprintf(w, pages.STATS_INDEX_PAGE)
	}

	status := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		fmt.Fprintf(w, "ok")
	}

	freemem := func(w http.ResponseWriter, r *http.Request) {
		debug.FreeOSMemory()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "ok")
	}

	if hashers != nil {
		listservers := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, max-age=0, no-cache")

			s_list := make(map[string][]string)

			for nm, serv := range hashers.Servers {
				if nm == DEFAULT_CONFIG_SECTION {
					continue
				}

				s_list[serv.Name] = []string{
					fmt.Sprintf("/%s", serv.Name),
					fmt.Sprintf("/%s/ping", serv.Name),
					fmt.Sprintf("/%s/ops/status", serv.Name),
					fmt.Sprintf("/%s/stats", serv.Name),
					fmt.Sprintf("/%s/addserver", serv.Name),
					fmt.Sprintf("/%s/purgeserver", serv.Name),
					fmt.Sprintf("/%s/accumulator", serv.Name),
				}
			}
			w.Header().Set("Content-Type", "application/json")
			resbytes, _ := json.Marshal(s_list)
			fmt.Fprintf(w, string(resbytes))
		}
		c.GetMux().HandleFunc("/servers", listservers)
	}
	c.GetMux().HandleFunc("/", fileserve)
	c.GetMux().HandleFunc("/ops/status", status)
	c.GetMux().HandleFunc("/ping", status)
	c.GetMux().HandleFunc("/status", status)
	c.GetMux().HandleFunc("/freemem", freemem)

	// stats stuff + profiler live on the same mux

	if len(c.HealthServerKey) > 0 && len(c.HealthServerCert) > 0 {
		cer, err := tls.LoadX509KeyPair(c.HealthServerCert, c.HealthServerKey)
		if err != nil {
			log.Panicf("Could not start https server: %v", err)
		}
		config := &tls.Config{
			Certificates:             []tls.Certificate{cer},
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		conn, err := tls.Listen("tcp", c.HealthServerBind, config)
		if err != nil {
			log.Panicf("Could not make tls http socket: %s", err)
		}
		go http.Serve(conn, c.mux)

	} else {

		err := http.ListenAndServe(c.HealthServerBind, c.mux)
		if err != nil {
			log.Panicf("Could not start http server %s", c.HealthServerBind)
		}
	}
}
