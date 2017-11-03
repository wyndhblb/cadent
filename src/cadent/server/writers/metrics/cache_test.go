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

package metrics

import (
	"cadent/server/schemas/repr"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacher(t *testing.T) {

	nm := repr.StatName{Key: "moo.goo.org", Resolution: 1, Ttl: 1000}
	rep := repr.StatRepr{
		Time:  time.Now().UnixNano(),
		Count: 1,
		Sum:   100,
		Name:  &nm,
	}

	Convey("CacherChunk should function basically", t, func() {
		tmp, _ := GetCacherSingleton("myniftyname", "chunk")
		c := tmp.(*CacherChunk)

		So(c, ShouldNotEqual, nil)

		c.SetName("monkey")
		So(c.Name, ShouldEqual, "monkey")

		c.SetSeriesEncoding("gob")

		err := c.Add(&nm, &rep)

		So(err, ShouldEqual, nil)

		So(c.SliceLen(), ShouldEqual, 1)
		So(c.ChunkLen(), ShouldEqual, 1)

	})

	Convey("CacherChunk should emit log messages", t, func() {
		tmp, _ := GetCacherSingleton("moogoo", "chunk")
		c := tmp.(*CacherChunk)

		c.SetMaxChunks(2)
		c.SetLogTimeWindow(2)
		c.Start()
		lchan := c.GetLogChan()

		for i := int64(0); i < 100; i++ {
			repCp := rep
			repCp.Time = time.Now().UnixNano() + i*int64(10*time.Second)
			err := c.Add(&nm, &repCp)
			So(err, ShouldEqual, nil)
		}

		gots := <-lchan.Ch
		g := gots.(*CacheChunkLog)
		So(g.SequenceId, ShouldEqual, 0)
		So(len(g.Slice), ShouldEqual, 1)
		So(len(g.Slice[nm.UniqueId()]), ShouldEqual, 100)
		So(c.SliceLen(), ShouldEqual, 0)
		lchan.Close()
		c.Stop()

	})

	Convey("CacherChunk should emit slice messages", t, func() {
		tmp, _ := GetCacherSingleton("anothername", "chunk")
		c := tmp.(*CacherChunk)

		c.SetMaxChunks(2)
		c.SetLogTimeWindow(2)
		c.SetChunkTimeWindow(10)
		c.Start()

		schan := c.GetSliceChan()

		doadd := func() {
			orName := rep.Name.Key
			for {
				for i := int64(0); i < 100; i++ {
					repCp := rep.Copy()
					repCp.SetKey(fmt.Sprintf("%s.%d", orName, i))
					repCp.Time = time.Now().UnixNano() + i*int64(10*time.Second)
					c.Add(repCp.Name, repCp)
					time.Sleep(100 * time.Microsecond)
				}
			}
		}
		go doadd()

		So(atomic.LoadInt64(&c.curSequenceId), ShouldEqual, 0)

		gots := <-schan.Ch
		g, ok := gots.(*CacheChunkSlice)
		if !ok {
			t.Fatalf("Not a CacheChunkLog object")
		}
		So(len(g.Slice.AllSeries()), ShouldEqual, 100)
		So(atomic.LoadInt64(&c.curSequenceId), ShouldEqual, 1)
		gots = <-schan.Ch
		g, ok = gots.(*CacheChunkSlice)
		if !ok {
			t.Fatalf("Not a CacheChunkLog object")
		}
		So(len(g.Slice.AllSeries()), ShouldEqual, 100)
		So(atomic.LoadInt64(&c.curSequenceId), ShouldEqual, 2)

	})

	Convey("CacherChunk should get Series", t, func() {
		tmp, _ := GetCacherSingleton("anothername2", "chunk")
		c := tmp.(*CacherChunk)

		c.SetMaxChunks(2)
		c.SetLogTimeWindow(2)
		c.SetChunkTimeWindow(10)
		c.Start()
		orName := rep.Name.Key

		doadd := func() {
			for {
				for i := int64(0); i < 100; i++ {
					repCp := rep.Copy()
					repCp.SetKey(fmt.Sprintf("%s.%d", orName, i))
					repCp.Time = time.Now().UnixNano() + i*int64(10*time.Second)
					c.Add(repCp.Name, repCp)
					time.Sleep(100 * time.Microsecond)
				}
			}
		}
		go doadd()
		time.Sleep(15 * time.Second)
		for i := 0; i < 100; i++ {
			repCp := rep.Copy()
			repCp.SetKey(fmt.Sprintf("%s.%d", orName, i))
			series, _ := c.GetAsRawRenderItem(repCp.Name)
			So(series, ShouldNotEqual, nil)
			time.Sleep(100 * time.Microsecond)
		}

	})

}
