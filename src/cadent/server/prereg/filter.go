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

package prereg

import (
	"bytes"
	"cadent/server/accumulator"
	"fmt"
	"regexp"
)

/********************** prefix filter ***********************/

type PrefixFilter struct {
	Prefix     string `json:"prefix"`
	prefixByte []byte
	IsReject   bool `json:"is_rejected"`
	backend    string
}

func (pref PrefixFilter) ToString() string {
	return fmt.Sprintf(
		"Type: `%-6s` Match:`%-30s` Rejected: `%v` Backend: `%s`",
		pref.Type(),
		pref.Name(),
		pref.Rejecting(),
		pref.Backend(),
	)
}

func (pref *PrefixFilter) Name() string {
	return pref.Prefix
}
func (pref *PrefixFilter) Type() string {
	return "prefix"
}
func (pref *PrefixFilter) Rejecting() bool {
	return pref.IsReject
}
func (pref *PrefixFilter) Init() error {
	pref.prefixByte = []byte(pref.Prefix)
	return nil
}

func (pref *PrefixFilter) Match(in []byte) (bool, bool, error) {
	match := bytes.HasPrefix(in, pref.prefixByte)
	return match, pref.IsReject, nil
}

func (pref *PrefixFilter) Backend() string {
	return pref.backend
}

func (pref *PrefixFilter) SetBackend(back string) (string, error) {
	pref.backend = back
	return back, nil
}

/**********************   SubString filter ***********************/
type SubStringFilter struct {
	SubString      string `json:"substring"`
	subStringBytes []byte
	IsReject       bool `json:"is_rejected"`
	backend        string
}

func (sfilter *SubStringFilter) Name() string {
	return sfilter.SubString
}
func (sfilter *SubStringFilter) Type() string {
	return "substring"
}
func (sfilter *SubStringFilter) Rejecting() bool {
	return sfilter.IsReject
}
func (sfilter *SubStringFilter) Init() error {
	sfilter.subStringBytes = []byte(sfilter.SubString)
	return nil
}
func (sfilter *SubStringFilter) Match(in []byte) (bool, bool, error) {
	match := bytes.Contains(in, sfilter.subStringBytes)
	return match, sfilter.IsReject, nil
}

func (sfilter *SubStringFilter) Backend() string {
	return sfilter.backend
}

func (sfilter *SubStringFilter) SetBackend(back string) (string, error) {
	sfilter.backend = back
	return back, nil
}
func (sfilter *SubStringFilter) ToString() string {
	return fmt.Sprintf(
		"Type: `%-6s` Match:`%-30s` Rejected: `%v` Backend: `%s`",
		sfilter.Type(),
		sfilter.Name(),
		sfilter.Rejecting(),
		sfilter.Backend(),
	)
}

/**********************   reg filter ***********************/
type RegexFilter struct {
	RegexString string `json:"regex"`
	IsReject    bool   `json:"is_rejected"`
	backend     string

	thereg *regexp.Regexp
}

func (refilter *RegexFilter) Name() string {
	return refilter.RegexString
}
func (refilter *RegexFilter) Type() string {
	return "regex"
}
func (refilter *RegexFilter) Rejecting() bool {
	return refilter.IsReject
}
func (refilter *RegexFilter) Init() error {
	refilter.thereg = regexp.MustCompile(refilter.RegexString)
	return nil
}
func (refilter *RegexFilter) Match(in []byte) (bool, bool, error) {
	match := refilter.thereg.Match(in)
	return match, refilter.IsReject, nil
}

func (refilter *RegexFilter) Backend() string {
	return refilter.backend
}

func (refilter *RegexFilter) SetBackend(back string) (string, error) {
	refilter.backend = back
	return back, nil
}
func (refilter *RegexFilter) ToString() string {
	return fmt.Sprintf(
		"Type: `%-6s` Match:`%-30s` Rejected: `%v` Backend: `%s`",
		refilter.Type(),
		refilter.Name(),
		refilter.Rejecting(),
		refilter.Backend(),
	)
}

/**********************   no-op filter ***********************/
type NoOpFilter struct {
	backend  string
	IsReject bool `json:"is_rejected"`
}

func (nop *NoOpFilter) Name() string {
	return "NoOp"
}
func (nop *NoOpFilter) Type() string {
	return "noop"
}
func (nop *NoOpFilter) Rejecting() bool {
	return false
}
func (nop *NoOpFilter) Init() error {
	return nil
}

//always match of true
func (nop *NoOpFilter) Match(in []byte) (bool, bool, error) {
	return true, false, nil
}

func (nop *NoOpFilter) Backend() string {
	return nop.backend
}

func (nop *NoOpFilter) SetBackend(back string) (string, error) {
	nop.backend = back
	return back, nil
}
func (nop *NoOpFilter) ToString() string {
	return fmt.Sprintf(
		"Type: `%-6s` Match:`%-30s` Rejected: `%v` Backend: `%s`",
		nop.Type(),
		nop.Name(),
		nop.Rejecting(),
		nop.Backend(),
	)
}

/**********************  filter list ***********************/

type PreReg struct {
	DefaultBackEnd string `json:"default_backend"`
	Name           string `json:"name"`

	// the actual "listening" server that this reg is pinned to
	ListenServer string `json:"listen_server_name"`

	FilterList  []FilterItem             `json:"map"`
	Accumulator *accumulator.Accumulator `json:"accumulator"`
}

func (pr *PreReg) MatchingFilters(line []byte) []FilterItem {
	// just the list of filters that match a string

	var fitems = make([]FilterItem, 0)
	for _, fil := range pr.FilterList {
		matched, _, _ := fil.Match(line)
		if matched {
			fitems = append(fitems, fil)
		}
	}
	return fitems
}

func (pr *PreReg) FirstMatchFilter(line []byte) (FilterItem, bool, error) {
	for _, fil := range pr.FilterList {
		matched, reject, err := fil.Match(line)
		if matched {
			return fil, reject, err
		}
	}
	return nil, false, nil
}

func (pr *PreReg) FirstMatchBackend(line []byte) (string, bool, error) {
	for _, fil := range pr.FilterList {
		matched, reject, err := fil.Match(line)
		if matched {
			return fil.Backend(), reject, err
		}
	}
	return pr.DefaultBackEnd, false, nil
}

func (pr *PreReg) Stop() {
	if pr.Accumulator != nil {
		pr.Accumulator.Stop()
	}
}

func (pr *PreReg) LogConfig() {
	log.Debug(" - Pre Filter Group: %s", pr.Name)
	log.Debug("   - Pinned To Listener: %s", pr.ListenServer)
	for idx, filter := range pr.FilterList {
		log.Debug("   - PreFilter %2d:: %s", idx, filter.ToString())
	}
	if pr.Accumulator != nil {
		log.Debug("   - PreFilter Accumulator")
		pr.Accumulator.LogConfig()
	}
}

/**********************  maps of filter list ***********************/

// list of filters
type PreRegMap map[string]*PreReg

func (lpr *PreRegMap) MatchingFilters(line []byte) []FilterItem {
	var fitems = make([]FilterItem, 0)
	for _, pr := range *lpr {
		gots := pr.MatchingFilters(line)
		if gots != nil {
			fitems = append(fitems, gots...)
		}
	}
	return fitems
}

func (lpr *PreRegMap) FirstMatchingFilters(line []byte) []FilterItem {
	var fitems = make([]FilterItem, 0)
	for _, pr := range *lpr {
		gots := pr.MatchingFilters(line)
		if gots != nil {
			fitems = append(fitems, gots...)
		}
	}
	return fitems
}
func (lpr *PreRegMap) FirstMatchBackends(line []byte) ([]string, []bool, []error) {
	var backs = make([]string, 0)
	var reject = make([]bool, 0)
	var errs = make([]error, 0)
	for _, pr := range *lpr {
		bck, rjc, err := pr.FirstMatchBackend(line)
		backs = append(backs, bck)
		reject = append(reject, rjc)
		errs = append(errs, err)
	}
	return backs, reject, errs
}

func (lpr *PreRegMap) LogConfig() {
	log.Debug("=== Pre Filters ===")
	for _, filter := range *lpr {
		filter.LogConfig()
	}
}
