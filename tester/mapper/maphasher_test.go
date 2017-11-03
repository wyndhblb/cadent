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

package mapper

import (
	mmh3 "github.com/reusee/mmh3"
	"hash/crc32"
	"math/rand"
	"testing"

	"hash/fnv"
)

var mp_str map[string]bool
var r_list_str []string

var mp_int map[int]bool
var r_list_int []int

var mp_crc map[uint32]bool
var mp_mmh map[uint32]bool
var mp_fvn map[uint32]bool

func init() {

	mp_str = make(map[string]bool)
	mp_int = make(map[int]bool)
	mp_crc = make(map[uint32]bool)
	mp_mmh = make(map[uint32]bool)
	mp_fvn = make(map[uint32]bool)

	r_list_str = make([]string, 100000)
	for idx := range r_list_str {
		r_list_str[idx] = RandStr()
	}

	r_list_int = make([]int, 100000)
	for idx := range r_list_int {
		r_list_int[idx] = rand.Intn(123123123123)
	}
}

func BenchmarkMapStringPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			mp_str[st] = true
		}
	}
}

func BenchmarkMapStringGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			_ = mp_str[st]
		}
	}
}

func BenchmarkMapIntPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_int {
			mp_int[st] = true
		}
	}
}

func BenchmarkMapIntGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_int {
			_ = mp_int[st]
		}
	}
}

func BenchmarkMapCRCPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			k := crc32.ChecksumIEEE([]byte(st))
			mp_crc[k] = true
		}
	}
}

func BenchmarkMapCRCGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			k := crc32.ChecksumIEEE([]byte(st))
			_ = mp_crc[k]
		}
	}
}

func BenchmarkMapMMHPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			hash := mmh3.New32()
			hash.Write([]byte(st))
			out := hash.Sum32()
			mp_mmh[out] = true
		}
	}
}

func BenchmarkMapMMHGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			hash := mmh3.New32()
			hash.Write([]byte(st))
			out := hash.Sum32()
			_ = mp_mmh[out]
		}
	}
}

func BenchmarkMapFVNPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			hash := fnv.New32a()
			hash.Write([]byte(st))
			out := hash.Sum32()
			mp_fvn[out] = true
		}
	}
}

func BenchmarkMapFVNGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			hash := fnv.New32a()
			hash.Write([]byte(st))
			out := hash.Sum32()
			_ = mp_fvn[out]
		}
	}
}
