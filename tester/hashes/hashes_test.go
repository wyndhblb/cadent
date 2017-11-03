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



// some hashing functions that return an int32 for our hasher
//
package hashes

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/dgryski/go-farm"
	"hash/crc32"
	"hash/fnv"
	"strconv"
	"testing"
	"github.com/reusee/mmh3"
	"github.com/dgryski/go-sip13"

)

func Crc32(str []byte) uint32 {
	return crc32.ChecksumIEEE(str)
}

func Fnv32(str []byte) uint32 {
	hash := fnv.New32()
	hash.Write(str)
	return hash.Sum32()
}

func Fnv32a(str []byte) uint32 {
	hash := fnv.New32a()
	hash.Write(str)
	return hash.Sum32()
}

func Fnv64(str []byte) uint64 {
	hash := fnv.New64()
	hash.Write(str)
	return hash.Sum64()
}

func Fnv64a(str []byte) uint64 {
	hash := fnv.New64a()
	hash.Write(str)
	return hash.Sum64()
}

func Farm32(str []byte) uint32 {
	return farm.Hash32(str)
}

func Farm64(str []byte) uint64 {
	return farm.Hash64(str)
}

func MMH(str []byte) uint32 {
	return mmh3.Sum32(str)
}


func SIP(str []byte) uint64 {
	var k0 uint64 = 0x0706050403020100
	var k1 uint64 = 0x0f0e0d0c0b0a0908
	return sip13.Sum64(k0, k1, str)
}

// this is hashing function used by graphite
// see https://github.com/graphite-project/carbon/blob/master/lib/carbon/hashing.py
// we cannot use a raw MD5 as it will need a bigint (beyond uint64 which is just silly for
// this application)
func SmallMd5(str []byte) uint32 {
	h := md5.New()
	h.Write(str)
	hexstr := hex.EncodeToString(h.Sum(nil))
	num, _ := strconv.ParseUint(hexstr[0:4], 16, 32)
	return uint32(num)
}

// we cannot use a raw SHA1 as it will need a bigint (beyond uint64 which is just silly for
// this application)
func SmallSHA1(str []byte) uint32 {
	h := sha1.New()
	h.Write(str)
	hexstr := hex.EncodeToString(h.Sum(nil))
	num, _ := strconv.ParseUint(hexstr[0:4], 16, 32)
	return uint32(num)
}

var res32 uint32
var res64 uint64

func Benchmark_Hashes(b *testing.B) {

	mp32 := map[string]interface{}{
		"crc32":  Crc32,
		"fnv32":  Fnv32,
		"fnv32a": Fnv32a,
		"farm32": Farm32,
		"mmh32": MMH,
	}
	mp64 := map[string]interface{}{
		"fnv64":  Fnv64,
		"fnv64a": Fnv64a,
		"farm64": Farm64,
		"sip": SIP,
	}

	str := []byte(`RMVx)@MLxH9M.WeGW-ktWwR3Cy1XS.,K~i@n-Y+!!yx4?AB%cM~l/#0=2:BOn7HPipG&o/6Qe<hU;$w1-~bU4Q7N&yk/8*Zz.Yg?zl9bVH/pXs6Bq^VdW#Z)NH!GcnH-UesRd@gDij?luVQ3;YHaQ<~SBm17G9;RWvGlsV7tpe*RCe=,?$nE1u9zvjd+rBMu7_Rg4)2AeWs^aaBr&FkC#rcwQ.L->I+Da7Qt~!C^cB2wq(^FGyB?kGQpd(G8I.A7")`)
	var r32 uint32
	var r64 uint64

	for hashname, f := range mp32 {
		b.Run(fmt.Sprintf("HASH_%s", hashname), func(b *testing.B) {
			useF := f.(func([]byte) uint32)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r32 = useF(str)
			}
			res32 = r32
		})

	}
	for hashname, f := range mp64 {
		b.Run(fmt.Sprintf("HASH_%s", hashname), func(b *testing.B) {
			useF := f.(func([]byte) uint64)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r64 = useF(str)
			}
			res64 = r64
		})

	}
}
