package model

import (
	"strings"
	"time"
)

func mergeComparable[T comparable](base, local, remote T) (T, bool, bool) {
	localChanged := local != base
	remoteChanged := remote != base
	if !localChanged && !remoteChanged {
		var zero T
		return zero, false, true
	}
	if localChanged && !remoteChanged {
		return local, true, true
	}
	if !localChanged && remoteChanged {
		return remote, true, true
	}
	if local == remote {
		return local, true, true
	}
	var zero T
	return zero, false, false
}

func mergeOptionalString(base, local, remote *string) (*string, bool, bool) {
	return mergeOptional(base, local, remote, func(a, b string) bool { return a == b })
}

func mergeOptionalInt(base, local, remote *int) (*int, bool, bool) {
	return mergeOptional(base, local, remote, func(a, b int) bool { return a == b })
}

func mergeOptionalTime(base, local, remote *time.Time) (*time.Time, bool, bool) {
	return mergeOptional(base, local, remote, func(a, b time.Time) bool { return a.Equal(b) })
}

func mergeOptional[T any](base, local, remote *T, equal func(a, b T) bool) (*T, bool, bool) {
	localChanged := optionalChanged(base, local, equal)
	remoteChanged := optionalChanged(base, remote, equal)
	if !localChanged && !remoteChanged {
		return nil, false, true
	}
	if localChanged && !remoteChanged {
		return local, true, true
	}
	if !localChanged && remoteChanged {
		return remote, true, true
	}
	if optionalEqual(local, remote, equal) {
		return local, true, true
	}
	return nil, false, false
}

func optionalChanged[T any](base, value *T, equal func(a, b T) bool) bool {
	return !optionalEqual(base, value, equal)
}

func optionalEqual[T any](a, b *T, equal func(a, b T) bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return equal(*a, *b)
}

func joinSorted(values []string) string {
	trimmed := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			trimmed = append(trimmed, v)
		}
	}
	for i := 0; i < len(trimmed)-1; i++ {
		for j := i + 1; j < len(trimmed); j++ {
			if trimmed[j] < trimmed[i] {
				trimmed[i], trimmed[j] = trimmed[j], trimmed[i]
			}
		}
	}
	return strings.Join(trimmed, ",")
}

func splitJoined(raw string) []string { return strings.Split(raw, ",") }
