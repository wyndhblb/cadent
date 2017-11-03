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
	Error definitions
*/

package schemas

import (
	"errors"
)

var ErrMetricIsNil = errors.New("The metric is nil")
var ErrMetricUnDecodable = errors.New("The metric is not able to be decoded")
var ErrNoMetricDefined = errors.New("No metric defined")
var ErrMergeStepSizeError = errors.New("To merge 2 RawRenderItems, the step size needs to be the same")
var ErrQuantizeTooLittleData = errors.New("Cannot quantize to step: too little data")
var ErrQuantizeStepTooSmall = errors.New("Cannot quantize: to a '0' step size ...")
var ErrQuantizeStepTooManyPoints = errors.New("Cannot quantize: Over the 10,000 point threashold, please try a smaller time window")
