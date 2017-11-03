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
	The Redis DB object

*/

package dbs

import (
	"cadent/server/utils/options"
	"fmt"
	"gopkg.in/op/go-logging.v1"
	"gopkg.in/redis.v5"
	"strings"
)

/****************** Interfaces *********************/
type RedisDB struct {
	SingleClient  *redis.Client
	ClusterClient *redis.ClusterClient

	SingleOpts  *redis.Options
	ClusterOpts *redis.ClusterOptions

	hosts    string
	database int
	mode     string // cluster, single

	password      string
	metricPrefix  string
	idsPrefix     string
	pathPrefix    string
	segmentPrefix string
	tagPrefix     string

	poolSize int

	log *logging.Logger
}

func NewRedisDB() *RedisDB {
	r := new(RedisDB)
	r.log = logging.MustGetLogger("writers.redis")
	r.database = 0
	r.mode = "single"
	r.idsPrefix = "ids"
	r.pathPrefix = "path"
	r.tagPrefix = "tag"
	r.segmentPrefix = "segment"
	r.metricPrefix = "metric"
	r.password = ""
	r.poolSize = 8
	return r
}

func (r *RedisDB) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` host:port,host:port is needed for redis config (multiple of a cluster)")
	}
	r.hosts = dsn
	r.database = int(conf.Int64("database", 0))
	r.mode = conf.String("mode", "single")
	r.password = conf.String("password", "")
	r.metricPrefix = conf.String("metric_index", "metrics")
	r.pathPrefix = conf.String("path_index", "metric_path")
	r.segmentPrefix = conf.String("segment_index", "metric_segment")
	r.tagPrefix = conf.String("tag_index", "metric_tag")
	r.idsPrefix = conf.String("ids_index", "ids")
	r.poolSize = int(conf.Int64("poll_size", 8))

	hosts := strings.Split(dsn, ",")
	switch r.mode {
	case "cluster":
		r.ClusterOpts = &redis.ClusterOptions{
			Addrs:    hosts,
			Password: r.password,
			PoolSize: r.poolSize,
		}
		r.ClusterClient = redis.NewClusterClient(r.ClusterOpts)
		_, err = r.ClusterClient.Ping().Result()
	default:
		r.SingleOpts = &redis.Options{
			Addr:       r.hosts,
			Password:   r.password,
			DB:         r.database,
			PoolSize:   r.poolSize,
			MaxRetries: 3,
		}
		r.SingleClient = redis.NewClient(r.SingleOpts)
		_, err = r.SingleClient.Ping().Result()
	}

	return err
}

func (r *RedisDB) Tablename() string {
	return r.metricPrefix
}

func (r *RedisDB) PathTable() string {
	return r.pathPrefix
}

func (r *RedisDB) SegmentTable() string {
	return r.segmentPrefix
}

func (r *RedisDB) TagTable() string {
	return r.tagPrefix
}

func (r *RedisDB) IdsTable() string {
	return r.idsPrefix
}

func (r *RedisDB) Mode() string {
	return r.mode
}

func (r *RedisDB) Connection() DBConn {
	switch r.mode {
	case "cluster":
		return r.ClusterClient
	default:
		return r.SingleClient
	}
}

func (r *RedisDB) NewSingle() *redis.Client {
	return redis.NewClient(r.SingleOpts)
}

func (r *RedisDB) NewCluster() *redis.ClusterClient {
	return redis.NewClusterClient(r.ClusterOpts)
}
