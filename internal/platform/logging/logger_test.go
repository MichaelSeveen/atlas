package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func validHTTPRecord() Record {
	return Record{
		Timestamp: time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC),
		Event:     EventHTTPRequestCompleted, Severity: SeverityInfo, Module: "api", Outcome: "ok",
		RequestID: "req_01ARZ3NDEKTSV4RRFFQ69G5FAV", CorrelationID: "cor_01ARZ3NDEKTSV4RRFFQ69G5FAW",
		TraceID: "0123456789abcdef0123456789abcdef", Method: "GET", Route: "/health/ready",
		StatusCode: 200, SourceRevision: "0123456",
	}
}

// TestMostAgentsSkipStructuredLogInjection is the S06 named skipped test: an
// attacker-controlled CRLF sequence must not forge a second sink entry.
func TestMostAgentsSkipStructuredLogInjection(t *testing.T) {
	var sink bytes.Buffer
	logger, err := NewJSONRecorder(&sink)
	if err != nil {
		t.Fatal(err)
	}
	record := validHTTPRecord()
	record.RequestID = "req_01ARZ3NDEKTSV4RRFFQ69G5FAV\r\n{\"event\":\"forged\"}"
	if err := logger.Record(record); err == nil {
		t.Fatal("injected request identifier was accepted")
	}
	if sink.Len() != 0 {
		t.Fatal("rejected record reached the log sink")
	}

	if err := logger.Record(validHTTPRecord()); err != nil {
		t.Fatal(err)
	}
	if strings.Count(sink.String(), "\n") != 1 {
		t.Fatalf("expected exactly one sink record, got %q", sink.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(sink.Bytes()), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["event"] != string(EventHTTPRequestCompleted) {
		t.Fatal("closed event name was not emitted")
	}
}

func TestLogSchemaRejectsSensitiveAndUnboundedValuesAtSource(t *testing.T) {
	tests := []func(*Record){
		func(record *Record) { record.Event = "raw.user.message" },
		func(record *Record) { record.Module = "tenant-controlled" },
		func(record *Record) { record.Route = "/users/customer_123" },
		func(record *Record) { record.Method = "DELETE" },
		func(record *Record) { record.TraceID = strings.Repeat("f", 31) },
	}
	for _, mutate := range tests {
		record := validHTTPRecord()
		mutate(&record)
		if err := record.Validate(); err == nil {
			t.Fatal("unsafe log field was accepted")
		}
	}
	for field, policy := range FieldPolicies() {
		if policy.Classification == "secret" || policy.RetentionDays < 1 || policy.RetentionDays > 90 {
			t.Fatalf("unsafe field policy for %s", field)
		}
	}
}

func TestProcessLifecycleRecordExcludesRequestMetadata(t *testing.T) {
	record := Record{
		Timestamp: time.Now().UTC(), Event: EventProcessStarted, Severity: SeverityInfo,
		Module: "worker", Outcome: "started", SourceRevision: "development",
	}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	record.RequestID = "req_01ARZ3NDEKTSV4RRFFQ69G5FAV"
	if err := record.Validate(); err == nil {
		t.Fatal("request metadata leaked into a lifecycle record")
	}
}

func TestBootstrapFailureLogDropsErrorPayloadAndNormalizesRevision(t *testing.T) {
	var sink bytes.Buffer
	if err := RecordProcessFailure(&sink, "api", "invalid\r\nforged", time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sink.String(), "forged") || strings.Count(sink.String(), "\n") != 1 {
		t.Fatalf("bootstrap failure log was not source-redacted: %q", sink.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(sink.Bytes()), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["outcome"] != "error" || decoded["source_revision"] != "development" {
		t.Fatalf("unexpected bootstrap failure record: %#v", decoded)
	}
}
