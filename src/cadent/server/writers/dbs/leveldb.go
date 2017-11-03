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

LevelDB key/value store for local saving for indexes on the stat key space
and how it maps to files on disk.

metrics are not all stored in on DB, instead we take the UID string

aqf8aa0006eh, take the first 2 chars a,q and put metrics into

a/q/metrics/{dbstuff}

meaning there potentially will be up to 1296 different DBs (any more partitions and we will quickly
hit some open file limit issues)

*/

package dbs

import (
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_filter "github.com/syndtr/goleveldb/leveldb/filter"
	leveldb_opt "github.com/syndtr/goleveldb/leveldb/opt"

	"cadent/server/utils/options"
	"fmt"
	leveldb_storage "github.com/syndtr/goleveldb/leveldb/storage"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"path/filepath"
	"sync"
)

const (
	DEFAULT_LEVELDB_READ_CACHE_SIZE  = 8
	DEFAULT_LEVELDB_FILE_SIZE        = 20
	DEFAULT_LEVELDB_SEGMENT_FILE     = "segments"
	DEFAULT_LEVELDB_METRICS_PATH     = "metrics"
	DEFAULT_LEVELDB_METRICS_LOG_FILE = "metriclogs"
	DEFAULT_LEVELDB_PATH_FILE        = "paths"
	DEFAULT_LEVELDB_UID_FILE         = "uids"
	DEFAULT_LEVELDB_TAG_FILE         = "tags"
	DEFAULT_LEVELDB_TAGXREF_FILE     = "tags_xref"
)

/****************** Interfaces *********************/
type LevelDB struct {
	segmentConn    *leveldb.DB
	pathConn       *leveldb.DB
	tagConn        *leveldb.DB
	tagxrefConn    *leveldb.DB
	uidConn        *leveldb.DB
	metricLogConn  *leveldb.DB
	tablePath      string
	segmentFile    string
	tagFile        string
	pathFile       string
	uidFile        string
	tagxrefFile    string
	metricPathRoot string
	metricsPath    string
	metricLogFile  string

	metricDBs map[string]*leveldb.DB
	mlock     sync.RWMutex

	levelOpts *leveldb_opt.Options

	log *logging.Logger
}

func NewLevelDB() *LevelDB {
	lb := new(LevelDB)
	lb.log = logging.MustGetLogger("writers.leveldb")
	lb.metricDBs = make(map[string]*leveldb.DB)
	return lb
}

func (lb *LevelDB) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (/path/to/db/folder) is needed for leveldb config")
	}

	lb.segmentFile = DEFAULT_LEVELDB_SEGMENT_FILE
	lb.tagFile = DEFAULT_LEVELDB_TAG_FILE
	lb.pathFile = DEFAULT_LEVELDB_PATH_FILE
	lb.uidFile = DEFAULT_LEVELDB_UID_FILE
	lb.tagxrefFile = DEFAULT_LEVELDB_TAGXREF_FILE
	lb.metricsPath = DEFAULT_LEVELDB_METRICS_PATH
	lb.metricLogFile = DEFAULT_LEVELDB_METRICS_LOG_FILE

	lb.tablePath = dsn
	lb.metricPathRoot = dsn
	lb.levelOpts = new(leveldb_opt.Options)
	lb.levelOpts.Filter = leveldb_filter.NewBloomFilter(10)

	lb.levelOpts.BlockCacheCapacity = int(conf.Int64("read_cache_size", DEFAULT_LEVELDB_READ_CACHE_SIZE)) * leveldb_opt.MiB
	lb.levelOpts.CompactionTableSize = int(conf.Int64("file_compact_size", DEFAULT_LEVELDB_FILE_SIZE)) * leveldb_opt.MiB
	lb.levelOpts.ReadOnly = conf.Bool("read_only", false)

	// for the "metrics" writer we do not want to open the index tables
	openIndexTables := conf.Bool("open_index_tables", true)
	if !openIndexTables {
		return nil
	}

	lb.tagConn, err = leveldb.OpenFile(lb.TagTableName(), lb.levelOpts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Tag DB is corrupt. Recovering.")
			lb.segmentConn, err = leveldb.RecoverFile(lb.TagTableName(), lb.levelOpts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lb.segmentConn, err = leveldb.OpenFile(lb.SegmentTableName(), lb.levelOpts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Segment DB is corrupt. Recovering.")
			lb.segmentConn, err = leveldb.RecoverFile(lb.SegmentTableName(), lb.levelOpts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lb.pathConn, err = leveldb.OpenFile(lb.PathTableName(), lb.levelOpts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Path DB is corrupt. Recovering.")
			lb.segmentConn, err = leveldb.RecoverFile(lb.PathTableName(), lb.levelOpts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	lb.uidConn, err = leveldb.OpenFile(lb.UidTableName(), lb.levelOpts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("UID DB is corrupt. Recovering.")
			lb.uidConn, err = leveldb.RecoverFile(lb.UidTableName(), lb.levelOpts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	lb.tagxrefConn, err = leveldb.OpenFile(lb.TagXrefTableName(), lb.levelOpts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("TagXref DB is corrupt. Recovering.")
			lb.uidConn, err = leveldb.RecoverFile(lb.TagXrefTableName(), lb.levelOpts)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (lb *LevelDB) TagTableName() string {
	return filepath.Join(lb.tablePath, lb.tagFile)
}

func (lb *LevelDB) TagXrefTableName() string {
	return filepath.Join(lb.tablePath, lb.tagxrefFile)
}

func (lb *LevelDB) SegmentTableName() string {
	return filepath.Join(lb.tablePath, lb.segmentFile)
}

func (lb *LevelDB) PathTableName() string {
	return filepath.Join(lb.tablePath, lb.pathFile)
}

func (lb *LevelDB) UidTableName() string {
	return filepath.Join(lb.tablePath, lb.uidFile)
}

func (lb *LevelDB) SegmentConn() *leveldb.DB {
	return lb.segmentConn
}

func (lb *LevelDB) PathConn() *leveldb.DB {
	return lb.pathConn
}

func (lb *LevelDB) UidConn() *leveldb.DB {
	return lb.uidConn
}

func (lb *LevelDB) TagConn() *leveldb.DB {
	return lb.tagConn
}

// this is just for the interface match ..
// real users need to cast this to a real LevelDB obj
func (lb *LevelDB) Connection() DBConn {
	return lb.pathFile
}

// MetricLogTableName file path for the metric log table
func (lb *LevelDB) MetricLogTableName() string {
	return filepath.Join(lb.tablePath, lb.metricLogFile)
}

// MetricLogDB get the metrics DB for the logs
func (lb *LevelDB) MetricLogDB() (*leveldb.DB, error) {
	if lb.metricLogConn != nil {
		return lb.metricLogConn, nil
	}

	var err error
	lb.metricLogConn, err = leveldb.OpenFile(lb.MetricLogTableName(), lb.levelOpts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Metric Log DB is corrupt. Recovering.")
			lb.uidConn, err = leveldb.RecoverFile(lb.MetricLogTableName(), lb.levelOpts)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return lb.metricLogConn, nil
}

// MetricDBTableName name of the path for the metrics table
func (lb *LevelDB) MetricDBTableName(uid string) string {
	c1 := uid[0]
	c2 := uid[1]
	return filepath.Join(lb.metricPathRoot, string(c1), string(c2), lb.metricsPath)
}

// MetricDB get the metrics DB for a given UID
func (lb *LevelDB) MetricDB(uid string) (*leveldb.DB, error) {
	dbName := lb.MetricDBTableName(uid)

	lb.mlock.RLock()
	if gots, ok := lb.metricDBs[dbName]; ok {
		lb.mlock.RUnlock()
		return gots, nil
	}
	lb.mlock.RUnlock()

	// need to make the dirs if not there
	info, err := os.Stat(dbName)

	if err != nil || !info.IsDir() {
		err = os.MkdirAll(dbName, 0755)
		if err != nil {
			return nil, err
		}
	}

	lb.mlock.Lock()
	defer lb.mlock.Unlock()

	dbconn, err := leveldb.OpenFile(dbName, lb.levelOpts)
	if err != nil {
		if _, ok := err.(*leveldb_storage.ErrCorrupted); ok {
			lb.log.Notice("Metrics DB '%s' is corrupt. Recovering.", dbName)
			dbconn, err = leveldb.RecoverFile(dbName, lb.levelOpts)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	lb.metricDBs[dbName] = dbconn
	return dbconn, nil
}

// OpenMetricTables get all the metric tables open (a copy of the internal map)
func (lb *LevelDB) OpenMetricTables() map[string]*leveldb.DB {
	lb.mlock.RLock()
	defer lb.mlock.RUnlock()

	dbs := make(map[string]*leveldb.DB)
	for k, v := range lb.metricDBs {
		dbs[k] = v
	}
	return dbs
}

func (lb *LevelDB) Close() (err error) {
	if lb.tagConn != nil {
		err = lb.tagConn.Close()
	}
	if lb.segmentConn != nil {
		err = lb.segmentConn.Close()
	}
	if lb.uidConn != nil {
		err = lb.uidConn.Close()
	}
	if lb.pathConn != nil {
		err = lb.pathConn.Close()
	}
	if lb.tagxrefConn != nil {
		err = lb.tagxrefConn.Close()
	}
	if lb.metricLogConn != nil {
		err = lb.metricLogConn.Close()
	}

	lb.mlock.Lock()
	for _, db := range lb.metricDBs {
		if db != nil {
			db.Close()
		}
	}
	lb.metricDBs = make(map[string]*leveldb.DB)
	lb.mlock.Unlock()

	return err
}
