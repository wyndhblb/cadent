// Code generated by protoc-gen-gogo.
// source: indexer.proto
// DO NOT EDIT!

/*
Package indexer is a generated protocol buffer package.

It is generated from these files:
	indexer.proto

It has these top-level messages:
	MetricFindItem
	MetricFindItemList
	MetricExpandItem
	MetricTagItem
	MetricTagItemList
*/
package indexer

import testing "testing"
import math_rand "math/rand"
import github_com_gogo_protobuf_proto "github.com/gogo/protobuf/proto"
import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "cadent/server/schemas/repr"
import _ "github.com/gogo/protobuf/gogoproto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

func BenchmarkMetricFindItemProtoMarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricFindItem, 10000)
	for i := 0; i < 10000; i++ {
		pops[i] = NewPopulatedMetricFindItem(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(pops[i%10000])
		if err != nil {
			panic(err)
		}
		total += len(data)
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricFindItemProtoUnmarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	datas := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(NewPopulatedMetricFindItem(popr, false))
		if err != nil {
			panic(err)
		}
		datas[i] = data
	}
	msg := &MetricFindItem{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += len(datas[i%10000])
		if err := github_com_gogo_protobuf_proto.Unmarshal(datas[i%10000], msg); err != nil {
			panic(err)
		}
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricFindItemListProtoMarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricFindItemList, 10000)
	for i := 0; i < 10000; i++ {
		pops[i] = NewPopulatedMetricFindItemList(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(pops[i%10000])
		if err != nil {
			panic(err)
		}
		total += len(data)
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricFindItemListProtoUnmarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	datas := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(NewPopulatedMetricFindItemList(popr, false))
		if err != nil {
			panic(err)
		}
		datas[i] = data
	}
	msg := &MetricFindItemList{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += len(datas[i%10000])
		if err := github_com_gogo_protobuf_proto.Unmarshal(datas[i%10000], msg); err != nil {
			panic(err)
		}
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricExpandItemProtoMarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricExpandItem, 10000)
	for i := 0; i < 10000; i++ {
		pops[i] = NewPopulatedMetricExpandItem(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(pops[i%10000])
		if err != nil {
			panic(err)
		}
		total += len(data)
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricExpandItemProtoUnmarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	datas := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(NewPopulatedMetricExpandItem(popr, false))
		if err != nil {
			panic(err)
		}
		datas[i] = data
	}
	msg := &MetricExpandItem{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += len(datas[i%10000])
		if err := github_com_gogo_protobuf_proto.Unmarshal(datas[i%10000], msg); err != nil {
			panic(err)
		}
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricTagItemProtoMarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricTagItem, 10000)
	for i := 0; i < 10000; i++ {
		pops[i] = NewPopulatedMetricTagItem(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(pops[i%10000])
		if err != nil {
			panic(err)
		}
		total += len(data)
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricTagItemProtoUnmarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	datas := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(NewPopulatedMetricTagItem(popr, false))
		if err != nil {
			panic(err)
		}
		datas[i] = data
	}
	msg := &MetricTagItem{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += len(datas[i%10000])
		if err := github_com_gogo_protobuf_proto.Unmarshal(datas[i%10000], msg); err != nil {
			panic(err)
		}
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricTagItemListProtoMarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricTagItemList, 10000)
	for i := 0; i < 10000; i++ {
		pops[i] = NewPopulatedMetricTagItemList(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(pops[i%10000])
		if err != nil {
			panic(err)
		}
		total += len(data)
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricTagItemListProtoUnmarshal(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	datas := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		data, err := github_com_gogo_protobuf_proto.Marshal(NewPopulatedMetricTagItemList(popr, false))
		if err != nil {
			panic(err)
		}
		datas[i] = data
	}
	msg := &MetricTagItemList{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += len(datas[i%10000])
		if err := github_com_gogo_protobuf_proto.Unmarshal(datas[i%10000], msg); err != nil {
			panic(err)
		}
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricFindItemSize(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricFindItem, 1000)
	for i := 0; i < 1000; i++ {
		pops[i] = NewPopulatedMetricFindItem(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += pops[i%1000].Size()
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricFindItemListSize(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricFindItemList, 1000)
	for i := 0; i < 1000; i++ {
		pops[i] = NewPopulatedMetricFindItemList(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += pops[i%1000].Size()
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricExpandItemSize(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricExpandItem, 1000)
	for i := 0; i < 1000; i++ {
		pops[i] = NewPopulatedMetricExpandItem(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += pops[i%1000].Size()
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricTagItemSize(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricTagItem, 1000)
	for i := 0; i < 1000; i++ {
		pops[i] = NewPopulatedMetricTagItem(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += pops[i%1000].Size()
	}
	b.SetBytes(int64(total / b.N))
}

func BenchmarkMetricTagItemListSize(b *testing.B) {
	popr := math_rand.New(math_rand.NewSource(616))
	total := 0
	pops := make([]*MetricTagItemList, 1000)
	for i := 0; i < 1000; i++ {
		pops[i] = NewPopulatedMetricTagItemList(popr, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		total += pops[i%1000].Size()
	}
	b.SetBytes(int64(total / b.N))
}

//These tests are generated by github.com/gogo/protobuf/plugin/testgen
