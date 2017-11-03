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

package metrics

// you best have some form of cassandra up
// XXX TODO THIS IS NOT REAL!

import (
	//. "github.com/smartystreets/goconvey/convey"
	"bytes"
	"cadent/server/schemas/repr"
	"cadent/server/utils/options"
	"encoding/json"
	"golang.org/x/net/context"
	"testing"
)

func TestCassandraReader(t *testing.T) {
	// skip it
	return
	prettyprint := func(b []byte) string {
		var out bytes.Buffer
		json.Indent(&out, b, "", "  ")
		return string(out.Bytes())
	}
	ctx := context.Background()
	//some tester strings
	t_config := options.New()
	t_config.Set("dsn", "192.168.99.100")

	reader := NewCassandraMetrics()
	reader.Config(&t_config)

	t_start, _ := ParseTime("-1h")
	t_end, _ := ParseTime("now")
	samples := uint32(10)
	tags := repr.SortingTags{}

	rdata, err := reader.RawRender(ctx, "consthash.zipperwork.local.[a-z]tatsd*", t_start, t_end, tags, samples)
	js, _ := json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.[a-z]tatsd*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

	rdata, err = reader.RawRender(ctx, "consthash.zipperwork.local.graphite-statsd.lrucache.*", t_start, t_end, tags, samples)
	js, _ = json.Marshal(rdata)
	t.Logf("consthash.zipperwork.local.graphite-statsd.lrucache.*: %v", prettyprint(js))
	t.Logf("ERR: %v", err)

}
