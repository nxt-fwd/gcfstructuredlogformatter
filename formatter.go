package gcfstructuredlogformatter

import (
	"encoding/json"

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

	mapEntry := map[string]interface{}{}
	mapEntry["severity"] = severity.String()
	mapEntry["message"] = entry.Message

	if entry.Context != nil {
		// try to get the trace id from the context
		span := trace.SpanFromContext(entry.Context)
		spanContext := span.SpanContext()
		if spanContext.IsValid() {
			mapEntry["logging.googleapis.com/trace"] = spanContext.TraceID().String()
			mapEntry["logging.googleapis.com/spanId"] = spanContext.SpanID().String()
		}
	}
	if len(f.Labels) > 0 {
		labels := map[string]string{}
		for key, value := range f.Labels {
			labels[key] = value
		}
		mapEntry["labels"] = labels
	}

	for key, value := range entry.Data {
		mapEntry[key] = value
	}
	contents, err := json.Marshal(mapEntry)
	if err != nil {
		return nil, err
	}
	return append(contents, []byte("\n")...), nil
}
