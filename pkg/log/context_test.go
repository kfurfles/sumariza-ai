package log

import (
	"context"
	"testing"
)

func TestWithRequestID_AddsToContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")

	id := RequestIDFromContext(ctx)
	if id != "req-123" {
		t.Errorf("RequestIDFromContext() = %q, want %q", id, "req-123")
	}
}

func TestRequestIDFromContext_NoID_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()

	id := RequestIDFromContext(ctx)
	if id != "" {
		t.Errorf("RequestIDFromContext() = %q, want empty", id)
	}
}

func TestRequestIDFromContext_NilContext_ReturnsEmpty(t *testing.T) {
	id := RequestIDFromContext(nil)
	if id != "" {
		t.Errorf("RequestIDFromContext(nil) = %q, want empty", id)
	}
}

func TestWithRequestID_OverwritesPrevious(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "first")
	ctx = WithRequestID(ctx, "second")

	id := RequestIDFromContext(ctx)
	if id != "second" {
		t.Errorf("RequestIDFromContext() = %q, want %q", id, "second")
	}
}

func TestWithFields_AddsFieldsToContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithFields(ctx, "service", "api", "version", "1.0")

	fields := FieldsFromContext(ctx)
	if fields["service"] != "api" {
		t.Errorf("fields[service] = %v, want %q", fields["service"], "api")
	}
	if fields["version"] != "1.0" {
		t.Errorf("fields[version] = %v, want %q", fields["version"], "1.0")
	}
}

func TestFieldsFromContext_NoFields_ReturnsNil(t *testing.T) {
	ctx := context.Background()

	fields := FieldsFromContext(ctx)
	if fields != nil {
		t.Errorf("FieldsFromContext() = %v, want nil", fields)
	}
}

func TestWithFields_MergesWithExisting(t *testing.T) {
	ctx := context.Background()
	ctx = WithFields(ctx, "a", "1")
	ctx = WithFields(ctx, "b", "2")

	fields := FieldsFromContext(ctx)
	if fields["a"] != "1" {
		t.Errorf("fields[a] = %v, want %q", fields["a"], "1")
	}
	if fields["b"] != "2" {
		t.Errorf("fields[b] = %v, want %q", fields["b"], "2")
	}
}
