package log

import "testing"

func TestLevel_String_ReturnsCorrectName(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{Trace, "TRACE"},
		{Debug, "DEBUG"},
		{Info, "INFO"},
		{Warn, "WARN"},
		{Error, "ERROR"},
		{Fatal, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.expected {
				t.Errorf("Level.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLevel_String_UnknownLevel_ReturnsUnknown(t *testing.T) {
	unknown := Level(99)
	got := unknown.String()
	if got != "UNKNOWN" {
		t.Errorf("unknown Level.String() = %q, want %q", got, "UNKNOWN")
	}
}

func TestParseLevel_ValidLevels_ReturnsCorrectLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"trace", Trace},
		{"TRACE", Trace},
		{"Trace", Trace},
		{"debug", Debug},
		{"DEBUG", Debug},
		{"info", Info},
		{"INFO", Info},
		{"warn", Warn},
		{"WARN", Warn},
		{"warning", Warn},
		{"error", Error},
		{"ERROR", Error},
		{"fatal", Fatal},
		{"FATAL", Fatal},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if err != nil {
				t.Errorf("ParseLevel(%q) error = %v, want nil", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseLevel_InvalidLevel_ReturnsError(t *testing.T) {
	_, err := ParseLevel("invalid")
	if err == nil {
		t.Error("ParseLevel(\"invalid\") error = nil, want error")
	}
}

func TestLevel_IsEnabled_HigherOrEqualLevelReturnsTrue(t *testing.T) {
	tests := []struct {
		current  Level
		check    Level
		expected bool
	}{
		{Info, Debug, false},  // Debug is lower than Info
		{Info, Info, true},    // Same level
		{Info, Warn, true},    // Warn is higher than Info
		{Info, Error, true},   // Error is higher than Info
		{Debug, Trace, false}, // Trace is lower than Debug
		{Trace, Trace, true},  // Same level
	}

	for _, tt := range tests {
		name := tt.current.String() + "_enables_" + tt.check.String()
		t.Run(name, func(t *testing.T) {
			got := tt.current.Enables(tt.check)
			if got != tt.expected {
				t.Errorf("%s.Enables(%s) = %v, want %v", tt.current, tt.check, got, tt.expected)
			}
		})
	}
}
