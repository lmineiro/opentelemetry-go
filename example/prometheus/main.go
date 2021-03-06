// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/exporter/metric/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/batcher/defaultkeys"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

var (
	fooKey    = key.New("ex.com/foo")
	barKey    = key.New("ex.com/bar")
	lemonsKey = key.New("ex.com/lemons")
)

func initMeter() *push.Controller {
	selector := simple.NewWithExactMeasure()
	exporter, err := prometheus.NewExporter(prometheus.Options{})

	if err != nil {
		log.Panicf("failed to initialize metric stdout exporter %v", err)
	}
	// Prometheus needs to use a stateful batcher since counters (and histogram since they are a collection of Counters)
	// are cumulative (i.e., monotonically increasing values) and should not be resetted after each export.
	//
	// Prometheus uses this approach to be resilient to scrape failures.
	// If a Prometheus server tries to scrape metrics from a host and fails for some reason,
	// it could try again on the next scrape and no data would be lost, only resolution.
	//
	// Gauges (or LastValues) and Summaries are an exception to this and have different behaviors.
	batcher := defaultkeys.New(selector, sdkmetric.NewDefaultLabelEncoder(), true)
	pusher := push.New(batcher, exporter, time.Second)
	pusher.Start()

	go func() {
		_ = http.ListenAndServe(":2222", exporter)
	}()

	global.SetMeterProvider(pusher)
	return pusher
}

func main() {
	defer initMeter().Stop()

	meter := global.MeterProvider().Meter("ex.com/basic")

	oneMetric := meter.NewFloat64Gauge("ex.com.one",
		metric.WithKeys(fooKey, barKey, lemonsKey),
		metric.WithDescription("A gauge set to 1.0"),
	)

	measureTwo := meter.NewFloat64Measure("ex.com.two", metric.WithKeys(key.New("A")))
	measureThree := meter.NewFloat64Counter("ex.com.three")

	commonLabels := meter.Labels(lemonsKey.Int(10), key.String("A", "1"), key.String("B", "2"), key.String("C", "3"))
	notSoCommonLabels := meter.Labels(lemonsKey.Int(13))

	ctx := context.Background()

	meter.RecordBatch(
		ctx,
		commonLabels,
		oneMetric.Measurement(1.0),
		measureTwo.Measurement(2.0),
		measureThree.Measurement(12.0),
	)

	meter.RecordBatch(
		ctx,
		notSoCommonLabels,
		oneMetric.Measurement(1.0),
		measureTwo.Measurement(2.0),
		measureThree.Measurement(22.0),
	)

	time.Sleep(5 * time.Second)

	meter.RecordBatch(
		ctx,
		commonLabels,
		oneMetric.Measurement(13.0),
		measureTwo.Measurement(12.0),
		measureThree.Measurement(13.0),
	)

	time.Sleep(100 * time.Second)
}
