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
   Workers for Kafka injector

   There are "4" basic workers each does something different

   1. RawMetric Worker -- takes a raw metric/value and injects it into an Accumulator loop
   2. UnProcessMetric Worker -- a non-process/accumulated repr metric (min/max/sum/last/count) and injects into Accumulator
   3. ProcessMetric Worker -- takes a "processed" (or accumulated) metric and injects it into a Writer
   4. SeriesMetric Worker -- takes a "processed series" (binary blob of many metrics) and injects it into a writer
   5. AnyMetric Worker -- do work on the messages based on the 4 main message types

*/

package injectors

import (
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	"cadent/server/utils/options"
	"cadent/server/writers"
	"fmt"
)

type Worker interface {
	Config(options.Options) error
	SetWriter(wr *writers.WriterLoop)
	GetWriter() *writers.WriterLoop
	SetSubWriter(wr *writers.WriterLoop)
	GetSubWriter() *writers.WriterLoop
	Repr(metric series.KMessageBase) *repr.StatRepr
	DoWork(metric series.KMessageBase) (*repr.StatRepr, error)
	DoWorkRepr(r *repr.StatRepr) error
	DoWorkReprOffset(r *repr.StatRepr, offset *metrics.OffsetInSeries) error
}

// WorkerBase base worker objects
type WorkerBase struct {
	writer    *writers.WriterLoop
	subWriter *writers.WriterLoop
}

func (wb *WorkerBase) Config(opts options.Options) error {
	return nil
}
func (wb *WorkerBase) SetWriter(wr *writers.WriterLoop) {
	wb.writer = wr
}
func (wb *WorkerBase) GetWriter() *writers.WriterLoop {
	return wb.writer
}
func (wb *WorkerBase) SetSubWriter(wr *writers.WriterLoop) {
	wb.subWriter = wr
}
func (wb *WorkerBase) GetSubWriter() *writers.WriterLoop {
	return wb.subWriter
}

func (wb *WorkerBase) DoWorkRepr(r *repr.StatRepr) (err error) {
	if wb.writer != nil {
		err = wb.writer.Metrics().Write(r)
	}
	if err != nil {
		return err
	}
	if wb.subWriter != nil {
		err = wb.subWriter.Metrics().Write(r)
	}
	return err
}

func (wb *WorkerBase) DoWorkReprOffset(r *repr.StatRepr, offset *metrics.OffsetInSeries) (err error) {
	if wb.writer != nil {
		err = wb.writer.Metrics().WriteWithOffset(r, offset)
	}
	if err != nil {
		return err
	}
	if wb.subWriter != nil {
		err = wb.subWriter.Metrics().WriteWithOffset(r, offset)
	}
	return err
}

/****************** RawMetric worker *********************/
type EchoWorker struct {
	WorkerBase
	Name string
}

func (w *EchoWorker) DoWork(metric series.KMessageBase) (*repr.StatRepr, error) {

	switch metric.(type) {
	case *series.KRawMetric:
		r := metric.(*series.KRawMetric).Repr()
		fmt.Println(r)
		return r, nil
	case *series.KUnProcessedMetric:
		r := metric.(*series.KUnProcessedMetric).Repr()
		fmt.Println(r)
		return r, nil
	case *series.KSingleMetric:
		r := metric.(*series.KSingleMetric).Repr()
		fmt.Println(metric.(*series.KSingleMetric).Repr())
		return r, nil
	case *series.KSeriesMetric:
		fmt.Println("metric series: " + metric.(*series.KSeriesMetric).Metric)
		return nil, nil
	case *series.KWrittenMessage:
		fmt.Println("Written message:" + metric.(*series.KWrittenMessage).Id())
		return nil, nil
	default:
		return nil, ErrorBadMessageType
	}
}

// Repr get the metric from the object
func (w *EchoWorker) Repr(metric series.KMessageBase) *repr.StatRepr {
	switch metric.(type) {
	case *series.KRawMetric:
		return metric.(*series.KRawMetric).Repr()
	case *series.KUnProcessedMetric:
		return metric.(*series.KUnProcessedMetric).Repr()
	case *series.KSingleMetric:
		return metric.(*series.KSingleMetric).Repr()
	case *series.KSeriesMetric:
		return nil
	case *series.KWrittenMessage:
		return nil
	default:
		return nil
	}
}

/****************** RawMetric worker *********************/
type RawMetricWorker struct {
	WorkerBase
	Name string
}

func (w *RawMetricWorker) DoWork(metric series.KMessageBase) (*repr.StatRepr, error) {
	if w.writer == nil {
		return nil, ErrorWriterNotDefined
	}
	r := w.Repr(metric)
	if r == nil {
		return nil, ErrorBadMessageType
	}
	w.DoWorkRepr(r)
	return r, nil
}

func (w *RawMetricWorker) Repr(metric series.KMessageBase) *repr.StatRepr {
	switch metric.(type) {
	case *series.KRawMetric:
		return metric.(*series.KRawMetric).Repr()
	default:
		return nil
	}
}

/****************** UnProcessed worker *********************/
type UnProcessedMetricWorker struct {
	WorkerBase
	Name string
}

func (w *UnProcessedMetricWorker) Repr(metric series.KMessageBase) *repr.StatRepr {
	switch metric.(type) {
	case *series.KUnProcessedMetric:
		return metric.(*series.KUnProcessedMetric).Repr()
	default:
		return nil
	}
}

func (w *UnProcessedMetricWorker) DoWork(metric series.KMessageBase) (*repr.StatRepr, error) {
	if w.writer == nil {
		return nil, ErrorWriterNotDefined
	}
	r := w.Repr(metric)
	if r == nil {
		return nil, ErrorBadMessageType
	}
	w.DoWorkRepr(r)
	return r, nil
}

/****************** Single worker *********************/
type SingleMetricWorker struct {
	WorkerBase
	Name string
}

func (w *SingleMetricWorker) Repr(metric series.KMessageBase) *repr.StatRepr {
	switch metric.(type) {
	case *series.KSingleMetric:
		return metric.(*series.KSingleMetric).Repr()
	default:
		return nil
	}
}

func (w *SingleMetricWorker) DoWork(metric series.KMessageBase) (*repr.StatRepr, error) {
	if w.writer == nil {
		return nil, ErrorWriterNotDefined
	}
	r := w.Repr(metric)
	if r == nil {
		return nil, ErrorBadMessageType
	}
	w.DoWorkRepr(r)
	return r, nil
}

// AnyMetricWorker
type AnyMetricWorker struct {
	WorkerBase
	Name string
}

func (w *AnyMetricWorker) Repr(metric series.KMessageBase) *repr.StatRepr {
	switch metric.(type) {
	case *series.KMetric:
		return metric.(*series.KMetric).Repr()
	case *series.KRawMetric:
		return metric.(*series.KRawMetric).Repr()
	case *series.KUnProcessedMetric:
		return metric.(*series.KUnProcessedMetric).Repr()
	case *series.KSingleMetric:
		return metric.(*series.KSingleMetric).Repr()
	default:
		return nil
	}
}
func (w *AnyMetricWorker) DoWork(metric series.KMessageBase) (*repr.StatRepr, error) {
	if w.writer == nil {
		return nil, ErrorWriterNotDefined
	}
	r := w.Repr(metric)
	if r == nil {
		return nil, ErrorBadMessageType
	}
	w.DoWorkRepr(r)
	return r, nil
}

// AnyMetricListWorker
type AnyMetricListWorker struct {
	WorkerBase
	Name string
}

func (w *AnyMetricListWorker) DoWork(metric series.KMessageListBase) error {
	if w.writer == nil {
		return ErrorWriterNotDefined
	}
	switch metric.(type) {
	case *series.KMetricList:
		for _, r := range metric.(*series.KMetricList).Reprs() {
			if r != nil {
				w.DoWorkRepr(r)
			}
		}
		return nil
	case *series.KRawMetricList:
		for _, r := range metric.(*series.KRawMetricList).Reprs() {
			if r != nil {
				w.DoWorkRepr(r)
			}
		}
		return nil
	case *series.KUnProcessedMetricList:
		for _, r := range metric.(*series.KUnProcessedMetricList).Reprs() {
			if r != nil {
				w.DoWorkRepr(r)
			}
		}
		return nil
	case *series.KSingleMetricList:
		for _, r := range metric.(*series.KSingleMetricList).Reprs() {
			if r != nil {
				w.DoWorkRepr(r)
			}
		}
		return nil

	default:
		return ErrorBadMessageType
	}
}

// RawMetricListWorker
type RawMetricListWorker struct {
	WorkerBase
	Name string
}

func (w *RawMetricListWorker) DoWork(metric series.KRawMetricList) error {
	if w.writer == nil {
		return ErrorWriterNotDefined
	}
	for _, r := range metric.Reprs() {
		if r != nil {
			w.DoWorkRepr(r)
		}
	}
	return nil
}

// UnprocessedMetricListWorker
type UnprocessedMetricListWorker struct {
	WorkerBase
	Name string
}

func (w *UnprocessedMetricListWorker) DoWork(metric series.KUnProcessedMetricList) error {
	if w.writer == nil {
		return ErrorWriterNotDefined
	}
	for _, r := range metric.Reprs() {
		if r != nil {
			w.DoWorkRepr(r)
		}
	}
	return nil
}

// SingleMetricListWorker
type SingleMetricListWorker struct {
	WorkerBase
	Name string
}

func (w *SingleMetricListWorker) DoWork(metric series.KSingleMetricList) error {
	if w.writer == nil {
		return ErrorWriterNotDefined
	}
	for _, r := range metric.Reprs() {
		if r != nil {
			w.DoWorkRepr(r)
		}
	}
	return nil
}
