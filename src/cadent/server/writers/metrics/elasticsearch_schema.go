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
	Elastic search types

	Prefixes are `_{resolution}s` (i.e. "_" + (uint32 resolution) + "s")


*/

package metrics

import (
	"cadent/server/utils"
	"cadent/server/writers/indexer"
	"fmt"
	"golang.org/x/net/context"
	es5 "gopkg.in/olivere/elastic.v5"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"time"
)

const ELASTIC_METRICS_TEMPLATE = `
{
    "template":   "%s*",
    "mappings":
    {
        "_default_":
        {
            "_all": {
                "enabled": false
            },
            "dynamic_templates":
            [
		{
                    "int_fields": {
                        "match_mapping_type": "long",
                        "mapping": {
                            "type": "integer",
                            "index": false
                        }
                    }
                },
                {
                    "number_fields":
                    {
                        "match_mapping_type": "double",
                        "mapping": {
                            "type": "double",
                            "index": false
                        }
                    }
                },
                {
                    "string_fields":
                    {
                        "match_mapping_type": "string",
                        "mapping": {
                            "index": "not_analyzed",
                            "norms": false,
                            "omit_norms": true,
                            "type": "string",
                            "ignore_above": 128
                        }
                    }
                }
            ]
        }
    }
}`

const ELASTIC_METRICS_FLAT_TABLE = `
{
   "dynamic_templates": [{
        	"notanalyze": {
         		"mapping": {
            			"index": "not_analyzed",
            			"omit_norms": true
         		},
          		"match_mapping_type": "*",
          		"match": "*"
       		}
   }],
   "_all": {
	"enabled": false
   },
   "properties":{
        "uid":{
            "type": "keyword",
            "index": "not_analyzed"
        },
        "path":{
            "type": "keyword",
            "index": "not_analyzed",
            "norms": false
        },
        "time":{
            "type": "date",
            "index": "not_analyzed",
            "format": "strict_date_optional_time||epoch_millis"
        },
        "min":{
            "type": "double",
            "index": "not_analyzed"
        },
        "max":{
            "type": "double",
            "index": "not_analyzed"
        },
        "sum":{
            "type": "double",
            "index": "not_analyzed"
        },
        "last":{
            "type": "double",
            "index": "not_analyzed"
        },
        "count":{
            "type": "long",
            "index": "not_analyzed"
        },
        "tags":{
            "type": "nested",
            "properties":{
                "name": {
                    "type": "keyword",
                    "index": "not_analyzed"
                },
                "value": {
                    "type": "keyword",
                    "index": "not_analyzed"
                },
                "is_meta":{
                    "type": "boolean",
                    "index": "not_analyzed"
                }
            }
        }
   }
}`

const ELASTIC_METRICS_FLATMAP_TABLE = `
{
   "dynamic_templates": [{
	"notanalyze": {
		"mapping": {
			"index": "not_analyzed",
			"omit_norms": true
		},
		"match_mapping_type": "*",
		"match": "*"
	}},
	{"unindexed_longs": {
		"match_mapping_type": "long",
		"mapping": {
			"type": "long",
			"index": false
		}
	}},
	{"unindexed_doubles": {
		"match_mapping_type": "double",
		"mapping": {
			"type": "float",
			"index": false
		}
	}
   }],
   "_all": {
	"enabled": false
   },
   "properties":{
        "uid":{
            "type": "keyword",
            "index": "not_analyzed",
            "norms": false
        },
        "path":{
            "type": "keyword",
            "index": "not_analyzed",
            "norms": false
        },
        "slab":{
            "type": "keyword",
            "index": "not_analyzed",
            "norms": false
        },
        "time":{
            "type": "date",
            "index": "not_analyzed",
            "format": "strict_date_optional_time||epoch_millis"
        },
        "tags":{
            "type": "nested",
            "properties":{
                "name": {
                    "type": "keyword",
                    "index": "not_analyzed",
                    "norms": false

                },
                "value": {
                    "type": "keyword",
                    "index": "not_analyzed",
                    "norms": false
                },
                "is_meta":{
                    "type": "boolean",
                    "index": "not_analyzed"
                }
            }
        }
   }
}`

const ELASTIC_METRICS_BLOB_TABLE = `
{
   "dynamic_templates": [{
        	"notanalyze": {
         		"mapping": {
            			"index": "not_analyzed",
            			"omit_norms": true
         		},
          		"match_mapping_type": "*",
          		"match": "*"
       		}
   }],
   "_all": {
	"enabled": false
   },
   "_source": {
	"enabled": false
   },

   "properties":{
        "uid":{
            "type": "string",
            "index": "not_analyzed"
        },
        "path":{
            "type": "string",
            "index": "not_analyzed"
        },
        "ptype":{
        	"type": "string",
            	"index": "not_analyzed"
        },
        "stime":{
            "type": "long",
            "index": "not_analyzed",
            "format": "strict_date_optional_time||epoch_millis"
        },
        "etime":{
            "type": "long",
            "index": "not_analyzed",
            "format": "strict_date_optional_time||epoch_millis"
        },
        "points":{
            "type": "binary",
            "index": "not_analyzed"
        },
        "tags":{
            "type": "nested",
            "properties":{
                "name": {
                    "type": "string",
                    "index": "not_analyzed"
                },
                "value": {
                    "type": "string",
                    "index": "not_analyzed"
                },
                "is_meta":{
                    "type":"boolean",
                    "index": "not_analyzed"
                }
            }
        }
    }
}`

// ESMetric struct for the full (count > 1) metric
type ESMetric struct {
	Uid   string          `json:"uid"`
	Path  string          `json:"path"`
	Time  time.Time       `json:"time"`
	Min   float64         `json:"min"`
	Max   float64         `json:"max"`
	Sum   float64         `json:"sum"`
	Last  float64         `json:"last"`
	Count int64           `json:"count"`
	Tags  []indexer.ESTag `json:"tags,omitempty"`
}

// ESSmallMetric struct for the small (count==1) metric
type ESSmallMetric struct {
	Uid  string          `json:"uid"`
	Path string          `json:"path"`
	Time time.Time       `json:"time"`
	Sum  float64         `json:"sum"`
	Tags []indexer.ESTag `json:"tags,omitempty"`
}

// ESBlobMetric struct for the bytes series
type ESBlobMetric struct {
	Uid    string          `json:"uid"`
	Path   string          `json:"path"`
	Stime  time.Time       `json:"stime"`
	Etime  time.Time       `json:"etime"`
	Ptype  string          `json:"ptype"`
	Points []byte          `json:"points"`
	Tags   []indexer.ESTag `json:"tags,omitempty"`
}

// need to be able to sort the points list in ES by time
type ESPoints [][]float64

func (p ESPoints) Len() int { return len(p) }
func (p ESPoints) Less(i, j int) bool {
	return p[i][0] < p[j][0]
}
func (p ESPoints) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// ESMetricPoints struct for the points slab flat-map in ES
type ESMetricPoints struct {
	Uid    string          `json:"uid"`
	Path   string          `json:"path"`
	Slab   string          `json:"slab"`
	Time   time.Time       `json:"time"`
	Points ESPoints        `json:"points"`
	Tags   []indexer.ESTag `json:"tags,omitempty"`
}

// ElasticMetricsSchema schema/type maker for elastic search
type ElasticMetricsSchema struct {
	conn        *es5.Client
	tableBase   string
	resolutions [][]int
	mode        string
	log         *logging.Logger
	startstop   utils.StartStop
}

func NewElasticMetricsSchema(conn *es5.Client, metricTable string, resolutions [][]int, mode string) *ElasticMetricsSchema {
	es := new(ElasticMetricsSchema)
	es.conn = conn
	es.tableBase = metricTable
	es.resolutions = resolutions
	es.mode = mode
	es.log = logging.MustGetLogger("writers.elastic.metric.schema")
	return es
}

func (es *ElasticMetricsSchema) AddMetricsTable() (err error) {
	es.startstop.Start(func() {

		if len(es.resolutions) == 0 {
			err = fmt.Errorf("Need resolutions")
			return
		}

		// do the main template
		tplQ := fmt.Sprintf(ELASTIC_METRICS_TEMPLATE, es.tableBase)
		es.log.Notice("Adding default metrics index template... %s", tplQ)

		gTpl, terr := es.conn.IndexPutTemplate("metrics-template").BodyString(tplQ).Do(context.Background())
		if terr != nil {
			err = terr
			return
		}
		if gTpl == nil {
			err = fmt.Errorf("ElasticSearch Schema Driver: Metric index failed, %v", gTpl)
			es.log.Errorf("%v", err)
			return
		}

		// resolutions are [resolution][ttl]
		es.log.Notice("Adding default elasticsearch schemas for resolutions %v ...", es.resolutions)
		for _, res := range es.resolutions {
			tname := fmt.Sprintf("%s_%ds", es.tableBase, res[0])
			es.log.Notice("Adding elastic metric indexes `%s`", tname)

			// Use the IndexExists service to check if a specified index exists.
			exists, terr := es.conn.IndexExists(tname).Do(context.Background())
			if terr != nil {
				err = terr
				return
			}
			if !exists {
				_, err = es.conn.CreateIndex(tname).Do(context.Background())
				// skip the "already made" errors
				if err != nil {
					if !strings.Contains(err.Error(), "index_already_exists_exception") {
						es.log.Errorf("ElasticSearch Schema Driver: Metric index failed, %v", err)
						return
					}
					err = nil
				}
			}

			var Q string
			switch es.mode {
			case "flat":
				Q = ELASTIC_METRICS_FLAT_TABLE
			case "flatmap":
				Q = ELASTIC_METRICS_FLATMAP_TABLE
			default:
				Q = ELASTIC_METRICS_BLOB_TABLE
			}

			// see if it's there already
			got, terr := es.conn.GetMapping().Index(tname).Type("metric").Do(context.Background())
			if terr != nil {
				err = terr
				return
			}
			es.log.Notice("ElasticSearch Schema Driver: have metric mapping %v", err)
			var putresp *es5.PutMappingResponse
			if len(got) == 0 || err != nil {
				putresp, err = es.conn.PutMapping().Index(tname).Type("metric").BodyString(Q).Do(context.Background())
				if err != nil {
					es.log.Errorf("ElasticSearch Schema Driver: Metric index failed, %v", err)
					return
				}
				if putresp == nil {
					es.log.Errorf("ElasticSearch Schema Driver: Metric index failed, no response")
					err = fmt.Errorf("ElasticSearch Schema Driver: Metric index failed, no response")
					return
				}
				if !putresp.Acknowledged {
					err = fmt.Errorf("expected put mapping ack; got: %v", putresp.Acknowledged)
					return
				}
			}
		}
	})
	return err
}
