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
	Simple echo to a file/stdout/stderr
*/

package metrics

import (
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/DataDog/zstd"
	"github.com/golang/snappy"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"io"
	"os"
)

/****************** Interfaces *********************/
type EchoMetrics struct {
	WriterBase
	log      *logging.Logger
	file     io.Writer
	jencoder *json.Encoder
	compress string
}

func NewEchoMetrics() *EchoMetrics {
	ec := new(EchoMetrics)
	ec.shutitdown = false
	ec.log = logging.MustGetLogger("writers.echo.metrics")
	return ec
}

func (ec *EchoMetrics) Config(conf *options.Options) error {
	ec.options = conf
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (stdout,stderr,filename) is needed for echo config")
	}

	gTag := conf.String("tags", "")
	if len(gTag) > 0 {
		ec.staticTags = repr.SortingTagsFromString(gTag)
	}

	var f io.Writer
	switch dsn {
	case "stdout":
		f = os.Stdout
	case "stderr":
		f = os.Stderr
	default:
		f, err = os.OpenFile(dsn, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			ec.log.Errorf("Failed to open file: %v", err)
			return err
		}
	}
	ec.compress = conf.String("compress", "none")
	switch ec.compress {
	case "gzip":
		ec.file = gzip.NewWriter(f)
	case "snappy":
		ec.file = snappy.NewBufferedWriter(f)
	case "zstd":
		ec.file = zstd.NewWriterLevel(f, zstd.BestCompression)
	default:
		ec.file = f
	}

	ec.jencoder = json.NewEncoder(ec.file)

	return nil
}

// Driver name
func (ec *EchoMetrics) Driver() string {
	return "echo"
}

// Start a noop
func (ec *EchoMetrics) Start() {}

func (ec *EchoMetrics) Stop() {
	ec.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if f, ok := ec.file.(*os.File); ok {
			f.Close()
		}
		if f, ok := ec.file.(*gzip.Writer); ok {
			f.Close()
		}
	})
}

func (ec *EchoMetrics) Write(stat *repr.StatRepr) error {
	return ec.WriteWithOffset(stat, nil)
}

func (ec *EchoMetrics) WriteWithOffset(stat *repr.StatRepr, offset *metrics.OffsetInSeries) error {

	if ec.shutitdown {
		return nil
	}

	stat.Name.MergeMetric2Tags(ec.staticTags)
	ec.indexer.Write(*stat.Name) // to the indexer
	var err error
	if offset != nil {
		o := map[string]interface{}{
			"stat":   stat,
			"offset": offset,
		}
		err = ec.jencoder.Encode(o)
	} else {
		err = ec.jencoder.Encode(stat)
	}
	if err != nil {
		ec.log.Errorf("Error in writing: %v", err)
		stats.StatsdClientSlow.Incr("writer.echo.metrics.errors", 1)
	} else {
		stats.StatsdClientSlow.Incr("writer.echo.metrics.writes", 1)
	}

	return err
}

/**** READER ***/
// needed to match interface, but we obviously cannot do this

func (ec *EchoMetrics) RawRender(context.Context, string, int64, int64, repr.SortingTags, uint32) ([]*metrics.RawRenderItem, error) {
	return []*metrics.RawRenderItem{}, ErrWillNotBeimplemented
}
func (ec *EchoMetrics) CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*metrics.RawRenderItem, error) {
	return nil, ErrWillNotBeimplemented
}
func (ec *EchoMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*metrics.TotalTimeSeries, error) {
	return nil, ErrWillNotBeimplemented
}
