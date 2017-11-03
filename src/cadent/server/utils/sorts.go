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

package utils

// simple little "lists" that need sort defined for them

/** simple list of int64 for sorting (the sort.Ints is `int` only) */
// for sorting below
type Int64List []int64

func (p Int64List) Len() int { return len(p) }
func (p Int64List) Less(i, j int) bool {
	return p[i] < p[j]
}
func (p Int64List) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
