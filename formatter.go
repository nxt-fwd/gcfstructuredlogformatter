package gcfstructuredlogformatter

import (
	"encoding/json"
	"fmt"

	"cloud.google.com/go/logging"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// ContextKey is the type for the context key.
// The Go docs recommend not using any built-in type for context keys in order
// to ensure that there are no collisions:
//
//	https://golang.org/pkg/context/#WithValue
type ContextKey string

// ContextKey constants.
const (
	ContextKeyTrace ContextKey = "trace" // This is the key for the trace identifier.
)

// logrusToGoogleSeverityMap maps a logrus level to a Google severity.
var logrusToGoogleSeverityMap = map[logrus.Level]logging.Severity{
	logrus.PanicLevel: logging.Emergency,
	logrus.FatalLevel: logging.Alert,
	logrus.ErrorLevel: logging.Error,
	logrus.WarnLevel:  logging.Warning,
	logrus.InfoLevel:  logging.Info,
	logrus.DebugLevel: logging.Debug,
	logrus.TraceLevel: logging.Default,
}

// Formatter is the logrus formatter.
type Formatter struct {
	Labels map[string]string // This is an optional map of additional "labels".
}

// logEntry is an abbreviated version of the Google "structured logging" data structure.
type logEntry struct {
	Severity    string            `json:"severity,omitempty"`
	Trace       string            `json:"logging.googleapis.com/trace,omitempty"`
	SpanID      string            `json:"logging.googleapis.com/spanId,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	JSONPayload logrus.Fields     `json:"jsonPayload"`
}

// New creates a new formatter.
func New() *Formatter {
	f := &Formatter{
		Labels: map[string]string{},
	}
	return f
}

// Levels are the available logging levels.
func (f *Formatter) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
}

// Format an entry.
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	severity := logging.Default
	if value, okay := logrusToGoogleSeverityMap[entry.Level]; okay {
		severity = value
	}

	newEntry := logEntry{
		Severity: severity.String(),
		Labels:   map[string]string{},
	}
	if entry.Context != nil {
		// try to get the trace id from the context
		span := trace.SpanFromContext(entry.Context)
		spanContext := span.SpanContext()
		if spanContext.IsValid() {
			newEntry.SpanID = spanContext.SpanID().String()
			newEntry.Trace = spanContext.TraceID().String()
		}
	}
	for key, value := range f.Labels {
		newEntry.Labels[key] = value
	}

	newEntry.JSONPayload = entry.Data
	newEntry.JSONPayload["message"] = entry.Message // This is the log message.

	if severity == logging.Error && entry.Caller != nil {
		newEntry.JSONPayload["exception"] = fmt.Sprintf("%s\n\t%s:%d\n", entry.Caller.Function, entry.Caller.File, entry.Caller.Line)
	}
	contents, err := json.Marshal(newEntry)
	if err != nil {
		return nil, err
	}
	return append(contents, []byte("\n")...), nil
}
