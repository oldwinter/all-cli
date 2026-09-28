package featureflags

import (
	"context"
	"strings"
	"testing"
)

func TestParseFeatureFlags(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		enabled bool
		wantErr string
	}{
		{name: "blank", raw: ""},
		{name: "known", raw: "telemetry-v1", enabled: true},
		{name: "whitespace and duplicate", raw: " telemetry-v1, telemetry-v1 ", enabled: true},
		{name: "unknown", raw: "future-z,future-a", wantErr: "future-a, future-z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := Parse(tt.raw)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Parse(%q) error = %v, want containing %q", tt.raw, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.raw, err)
			}
			ctx := WithContext(context.Background(), set)
			if got := Enabled(ctx, TelemetryV1); got != tt.enabled {
				t.Fatalf("Enabled(TelemetryV1) = %t, want %t", got, tt.enabled)
			}
		})
	}
}

func TestEnabledWithoutSetOrNilContext(t *testing.T) {
	var nilCtx context.Context
	if Enabled(nilCtx, TelemetryV1) {
		t.Fatal("Enabled on nil context = true, want false")
	}
	if Enabled(context.Background(), TelemetryV1) {
		t.Fatal("Enabled without set in context = true, want false")
	}
	set, _ := Parse("")
	ctx := WithContext(context.Background(), set)
	if Enabled(ctx, TelemetryV1) {
		t.Fatal("Enabled for unset flag = true, want false")
	}
}

func TestWithContextNilContext(t *testing.T) {
	set, err := Parse("telemetry-v1")
	if err != nil {
		t.Fatal(err)
	}
	var nilCtx context.Context
	ctx := WithContext(nilCtx, set)
	if !Enabled(ctx, TelemetryV1) {
		t.Fatal("WithContext(nil) should behave like Background")
	}
}

func TestWithContextCopiesSet(t *testing.T) {
	set, err := Parse("telemetry-v1")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	ctx := WithContext(context.Background(), set)

	delete(set.enabled, TelemetryV1)

	if !Enabled(ctx, TelemetryV1) {
		t.Fatal("mutating the caller's set changed the context feature flags")
	}
}
