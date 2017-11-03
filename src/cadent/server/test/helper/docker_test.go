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

// some helpers for testing various components that are hard to "mock"
// like databases and what not, here we have a docker compose "fire-upper"

package helper

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDockerUppers(t *testing.T) {

	Convey("Docker up and down", t, func() {

		ok := DockerUp("kafka")
		So(ok, ShouldEqual, true)

		ok = DockerWaitUntilReady("kafka")
		So(ok, ShouldEqual, true)

		ok = DockerDown("kafka")
		So(ok, ShouldEqual, true)
	})
}
