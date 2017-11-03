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
	Cassandra Metric schema injector
*/

package metrics

import (
	logging "gopkg.in/op/go-logging.v1"

	"bytes"
	"cadent/server/schemas/repr"
	"fmt"
	"github.com/gocql/gocql"
	"strings"
	"sync"
	"text/template"
)

// syncPool for the "map" type we need to bind with
var mapPool sync.Pool
var subMapPool sync.Pool

// base object that we need to "select" from the short names are there for cassandra improvments
type cqlMapPoint struct {
	Min   float64 `cql:"mi"`
	Max   float64 `cql:"mx"`
	Sum   float64 `cql:"s"`
	Last  float64 `cql:"l"`
	Count int64   `cql:"c"`
}

type cqlMapPointTimePoint map[int64]cqlMapPoint

func getSubMapItem(stat *repr.StatRepr) *cqlMapPoint {
	x := subMapPool.Get()
	if x == nil {
		mper := new(cqlMapPoint)
		mper.Sum = float64(stat.Sum)
		mper.Min = float64(stat.Min)
		mper.Max = float64(stat.Max)
		mper.Last = float64(stat.Last)
		mper.Count = stat.Count
		return mper
	}
	mper := x.(*cqlMapPoint)
	mper.Sum = float64(stat.Sum)
	mper.Min = float64(stat.Min)
	mper.Max = float64(stat.Max)
	mper.Last = float64(stat.Last)
	mper.Count = stat.Count
	return mper
}

func putSubMapItem(item map[string]interface{}) {
	subMapPool.Put(item)
}

func getMapItem(stat *repr.StatRepr) map[int64]*cqlMapPoint {
	x := mapPool.Get()
	if x == nil {
		mper := make(map[int64]*cqlMapPoint)
		mper[stat.ToTime().UnixNano()] = getSubMapItem(stat)
		return mper
	}
	mper := x.(map[int64]*cqlMapPoint)
	for k := range mper {
		delete(mper, k)
	}
	mper[stat.ToTime().UnixNano()] = getSubMapItem(stat)
	return mper
}

func putMapItem(item map[int64]*cqlMapPoint) {
	for _, v := range item {
		subMapPool.Put(v)
	}
	mapPool.Put(item)
}

// syncPool for the "set" type we need to bind with
var setPool sync.Pool
var subSetPool sync.Pool

// base object that we need to "select" from the short names are there for cassandra improvements
type cqlSetPoint struct {
	Time  int64   `cql:"t"`
	Min   float64 `cql:"mi"`
	Max   float64 `cql:"mx"`
	Sum   float64 `cql:"s"`
	Last  float64 `cql:"l"`
	Count int64   `cql:"c"`
}

func newCqlSetPoint(stat *repr.StatRepr) *cqlSetPoint {
	mper := new(cqlSetPoint)
	mper.Sum = float64(stat.Sum)
	mper.Min = float64(stat.Min)
	mper.Max = float64(stat.Max)
	mper.Last = float64(stat.Last)
	mper.Count = stat.Count
	mper.Time = stat.Time
	return mper
}

type cqlSetPointTimePoint []*cqlSetPoint

func (v cqlSetPointTimePoint) Len() int           { return len(v) }
func (v cqlSetPointTimePoint) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v cqlSetPointTimePoint) Less(i, j int) bool { return v[i].Time < v[j].Time }

func getSubSetItem(stat *repr.StatRepr) *cqlSetPoint {
	x := subSetPool.Get()
	if x == nil {
		return newCqlSetPoint(stat)
	}
	mper := x.(*cqlSetPoint)
	mper.Sum = float64(stat.Sum)
	mper.Min = float64(stat.Min)
	mper.Max = float64(stat.Max)
	mper.Last = float64(stat.Last)
	mper.Count = stat.Count
	mper.Time = stat.Time
	return mper
}

func putSubSetItem(item []interface{}) {
	subSetPool.Put(item)
}

func getSetItem() []*cqlSetPoint {
	x := setPool.Get()
	if x == nil {
		return make([]*cqlSetPoint, 0)
	}
	mper := x.([]*cqlSetPoint)
	return mper[:0]
}

func putSetItem(item []*cqlSetPoint) {
	for _, v := range item {
		subSetPool.Put(v)
	}
	setPool.Put(item)
}

// cassandra wants a [stat] for the pts = pts + [stat] not just a raw stat
func getSetItemStat(stat *repr.StatRepr) []*cqlSetPoint {
	x := setPool.Get()
	if x == nil {
		mper := make([]*cqlSetPoint, 1)
		mper[0] = getSubSetItem(stat)
		return mper
	}
	mper := x.([]*cqlSetPoint)

	mper = mper[:0]
	mper = append(mper, getSubSetItem(stat))
	return mper
}

const CASSANDRA_METRICS_FLAT_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_point (
	mx double,
	mi double,
	s double,
	l double,
	c int
);

==SPLIT==

CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.metric_id_res (
	id ascii,
	res int
);
==SPLIT==

CREATE TABLE IF NOT EXISTS  {{.Keyspace}}.{{.MetricsTable}} (
	mid frozen<metric_id_res>,
	t bigint,
	pt frozen<metric_point>,
	PRIMARY KEY (mid, t)
) WITH COMPACT STORAGE
	AND CLUSTERING ORDER BY (t ASC)
	AND {{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_FLAT_PER_RES_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_point (
	mx double,
	mi double,
	s double,
	l double,
	c int
);
==SPLIT==
CREATE TABLE IF NOT EXISTS  {{.Keyspace}}.{{.MetricsTable}}_{{.Resolution}}s (
	id ascii,
	t bigint,
	pt frozen<metric_point>,
	PRIMARY KEY (id, time)
) WITH COMPACT STORAGE
	AND CLUSTERING ORDER BY (t ASC)
	AND {{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_FLATMAP_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_point (
	mx double,
	mi double,
	s double,
	l double,
	c int
);

==SPLIT==

CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.metric_id_res (
	id ascii,
	res int
);
==SPLIT==

CREATE TABLE IF NOT EXISTS  {{.Keyspace}}.{{.MetricsTable}} (
	uid ascii,
	res int,
	slab ascii,
	ord ascii,
	points map<bigint, frozen<metric_point>>,
	PRIMARY KEY ((uid, res, slab), ord)
) WITH CLUSTERING ORDER BY (ord ASC) AND
{{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_FLATMAP_PER_RES_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_point (
	mx double,
	mi double,
	s double,
	l double,
	c int
);
==SPLIT==
CREATE TABLE IF NOT EXISTS  {{.Keyspace}}.{{.MetricsTable}}_{{.Resolution}}s (
	uid ascii,
	slab ascii,
	ord ascii,
	pts map<bigint, frozen<metric_point>>,
	PRIMARY KEY ((uid, slab), ord)
) WITH CLUSTERING ORDER BY (ord ASC) AND
   {{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_FLATSET_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_set_point (
	t bigint,
	mx double,
	mi double,
	s double,
	l double,
	c int
);

==SPLIT==

CREATE TABLE IF NOT EXISTS  {{.Keyspace}}.{{.MetricsTable}} (
    uid ascii,
    res int,
    slab ascii,
    ord ascii,
    pts list<frozen<metric_set_point>>,
    PRIMARY KEY ((uid, res, slab), ord)
) WITH CLUSTERING ORDER BY (ord ASC) AND
	{{ if .CassVersionTwo }}
    compaction = {
	'class': 'DateTieredCompactionStrategy',
	'min_threshold': '12',
	'max_threshold': '32',
	'max_sstable_age_days': '1',
	'base_time_seconds': '50'
    }
    {{ end }}
    {{ if .CassVersionThree }}
    compaction = {
	'class': 'TimeWindowCompactionStrategy',
	'compaction_window_unit': 'DAYS',
	'compaction_window_size': '1'
    }
    {{ end }}
    AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
    AND gc_grace_seconds=86400
    AND speculative_retry = 'NONE'
    AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_FLATSET_PER_RES_TEMPLATE = `

CREATE TYPE IF NOT EXISTS {{.Keyspace}}.metric_set_point (
	t bigint,
	mx double,
	mi double,
	s double,
	l double,
	c int
);
==SPLIT==
CREATE TABLE IF NOT EXISTS  {{.Keyspace}}.{{.MetricsTable}}_{{.Resolution}}s (
	uid ascii,
	slab ascii,
	ord ascii,
	pts list<frozen<metric_set_point>>,
	PRIMARY KEY ((uid, slab), ord)
) WITH CLUSTERING ORDER BY (ord ASC) AND
   {{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_BLOB_TEMPLATE = `
CREATE TYPE IF NOT EXISTS  {{.Keyspace}}.metric_id_res (
            id ascii,
            res int
        );
==SPLIT==
CREATE TABLE IF NOT EXISTS {{.Keyspace}}.{{.MetricsTable}} (
	mid frozen<metric_id_res>,
	etime bigint,
	stime bigint,
	ptype int,
	points blob,
	PRIMARY KEY (mid, etime, stime, ptype)
) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (etime ASC)
	AND {{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_BLOB_PER_RES_TEMPLATE = `
CREATE TABLE IF NOT EXISTS {{.Keyspace}}.{{.MetricsTable}}_{{.Resolution}}s (
	id ascii,
	etime bigint,
	stime bigint,
	ptype int,
	points blob,
	PRIMARY KEY (id, etime, stime, ptype)
) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (etime ASC)
	AND {{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

const CASSANDRA_METRICS_LOG_TEMPLATE = `
CREATE TABLE IF NOT EXISTS {{.Keyspace}}.{{.LogTable}}_{{.WriteIndex}}_{{.Resolution}}s (
	seq bigint,
	ts bigint,
	pts blob,
	PRIMARY KEY (seq, ts)
) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (ts ASC)
	AND {{ if .CassVersionTwo }}
	compaction = {
		'class': 'DateTieredCompactionStrategy',
		'min_threshold': '12',
		'max_threshold': '32',
		'max_sstable_age_days': '1',
		'base_time_seconds': '50'
	}
	{{ end }}
	{{ if .CassVersionThree }}
	compaction = {
		'class': 'TimeWindowCompactionStrategy',
		'compaction_window_unit': 'DAYS',
		'compaction_window_size': '1'
	}
	{{ end }}
	AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'}
	AND gc_grace_seconds=86400
	AND speculative_retry = 'NONE'
	AND dclocal_read_repair_chance = 0.0;`

/****************** Interfaces *********************/
type cassandraMetricsSchema struct {
	conn             *gocql.Session
	Keyspace         string
	MetricsTable     string
	Resolutions      [][]int
	Resolution       int
	LogTable         string
	WriteIndex       int
	Mode             string
	Perres           bool
	CassVersionTwo   bool
	CassVersionThree bool
	log              *logging.Logger
}

func NewCassandraMetricsSchema(
	conn *gocql.Session,
	keyscpace string,
	mtable string,
	resolutions [][]int,
	mode string,
	perres bool,
	version string,
) *cassandraMetricsSchema {
	cass := new(cassandraMetricsSchema)
	cass.conn = conn
	cass.Keyspace = keyscpace
	cass.MetricsTable = mtable
	cass.Resolutions = resolutions
	cass.Mode = mode
	cass.Perres = perres
	cass.CassVersionTwo = true
	cass.CassVersionThree = false

	if len(version) > 0 && strings.Split(version, ".")[0] == "3" {
		cass.CassVersionTwo = false
		cass.CassVersionThree = true
	}
	cass.log = logging.MustGetLogger("writers.cassandara.metric.schema")
	return cass
}

func (cass *cassandraMetricsSchema) AddMetricsTable() error {
	var err error

	if len(cass.Resolutions) == 0 {
		return fmt.Errorf("Need resolutions")
	}

	baseTpl := CASSANDRA_METRICS_BLOB_TEMPLATE
	if cass.Perres {
		baseTpl = CASSANDRA_METRICS_BLOB_PER_RES_TEMPLATE
	}

	if cass.Mode == "flat" {
		baseTpl = CASSANDRA_METRICS_FLAT_TEMPLATE
		if cass.Perres {
			baseTpl = CASSANDRA_METRICS_FLAT_PER_RES_TEMPLATE
		}
	}

	if cass.Mode == "flatmap" {
		baseTpl = CASSANDRA_METRICS_FLATMAP_TEMPLATE
		if cass.Perres {
			baseTpl = CASSANDRA_METRICS_FLATMAP_PER_RES_TEMPLATE
		}
	}
	if cass.Mode == "flatset" {
		baseTpl = CASSANDRA_METRICS_FLATSET_TEMPLATE
		if cass.Perres {
			baseTpl = CASSANDRA_METRICS_FLATSET_PER_RES_TEMPLATE
		}
	}

	if cass.Perres {
		for _, r := range cass.Resolutions {
			cass.Resolution = r[0]

			buf := bytes.NewBuffer(nil)
			cass.log.Notice("Cassandra Schema Driver: verifing schema")
			tpl := template.Must(template.New("cassmetric").Parse(baseTpl))
			err = tpl.Execute(buf, cass)
			if err != nil {
				cass.log.Errorf("%s", err)
				err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
				return err
			}
			Q := string(buf.Bytes())

			for _, q := range strings.Split(Q, "==SPLIT==") {

				err = cass.conn.Query(q).Exec()
				if err != nil {
					cass.log.Errorf("%s", err)
					err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
					return err
				}
				cass.log.Notice("Added table for resolution %s_%ds", cass.MetricsTable, r[0])
			}

		}

	} else {

		buf := bytes.NewBuffer(nil)
		cass.log.Notice("Cassandra Schema Driver: verifing schema")
		tpl := template.Must(template.New("cassmetric").Parse(baseTpl))
		err = tpl.Execute(buf, cass)

		if err != nil {
			cass.log.Errorf("%s", err)
			err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
			return err
		}
		cass.log.Notice("Added table for all resolutions %s", cass.MetricsTable)

		Q := string(buf.Bytes())
		for _, q := range strings.Split(Q, "==SPLIT==") {

			err = cass.conn.Query(q).Exec()
			if err != nil {
				cass.log.Errorf("%s", err)
				err = fmt.Errorf("Cassandra Schema Driver: Metric failed, %v", err)
				return err
			}
		}
	}
	return err
}

func (cass *cassandraMetricsSchema) AddMetricsLogTable() error {
	var err error

	buf := bytes.NewBuffer(nil)
	cass.log.Notice("Cassandra Schema Driver: verifing log schema")
	tpl := template.Must(template.New("cassmetriclog").Parse(CASSANDRA_METRICS_LOG_TEMPLATE))
	err = tpl.Execute(buf, cass)

	if err != nil {
		cass.log.Errorf("%s", err)
		err = fmt.Errorf("Cassandra Schema Driver: Metric Log failed, %v", err)
		return err
	}
	cass.log.Notice("Adding log table %s_%d_%ds", cass.LogTable, cass.WriteIndex, cass.Resolution)

	Q := string(buf.Bytes())
	for _, q := range strings.Split(Q, "==SPLIT==") {

		err = cass.conn.Query(q).Exec()
		if err != nil {
			cass.log.Errorf("%s", err)
			err = fmt.Errorf("Cassandra Schema Driver: Metric Log failed, %v", err)
			return err
		}
	}

	return err
}
