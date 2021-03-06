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

package stdout

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	export "go.opentelemetry.io/otel/sdk/export/trace"
)

func TestExporter_ExportSpan(t *testing.T) {
	ex, err := NewExporter(Options{})
	if err != nil {
		t.Errorf("Error constructing stdout exporter %s", err)
	}

	// override output writer for testing
	var b bytes.Buffer
	ex.outputWriter = &b

	// setup test span
	now := time.Now()
	traceID, _ := core.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	spanID, _ := core.SpanIDFromHex("0102030405060708")
	keyValue := "value"
	doubleValue := 123.456

	testSpan := &export.SpanData{
		SpanContext: core.SpanContext{
			TraceID: traceID,
			SpanID:  spanID,
		},
		Name:      "/foo",
		StartTime: now,
		EndTime:   now,
		Attributes: []core.KeyValue{
			key.String("key", keyValue),
			key.Float64("double", doubleValue),
		},
		MessageEvents: []export.Event{
			{Name: "foo", Attributes: []core.KeyValue{key.String("key", keyValue)}, Time: now},
			{Name: "bar", Attributes: []core.KeyValue{key.Float64("double", doubleValue)}, Time: now},
		},
		SpanKind: trace.SpanKindInternal,
		Status:   codes.Unknown,
	}
	ex.ExportSpan(context.Background(), testSpan)

	expectedSerializedNow, _ := json.Marshal(now)

	got := b.String()
	expectedOutput := `{"SpanContext":{` +
		`"TraceID":"0102030405060708090a0b0c0d0e0f10",` +
		`"SpanID":"0102030405060708","TraceFlags":0},` +
		`"ParentSpanID":"0000000000000000",` +
		`"SpanKind":1,` +
		`"Name":"/foo",` +
		`"StartTime":` + string(expectedSerializedNow) + "," +
		`"EndTime":` + string(expectedSerializedNow) + "," +
		`"Attributes":[` +
		`{` +
		`"Key":"key",` +
		`"Value":{"Type":"STRING","Value":"value"}` +
		`},` +
		`{` +
		`"Key":"double",` +
		`"Value":{"Type":"FLOAT64","Value":123.456}` +
		`}` +
		`],` +
		`"MessageEvents":[` +
		`{` +
		`"Name":"foo",` +
		`"Attributes":[` +
		`{` +
		`"Key":"key",` +
		`"Value":{"Type":"STRING","Value":"value"}` +
		`}` +
		`],` +
		`"Time":` + string(expectedSerializedNow) +
		`},` +
		`{` +
		`"Name":"bar",` +
		`"Attributes":[` +
		`{` +
		`"Key":"double",` +
		`"Value":{"Type":"FLOAT64","Value":123.456}` +
		`}` +
		`],` +
		`"Time":` + string(expectedSerializedNow) +
		`}` +
		`],` +
		`"Links":null,` +
		`"Status":2,` +
		`"HasRemoteParent":false,` +
		`"DroppedAttributeCount":0,` +
		`"DroppedMessageEventCount":0,` +
		`"DroppedLinkCount":0,` +
		`"ChildSpanCount":0}` + "\n"

	if got != expectedOutput {
		t.Errorf("Want: %v but got: %v", expectedOutput, got)
	}
}
