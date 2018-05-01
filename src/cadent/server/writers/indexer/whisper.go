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
	THe Whisper "indexer"

	Does not "index" as the metrics writer will do that, but does do the globing matcher

*/

package indexer

import (
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"fmt"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

/****************** Interfaces *********************/
type WhisperIndexer struct {
	RamIndexer
	basePath  string
	indexerId string
	log       *logging.Logger
}

func NewWhisperIndexer() *WhisperIndexer {
	my := new(WhisperIndexer)
	my.RamIndexer = *NewRamIndexer()
	my.log = logging.MustGetLogger("indexer.whisper")
	return my
}

func (ws *WhisperIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` /root/path/of/data is needed for whisper config")
	}

	// config the Ram Cacher
	err = ws.RamIndexer.ConfigRam(conf)
	if err != nil {
		ws.log.Critical(err.Error())
		return err
	}

	ws.indexerId = conf.String("name", "indexer:whisper:"+ws.basePath)
	// reg ourselves before try to get conns
	ws.log.Noticef("Registering indexer: %s", ws.Name())
	err = RegisterIndexer(ws.Name(), ws)
	if err != nil {
		return err
	}

	ws.basePath = dsn

	// remove trialing "/"
	if strings.HasSuffix(dsn, "/") {
		ws.basePath = dsn[0 : len(dsn)-1]
	}

	return nil
}

func (ws *WhisperIndexer) Start() {
	ws.RamIndexer.Start()
}

func (ws *WhisperIndexer) Stop() {
	ws.RamIndexer.Stop()
}

func (ws *WhisperIndexer) Name() string { return ws.indexerId }

func (ws *WhisperIndexer) Delete(name *repr.StatName) error {
	return ws.RamIndexer.Delete(name)
}

func (ws *WhisperIndexer) Write(skey repr.StatName) error {
	go func() {
		if ws.RamIndexer.NeedsWrite(skey, CASSANDRA_HIT_LAST_UPDATE) {
			ws.RamIndexer.Write(skey) //add to ram index
		}
	}()
	return nil
}

// change {xxx,yyy} -> * as that's all the go lang glob can handle
// and so we turn it into a regex post
func (ws *WhisperIndexer) toGlob(metric string) (string, string, []string) {

	baseReg, outStrs := toGlob(metric)

	// need to include our base paths as we really are globing the file system
	globStr := filepath.Join(ws.basePath, strings.Replace(baseReg, ".", "/", -1))
	//regStr := filepath.Join(ws.basePath, strings.Replace(baseReg, ".", "/", -1))
	regStr := strings.Replace(baseReg, "/", "\\/", -1) + "(.*|.wsp)"

	return globStr, regStr, outStrs
}

/**** READER ***/
func (ws *WhisperIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return ws.RamIndexer.Find(ctx, metric, tags)
}

func (ws *WhisperIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	sp, closer := ws.GetSpan("Find", ctx)
	sp.LogKV("driver", "WhisperIndexer", "metric", metric, "tags", tags)
	defer closer()

	stats.StatsdClientSlow.Incr("indexer.whisper.finds", 1)

	// golangs globber does not handle "{a,b}" things, so we need to basically do a "*" then
	// perform a regex of the form (a|b) on the result ..
	globPath, regStr, _ := ws.toGlob(metric)

	var reger *regexp.Regexp
	var err error
	var mt indexer.MetricFindItems
	reger, err = regexp.Compile(regStr)
	if err != nil {
		return mt, err
	}

	paths, err := filepath.Glob(globPath)

	if err != nil {
		return mt, err
	}

	// try a direct match if nothing found
	if len(paths) == 0 {
		paths, err = filepath.Glob(globPath + ".wsp")
	}

	if err != nil {
		return mt, err
	}

	// a little special case for "exact" data metric matches
	// i.e. servers.all-1-stats-infra-integ.iostat.xvdg.writes
	// will match servers.all-1-stats-infra-integ.iostat.xvdg.writes and
	// servers.all-1-stats-infra-integ.iostat.xvdg.writes_bytes ...
	// so if we get an exact data hit .. just return that one
	for _, p := range paths {
		ms := new(indexer.MetricFindItem)

		// convert to the "." scheme again
		t := strings.Replace(p, ws.basePath+"/", "", 1)

		isData := filepath.Ext(p) == ".wsp"
		t = strings.Replace(t, ".wsp", "", -1)
		//ws.log.Critical("REG: %s, %s, %s", globPath, regStr, p)

		if !reger.MatchString(p) {
			continue
		}
		t = strings.Replace(t, "/", ".", -1)

		spl := strings.Split(t, ".")

		ms.Text = spl[len(spl)-1]

		ms.Id = t   // ID is whatever we want
		ms.Path = t // "path" is what our interface expects to be the "moo.goo.blaa" thing

		stat_name := repr.StatName{Key: p}
		if isData {
			uid := stat_name.UniqueIdString()
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			ms.UniqueId = uid
		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
		}

		// exact match special case
		if isData && t == metric {
			mt_ext := make(indexer.MetricFindItems, 1)
			mt_ext[0] = ms
			return mt_ext, nil
		}

		mt = append(mt, ms)
	}
	return mt, nil
}

func (ws *WhisperIndexer) Expand(metric string) (indexer.MetricExpandItem, error) {
	// a basic "Dir" operation
	globPath := filepath.Join(ws.basePath, strings.Replace(metric, ".", "/", -1))

	// golang's globber does not handle "{a,b}" things, so we need to basically do a "*" then
	// perform a regex of the form (a|b) on the result
	// TODO

	var mt indexer.MetricExpandItem
	paths, err := filepath.Glob(globPath)
	if err != nil {
		return mt, err
	}

	for _, p := range paths {
		// convert to the "." scheme again
		t := strings.Replace(p, ws.basePath+"/", "", 1)
		t = strings.Replace(t, "/", ".", -1)

		isData := filepath.Ext(p) == "wsp"

		if isData {
			t = strings.Replace(t, ".wsp", "", -1)
			mt.Results = append(mt.Results, t)
		}

	}
	return mt, nil
}

func (ws *WhisperIndexer) List(has_data bool, page int) (indexer.MetricFindItems, error) {

	// walk the file tree
	fi := indexer.MetricFindItems{}

	walker := func(path string, f os.FileInfo, err error) error {
		if err != nil {
			ws.log.Error("Error in directory walk: %v", err)
			return nil
		}

		if strings.HasSuffix(path, ".wsp") {
			onPth := strings.Replace(path, ws.basePath+"/", "", 1)
			onPth = strings.Replace(onPth, ".wsp", "", -1)
			onPth = strings.Replace(onPth, "/", ".", -1)

			ms := new(indexer.MetricFindItem)

			spl := strings.Split(onPth, ".")

			ms.Text = spl[len(spl)-1]
			ms.Id = onPth
			ms.Path = onPth

			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			//ms.UniqueId = id

			fi = append(fi, ms)
		}
		return nil
	}

	err := filepath.Walk(ws.basePath+"/", walker)

	return fi, err
}

func (ws *WhisperIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	return tags, metatags, ErrWillNotBeimplemented
}

func (ws *WhisperIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrWillNotBeimplemented
}

func (ws *WhisperIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrWillNotBeimplemented
}

func (ws *WhisperIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return uids, ErrWillNotBeimplemented
}
