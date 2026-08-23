package featureflags

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type Flag string

const TelemetryV1 Flag = "telemetry-v1"

type Set struct {
	enabled map[Flag]struct{}
}

var supported = map[Flag]struct{}{
	TelemetryV1: {},
}

type contextKey struct{}

func Parse(raw string) (Set, error) {
	set := Set{enabled: make(map[Flag]struct{})}
	var unknown []string
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		flag := Flag(name)
		if _, ok := supported[flag]; !ok {
			unknown = append(unknown, name)
			continue
		}
		set.enabled[flag] = struct{}{}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return Set{}, fmt.Errorf("unknown ALL_CLI_FEATURES values: %s", strings.Join(unknown, ", "))
	}
	return set, nil
}

func WithContext(ctx context.Context, set Set) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	copySet := Set{enabled: make(map[Flag]struct{}, len(set.enabled))}
	for flag := range set.enabled {
		copySet.enabled[flag] = struct{}{}
	}
	return context.WithValue(ctx, contextKey{}, copySet)
}

func Enabled(ctx context.Context, flag Flag) bool {
	if ctx == nil {
		return false
	}
	set, ok := ctx.Value(contextKey{}).(Set)
	if !ok {
		return false
	}
	_, ok = set.enabled[flag]
	return ok
}
