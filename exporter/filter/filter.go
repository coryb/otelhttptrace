package filter

import (
	"context"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Filter is a filter function that will be applied to each span via
// Spans.  If the span argument is nil then the Filter must return nil and
// this span will be removed from the list returned from Spans.
type Filter func(span sdktrace.ReadOnlySpan) sdktrace.ReadOnlySpan

// Spans takes a list of ReadOnlySpans and applies the filter each span,
// returning a new list of spans.
func Spans(spans []sdktrace.ReadOnlySpan, filters ...Filter) []sdktrace.ReadOnlySpan {
	if len(filters) == 0 {
		return spans
	}

	filtered := make([]sdktrace.ReadOnlySpan, len(spans), 0)
	for _, span := range spans {
		for _, f := range filters {
			span = f(span)
		}
		if span != nil {
			filtered = append(filtered, span)
		}
	}
	return filtered
}

type contextCancelNormal struct {
	sdktrace.ReadOnlySpan
}

func (s contextCancelNormal) Status() sdktrace.Status {
	return sdktrace.Status{
		Code: codes.Ok,
	}
}

func (s contextCancelNormal) Attributes() []attribute.KeyValue {
	return append(s.ReadOnlySpan.Attributes(), attribute.String("message", context.Canceled.Error()))
}

// ContextCancelNormal will downgrade spans "errors" that were caused by
// context.Canceled
func ContextCancelNormal() Filter {
	return func(span sdktrace.ReadOnlySpan) sdktrace.ReadOnlySpan {
		if span == nil {
			return nil
		}
		status := span.Status()
		if status.Code == codes.Error && status.Description == context.Canceled.Error() {
			return contextCancelNormal{span}
		}
		return span
	}
}

type httpStatusNormal struct {
	sdktrace.ReadOnlySpan
}

func (s httpStatusNormal) Status() sdktrace.Status {
	return sdktrace.Status{
		Code: codes.Ok,
	}
}

// HTTPStatusNormal will convert errors spans with provided status code into
// non-error "normal" spans.  For example you might want to allow 404 status
// codes as valid response and now have those spans show up as errors.
func HTTPStatusNormal(status int) Filter {
	statusCodeString := strconv.Itoa(status)
	return func(span sdktrace.ReadOnlySpan) sdktrace.ReadOnlySpan {
		if span == nil {
			return nil
		}
		if span.Status().Code == codes.Error {
			for _, attr := range span.Attributes() {
				if attr.Key == semconv.HTTPStatusCodeKey && attr.Value.Emit() == statusCodeString {
					return httpStatusNormal{span}
				}
			}
		}
		return span
	}
}