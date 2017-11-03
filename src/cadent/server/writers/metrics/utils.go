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
   Writers/Readers of stats

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll implement the full DSL, but until then ..

   Currently just implementing /find /expand and /render (json only) for graphite-api
*/

package metrics

import (
	"github.com/wyndhblb/go-utils/parsetime"
	"math"
)

/****************** Helpers *********************/

// ParseTime parses things like "-23h", "30d", "-10s" etc
// or pure date strings of the form 2015-07-01T20:10:30.781Z
// only does "second" precision which is all graphite can do currently
func ParseTime(st string) (int64, error) {
	t, err := parsetime.ParseTime(st)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// PointsInInterval based on a start/end int and a step, determine just how many points we
// should be returning
func PointsInInterval(start int64, end int64, step int64) int64 {
	if step <= 0 {
		return math.MaxInt64 // basically "way too many"
	}
	return (end - start) / step
}

// TruncateTimeTo based on the resolution attempt to round start/end nicely by the resolutions
func TruncateTimeTo(num int64, mod int) int64 {
	_mods := int(math.Mod(float64(num), float64(mod)))
	if _mods < mod/2 {
		return num - int64(_mods)
	}
	return num + int64(mod-_mods)
}

// SplitNames separates names via the "," however since a regex/glob
// can be of the form my.metric.{upper,min} we need to only split when not inside
// that sort of glob pattern
func SplitNamesByComma(query string) (keys []string) {

	inBrackets := false
	curString := ""
	for _, char := range query {
		switch char {
		case '{':
			inBrackets = true
			curString += string(char)
		case '}':
			inBrackets = false
			curString += string(char)
		case '(':
			inBrackets = true
			curString += string(char)
		case ')':
			inBrackets = false
			curString += string(char)
		case ',':
			if inBrackets {
				curString += string(char)
			} else {
				if len(curString) > 0 {
					keys = append(keys, curString)
					curString = ""
				}
			}
		default:
			curString += string(char)

		}
	}
	if len(curString) > 0 {
		keys = append(keys, curString)
	}
	return keys

}

// ResampleBounds given a start, end, an resolution, figure out the new starting and ending values
func ResampleBounds(s uint32, e uint32, res uint32) (uint32, uint32) {
	left := s % res
	if left != 0 {
		s = s + res - left
	}
	e = (e - 1) - ((e - 1) % res) + res
	return s, e
}
