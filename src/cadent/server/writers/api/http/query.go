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
   functions to help parse the input params from ye old http internface
*/

package http

import (
	sapi "cadent/server/schemas/api"
	"cadent/server/schemas/repr"
	"cadent/server/writers/metrics"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
)

var ErrorTargetRequired = errors.New("Target is required")
var ErrorInvalidStep = errors.New("Invalid `step` size")
var ErrorInvalidMaxPts = errors.New("Invalid `max_points` size")
var ErrorInvalidStartTime = errors.New("Invalid `start` time")
var ErrorInvalidEndTime = errors.New("Invalid `end` time")
var ErrorInvalidPage = errors.New("Invalid `page`")
var ErrorBadTagQuery = errors.New("Invalid Tag query `{name=val, name=val}`")

func getCType(h string) string {
	switch h {
	case "application/x-msgpack", "application/msgpack", "msgpack", "mp":
		return "msgpack"
	case "application/x-protobuf", "application/protobuf", "application/vnd.google.protobuf", "protobuf", "pb":
		return "protobuf"
	case "application/yaml", "application/x-yaml", "text/yaml", "yaml", "yml":
		return "yaml"
	default:
		return "json"
	}
}

// FormatFromHeaders get the format (msgpack, protobuf, json, yaml) from the http headers
func FormatFromHeaders(r *http.Request) string {
	// obey input content type first
	gHeader := strings.TrimSpace(r.Header.Get("Content-Type"))
	if len(gHeader) > 0 {
		return getCType(gHeader)
	}

	// pick the first one
	for _, h := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		return getCType(strings.TrimSpace(h))
	}
	return "json"
}

func GetOutFormat(r *http.Request) string {
	// try the AcceptEncoding header
	f := strings.TrimSpace(r.Form.Get("format"))
	if len(f) == 0 {
		return FormatFromHeaders(r)
	}

	return getCType(f)
}

// ParseNameToTags parse a tag query of the form key{name=val, name=val...}
func ParseNameToTags(query string) (key string, tags repr.SortingTags, err error) {
	// find the bits inside the {}

	inner := ""
	collecting := false
	key_collecting := true
	for _, char := range query {
		switch char {
		case '{':
			collecting = true
			key_collecting = false
		case '}':
			collecting = false
		default:
			if collecting {
				inner += string(char)
			}
			if key_collecting {
				key += string(char)
			}
		}
	}

	if len(inner) == 0 || collecting {
		return key, tags, ErrorBadTagQuery
	}
	t_arr := strings.Split(inner, ",")
	oTags := &repr.SortingTags{}
	for _, tg := range t_arr {
		t_split := strings.Split(strings.TrimSpace(tg), "=")
		if len(t_split) == 2 {
			oTags.Set(t_split[0], strings.Replace(t_split[1], "\"", "", -1))
		}
	}
	// if there are no tags, then we really have a {moo,goo} like regex
	if len(tags) == 0 {
		return query, *oTags, nil
	}

	return key, *oTags, nil

}

func ParseMetricQuery(r *http.Request) (mq sapi.MetricQuery, err error) {
	r.ParseForm()
	var target string
	var from string
	var to string
	var agg uint32
	var _tags string
	tags := &repr.SortingTags{}

	l := len(r.Form["target"])
	for idx, tar := range r.Form["target"] {
		target += strings.TrimSpace(tar)
		switch {
		case idx < l-1:
			target += ","
		}
	}

	// if no target try "path"
	if len(target) == 0 {
		l = len(r.Form["path"])
		for idx, tar := range r.Form["path"] {
			target += strings.TrimSpace(tar)
			switch {
			case idx < l-1:
				target += ","
			}
		}
	}

	// if no target try "query"
	if len(target) == 0 {
		l = len(r.Form["query"])
		for idx, tar := range r.Form["query"] {
			target += strings.TrimSpace(tar)
			switch {
			case idx < l-1:
				target += ","
			}
		}
	}

	l = len(r.Form["tags"])
	for idx, tgs := range r.Form["tags"] {
		_tags += strings.TrimSpace(tgs)
		switch {
		case idx < l-1:
			_tags += ","
		}
	}
	if _tags != "" {
		tags = repr.SortingTagsFromString(_tags)
	}

	// find a agg if desired
	_agg := r.Form.Get("agg")
	if len(_agg) > 0 {
		agg = repr.AggTypeFromTag(_agg)
	}

	// if true, will only return metrics that are in the ram caches
	inC := false
	_inC := r.Form.Get("in_cache")
	if len(_inC) > 0 {
		inC = _inC == "true" || _inC == "True" || _inC == "0"
	}

	if len(target) > 0 {
		// see if the name has a key{tag,tag}
		aKey, aTags, err := ParseNameToTags(target)
		if err == nil && len(aTags) > 0 {
			target = aKey
			tags.Merge(&aTags)
		}
	}

	from = strings.TrimSpace(r.Form.Get("from"))
	to = strings.TrimSpace(r.Form.Get("to"))

	if len(target) == 0 && tags.IsEmpty() {
		return mq, ErrorTargetRequired
	}

	if len(from) == 0 {
		// try "start"
		from = strings.TrimSpace(r.Form.Get("start"))
	}
	if len(from) == 0 {
		from = "-1h"
	}
	if len(to) == 0 {
		// try "end"
		to = strings.TrimSpace(r.Form.Get("end"))
	}
	if len(to) == 0 {
		to = "now"
	}

	start, err := metrics.ParseTime(from)
	if err != nil {
		return mq, ErrorInvalidStartTime
	}

	end, err := metrics.ParseTime(to)
	if err != nil {
		return mq, ErrorInvalidEndTime
	}
	if end < start {
		start, end = end, start
	}

	// grab a step if desired (resolution resampling)
	_step := strings.TrimSpace(r.Form.Get("step"))

	if len(_step) == 0 {
		_step = strings.TrimSpace(r.Form.Get("sample"))
	}

	step := uint32(0)
	if len(_step) > 0 {
		tstep, err := (strconv.ParseUint(_step, 10, 32))
		if err != nil {
			return mq, ErrorInvalidStep
		}
		step = uint32(tstep)
	}

	// grab a maxPoints if desired (resolution resampling)
	_maxpts := strings.TrimSpace(r.Form.Get("maxDataPoints"))
	if len(_maxpts) == 0 {
		_maxpts = strings.TrimSpace(r.Form.Get("max_points"))
	}
	if len(_maxpts) == 0 {
		_maxpts = strings.TrimSpace(r.Form.Get("maxpts"))
	}

	maxpts := uint64(0)
	if len(_maxpts) > 0 {
		maxpts, err = (strconv.ParseUint(_maxpts, 10, 32))
		if err != nil {
			return mq, ErrorInvalidMaxPts
		}
		// if maxPoints, need to resample to fit things if data
		t_step := uint32(end-start) / uint32(maxpts)
		if t_step > step {
			step = t_step
		}
	}

	// based on the min res provided, pick that as the "default" step

	// finally limit the number of points that can be returned
	if step > 0 {
		on_pts := uint32(end-start) / step
		if on_pts > MAX_METRIC_POINTS {
			step = uint32(end-start) / MAX_METRIC_POINTS
		}
	}

	return sapi.MetricQuery{
		Target:    target,
		Start:     start,
		End:       end,
		Step:      step,
		MaxPoints: uint32(maxpts),
		Tags:      *tags,
		Agg:       agg,
		Format:    GetOutFormat(r),
		InCache:   inC,
	}, nil
}

func ParseFindQuery(r *http.Request) (mq sapi.IndexQuery, err error) {
	r.ParseForm()

	vars := mux.Vars(r)

	query := strings.TrimSpace(r.Form.Get("name"))
	val := strings.TrimSpace(r.Form.Get("value"))
	inpage := strings.TrimSpace(r.Form.Get("page"))
	hasData := strings.TrimSpace(r.Form.Get("hasdata"))

	if len(query) == 0 {
		l := len(r.Form["query"])
		for idx, tar := range r.Form["query"] {
			query += strings.TrimSpace(tar)
			switch {
			case idx < l-1:
				query += ","
			}
		}
	}
	// try target from the params
	if len(query) == 0 {
		l := len(r.Form["target"])
		for idx, tar := range r.Form["target"] {
			query += strings.TrimSpace(tar)
			switch {
			case idx < l-1:
				query += ","
			}
		}
	}
	// try name from the URL
	if len(query) == 0 {
		query = vars["name"]
	}
	// try target from the url
	if len(query) == 0 {
		query = vars["target"]
	}

	if len(val) == 0 {
		val = vars["value"]
	}

	var tTags string
	tags := &repr.SortingTags{}

	l := len(r.Form["tags"])
	for idx, tgs := range r.Form["tags"] {
		tTags += strings.TrimSpace(tgs)
		switch {
		case idx < l-1:
			tTags += ","
		}
	}
	if tTags != "" {
		tags = repr.SortingTagsFromString(tTags)
	}

	if len(query) > 0 {
		// see if the name has a key{tag,tag}
		aKey, aTags, err := ParseNameToTags(query)
		if err == nil && len(aTags) > 0 {
			query = aKey
			tags.Merge(&aTags)
		}
	}

	// if true, will only return metrics that are in the ram caches
	inC := false
	_inC := r.Form.Get("in_cache")
	if len(_inC) > 0 {
		inC = _inC == "true" || _inC == "True" || _inC == "0"
	}
	mq.Tags = *tags
	mq.Query = query
	mq.Value = val
	mq.HasData = true
	mq.Format = GetOutFormat(r)
	mq.InCache = inC

	if len(hasData) > 0 && (hasData == "0" || hasData == "false") {
		mq.HasData = false
	}

	if len(inpage) > 0 {
		pg, err := (strconv.ParseUint(inpage, 10, 32))
		if err != nil {
			return mq, ErrorInvalidPage
		}
		mq.Page = uint32(pg)
	}

	return

}

func ParseDiscoverQuery(r *http.Request) (mq sapi.DiscoveryQuery, err error) {
	r.ParseForm()

	vars := mux.Vars(r)

	query := strings.TrimSpace(r.Form.Get("host"))

	if len(query) == 0 {
		query = strings.TrimSpace(r.Form.Get("query"))
	}
	if len(query) == 0 {
		query = strings.TrimSpace(r.Form.Get("name"))
	}
	// try target from the params
	if len(query) == 0 {
		query = strings.TrimSpace(r.Form.Get("target"))
	}
	// try name from the URL
	if len(query) == 0 {
		query = vars["name"]
	}
	// try target from the url
	if len(query) == 0 {
		query = vars["target"]
	}
	// try target from the url
	if len(query) == 0 {
		query = vars["host"]
	}

	var tTags string
	tags := &repr.SortingTags{}

	l := len(r.Form["tags"])
	for idx, tgs := range r.Form["tags"] {
		tTags += strings.TrimSpace(tgs)
		switch {
		case idx < l-1:
			tTags += ","
		}
	}
	if tTags != "" {
		tags = repr.SortingTagsFromString(tTags)
	}

	if len(query) > 0 {
		// see if the name has a key{tag,tag}
		aKey, aTags, err := ParseNameToTags(query)
		if err == nil && len(aTags) > 0 {
			query = aKey
			tags.Merge(&aTags)
		}
	}

	mq.Tags = *tags
	mq.Host = query
	mq.Format = GetOutFormat(r)

	return

}
