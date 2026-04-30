package model

import "strings"

var supportedRRULEKeys = map[string]struct{}{
	"FREQ":     {},
	"INTERVAL": {},
	"BYDAY":    {},
	"UNTIL":    {},
	"COUNT":    {},
}

var supportedBYDAYValues = map[string]struct{}{
	"MO": {},
	"TU": {},
	"WE": {},
	"TH": {},
	"FR": {},
	"SA": {},
	"SU": {},
}

// IsComplexRRule returns true when the recurrence rule contains tokens
// outside the MVP editor support.
func IsComplexRRule(rule string) bool {
	trimmed := strings.TrimSpace(rule)
	if trimmed == "" {
		return false
	}

	parts := strings.Split(trimmed, ";")
	var (
		freq       string
		hasUntil   bool
		hasCount   bool
		hasByDay   bool
		hasInvalid bool
	)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return true
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.ToUpper(strings.TrimSpace(value))

		if _, ok := supportedRRULEKeys[key]; !ok {
			return true
		}

		switch key {
		case "FREQ":
			if value != "DAILY" && value != "WEEKLY" && value != "MONTHLY" && value != "YEARLY" {
				return true
			}
			freq = value
		case "INTERVAL", "COUNT":
			if !isPositiveNumber(value) {
				return true
			}
			if key == "COUNT" {
				hasCount = true
			}
		case "BYDAY":
			if value == "" {
				return true
			}
			hasByDay = true
			days := strings.Split(value, ",")
			for _, day := range days {
				day = strings.TrimSpace(day)
				if _, ok := supportedBYDAYValues[day]; !ok {
					return true
				}
			}
		case "UNTIL":
			if value == "" {
				return true
			}
			hasUntil = true
		default:
			hasInvalid = true
		}
	}

	if hasInvalid {
		return true
	}
	if hasUntil && hasCount {
		return true
	}
	if hasByDay && freq != "WEEKLY" {
		return true
	}

	return false
}

func isPositiveNumber(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != "0"
}
