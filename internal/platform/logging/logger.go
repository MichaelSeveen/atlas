// Package logging emits the closed, source-redacted Atlas foundation log schema.
package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sync"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type Event string

const (
	EventHTTPRequestCompleted Event = "http.request.completed"
	EventProcessStarted       Event = "process.started"
	EventProcessStopped       Event = "process.stopped"
)

type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityError Severity = "error"
)

// Record deliberately has no free-form message, error, body, header, actor,
// tenant, credential, or payload field. Sensitive values therefore cannot be
// passed to the foundation sink by accident.
type Record struct {
	Timestamp      time.Time
	Event          Event
	Severity       Severity
	Module         string
	Outcome        string
	RequestID      string
	CorrelationID  string
	TraceID        string
	Method         string
	Route          string
	StatusCode     int
	SourceRevision string
}

type Recorder interface {
	Record(Record) error
}

type Discard struct{}

func (Discard) Record(Record) error { return nil }

type JSONRecorder struct {
	writer io.Writer
	mu     sync.Mutex
}

func NewJSONRecorder(writer io.Writer) (*JSONRecorder, error) {
	if writer == nil {
		return nil, errors.New("log sink is required")
	}
	return &JSONRecorder{writer: writer}, nil
}

// RecordProcessFailure is the bootstrap-safe failure path for runtime
// entrypoints. It deliberately accepts no error or message value, so an error
// returned by a dependency can never be copied into the sink before the normal
// logger has been constructed.
func RecordProcessFailure(writer io.Writer, module, sourceRevision string, now time.Time) error {
	if !sourceRevisionPattern.MatchString(sourceRevision) {
		sourceRevision = "development"
	}
	logger, err := NewJSONRecorder(writer)
	if err != nil {
		return err
	}
	return logger.Record(Record{
		Timestamp:      now.UTC(),
		Event:          EventProcessStopped,
		Severity:       SeverityError,
		Module:         module,
		Outcome:        "error",
		SourceRevision: sourceRevision,
	})
}

var (
	traceIDPattern        = regexp.MustCompile(`^[0-9a-f]{32}$`)
	sourceRevisionPattern = regexp.MustCompile(`^(development|[0-9a-f]{7,64})$`)
)

var allowedEvents = map[Event]struct{}{
	EventHTTPRequestCompleted: {},
	EventProcessStarted:       {},
	EventProcessStopped:       {},
}

var allowedValues = map[string]map[string]struct{}{
	"severity": {string(SeverityInfo): {}, string(SeverityError): {}},
	"module":   {"api": {}, "worker": {}, "simulator": {}, "database": {}, "platform": {}},
	"outcome":  {"ok": {}, "error": {}, "rejected": {}, "ready": {}, "not_ready": {}, "started": {}, "stopped": {}, "degraded": {}},
	"method":   {"": {}, "GET": {}, "OPTIONS": {}, "OTHER": {}},
	"route":    {"": {}, "/health/live": {}, "/health/ready": {}, "/version": {}, "unmatched": {}},
}

func (record Record) Validate() error {
	if record.Timestamp.IsZero() {
		return errors.New("log timestamp is required")
	}
	if _, ok := allowedEvents[record.Event]; !ok {
		return errors.New("log event is outside the foundation schema")
	}
	for field, value := range map[string]string{
		"severity": string(record.Severity), "module": record.Module, "outcome": record.Outcome,
		"method": record.Method, "route": record.Route,
	} {
		if _, ok := allowedValues[field][value]; !ok {
			return errors.New("log field is outside its allowlist")
		}
	}
	if err := validateOpaqueID(record.RequestID, "req"); err != nil {
		return err
	}
	if err := validateOpaqueID(record.CorrelationID, "cor"); err != nil {
		return err
	}
	if record.TraceID != "" && !traceIDPattern.MatchString(record.TraceID) {
		return errors.New("log trace identifier is invalid")
	}
	if record.StatusCode != 0 && (record.StatusCode < 100 || record.StatusCode > 599) {
		return errors.New("log status code is invalid")
	}
	if !sourceRevisionPattern.MatchString(record.SourceRevision) {
		return errors.New("log source revision is invalid")
	}
	if record.Event == EventHTTPRequestCompleted {
		if record.RequestID == "" || record.CorrelationID == "" || record.Method == "" || record.Route == "" || record.StatusCode == 0 {
			return errors.New("HTTP completion log is incomplete")
		}
	} else if record.RequestID != "" || record.CorrelationID != "" || record.TraceID != "" || record.Method != "" || record.Route != "" || record.StatusCode != 0 {
		return errors.New("process lifecycle log contains request fields")
	}
	return nil
}

func validateOpaqueID(value, prefix string) error {
	if value == "" {
		return nil
	}
	parsed, err := identifier.Parse(value)
	if err != nil || parsed.Prefix() != prefix {
		return errors.New("log opaque identifier is invalid")
	}
	return nil
}

type wireRecord struct {
	Timestamp      string   `json:"timestamp"`
	Event          Event    `json:"event"`
	Severity       Severity `json:"severity"`
	Module         string   `json:"module"`
	Outcome        string   `json:"outcome"`
	RequestID      string   `json:"request_id,omitempty"`
	CorrelationID  string   `json:"correlation_id,omitempty"`
	TraceID        string   `json:"trace_id,omitempty"`
	Method         string   `json:"http_method,omitempty"`
	Route          string   `json:"http_route,omitempty"`
	StatusCode     int      `json:"http_status_code,omitempty"`
	SourceRevision string   `json:"source_revision"`
}

func (logger *JSONRecorder) Record(record Record) error {
	if logger == nil || logger.writer == nil {
		return errors.New("log sink is unavailable")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	wire := wireRecord{
		Timestamp: record.Timestamp.UTC().Format(time.RFC3339Nano), Event: record.Event,
		Severity: record.Severity, Module: record.Module, Outcome: record.Outcome,
		RequestID: record.RequestID, CorrelationID: record.CorrelationID, TraceID: record.TraceID,
		Method: record.Method, Route: record.Route, StatusCode: record.StatusCode,
		SourceRevision: record.SourceRevision,
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return errors.New("encode structured log")
	}
	var line bytes.Buffer
	line.Write(encoded)
	line.WriteByte('\n')
	logger.mu.Lock()
	defer logger.mu.Unlock()
	_, err = logger.writer.Write(line.Bytes())
	if err != nil {
		return errors.New("write structured log")
	}
	return nil
}

// FieldPolicy is the executable inventory for the only fields accepted by the
// foundation sink. All are operational metadata and have bounded retention.
type FieldPolicy struct {
	Classification string
	RetentionDays  int
}

func FieldPolicies() map[string]FieldPolicy {
	return map[string]FieldPolicy{
		"timestamp":        {Classification: "internal", RetentionDays: 30},
		"event":            {Classification: "internal", RetentionDays: 30},
		"severity":         {Classification: "internal", RetentionDays: 30},
		"module":           {Classification: "internal", RetentionDays: 30},
		"outcome":          {Classification: "internal", RetentionDays: 30},
		"request_id":       {Classification: "confidential-pseudonymous", RetentionDays: 30},
		"correlation_id":   {Classification: "confidential-pseudonymous", RetentionDays: 30},
		"trace_id":         {Classification: "confidential-pseudonymous", RetentionDays: 30},
		"http_method":      {Classification: "internal", RetentionDays: 30},
		"http_route":       {Classification: "internal", RetentionDays: 30},
		"http_status_code": {Classification: "internal", RetentionDays: 30},
		"source_revision":  {Classification: "internal", RetentionDays: 90},
	}
}
