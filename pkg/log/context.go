package log

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey int

const (
	requestIDKey contextKey = iota
	fieldsKey
)

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext extracts the request ID from context.
// Returns empty string if not found or context is nil.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// WithFields adds structured fields to the context.
// Fields are merged with any existing fields.
func WithFields(ctx context.Context, keysAndValues ...any) context.Context {
	existing := FieldsFromContext(ctx)
	fields := make(map[string]any)

	// Copy existing fields
	for k, v := range existing {
		fields[k] = v
	}

	// Add new fields
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			fields[key] = keysAndValues[i+1]
		}
	}

	return context.WithValue(ctx, fieldsKey, fields)
}

// FieldsFromContext extracts structured fields from context.
// Returns nil if no fields are set.
func FieldsFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	fields, _ := ctx.Value(fieldsKey).(map[string]any)
	return fields
}
