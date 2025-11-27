package otel

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Error type constants for structured error recording
const (
	ErrorTypeNetwork    = "network"
	ErrorTypeHTTP       = "http"
	ErrorTypeParse      = "parse"
	ErrorTypeValidation = "validation"
)

// RecordError records an error on a span with structured attributes and sets the span status to Error.
// This provides consistent error recording across all components with OTEL semantic conventions.
func RecordError(span trace.Span, err error, errorType string, transient bool) {
	span.RecordError(err, trace.WithAttributes(
		attribute.String("error.type", errorType),
		attribute.Bool("error.transient", transient),
	))
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanOk sets the span status to Ok, indicating successful completion.
func SetSpanOk(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}
