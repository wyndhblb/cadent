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
	THe MySQL write

	The table should have this schema to match the repr item

// this is for easy index searches on paths


CREATE TABLE `{segmentTable}` (
  `segment` varchar(255) NOT NULL DEFAULT '',
  `pos` int NOT NULL,
  PRIMARY KEY (`pos`, `segment`)
);

CREATE TABLE `{pathTable}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
  `segment` varchar(255) NOT NULL,
  `pos` int NOT NULL,
  `uid` varchar(50) NOT NULL,
  `path` varchar(255) NOT NULL DEFAULT '',
  `length` int NOT NULL,
  `has_data` bool DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `seg_pos` (`segment`, `pos`),
  KEY `uid` (`uid`),
  KEY `length` (`length`)
);


// for "mysql-flat
CREATE TABLE `{metrics-table}{resolutionprefix}` (
      `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `sum` float NOT NULL,
      `min` float NOT NULL,
      `max` float NOT NULL,
      `first` float NOT NULL,
      `last` float NOT NULL,
      `count` float NOT NULL,
      `resolution` int(11) NOT NULL,
      `time` datetime(6) NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`time`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

// for "mysql"  blob form
CREATE TABLE `{table}{prefix}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `point_type` varchar(20) NOT NULL DEFAULT '',
      `points` blob,
      `stime` BIGINT unsigned NOT NULL,
      `etime` BIGINT unsigned NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`stime`, `etime`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	OPTIONS: For `Config`

		table: base table name (default: metrics)
		pathTable: base table name (default: metric_path)
		prefix: table prefix if any (_1s, _5m)
		batch_count: batch this many inserts for much faster insert performance (default 1000)
		periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package dbs

import (
	"cadent/server/utils/options"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"time"
)

/****************** Interfaces *********************/
type MySQLDB struct {
	conn         *sql.DB
	table        string
	idTable      string
	pathTable    string
	segmentTable string
	tablePrefix  string
	tagTable     string
	tagTableXref string

	log *logging.Logger
}

func NewMySQLDB() *MySQLDB {
	my := new(MySQLDB)
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func (my *MySQLDB) Config(conf *options.Options) error {

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}

	//parse time yes
	if !strings.Contains(dsn, "parseTime=true") {
		dsn = dsn + "?parseTime=true"
	}

	my.conn, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	my.table = conf.String("table", "metrics")
	my.pathTable = conf.String("path_table", "metric_path")
	my.segmentTable = conf.String("segment_table", "metric_segment")
	my.tagTable = conf.String("tag_table", "metric_tag")
	my.idTable = conf.String("id_table", "metric_ids")
	my.tagTableXref = conf.String("tagTable_xref", my.tagTable+"_xref")
	my.tablePrefix = conf.String("prefix", "")

	// some reasonable defaults
	my.conn.SetMaxOpenConns(50)
	my.conn.SetConnMaxLifetime(time.Duration(5 * time.Minute))

	return nil
}

func (my *MySQLDB) RootMetricsTableName() string {
	return my.table
}

func (my *MySQLDB) Tablename() string {
	return my.table + my.tablePrefix
}

func (my *MySQLDB) PathTable() string {
	return my.pathTable
}

func (my *MySQLDB) IdTable() string {
	return my.idTable
}

func (my *MySQLDB) TagTable() string {
	return my.tagTable
}

func (my *MySQLDB) TagTableXref() string {
	return my.tagTableXref
}

func (my *MySQLDB) SegmentTable() string {
	return my.segmentTable
}

func (my *MySQLDB) Connection() DBConn {
	return my.conn
}
