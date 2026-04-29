package model

import "strings"

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
