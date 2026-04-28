package model

import (
	"strings"
	"time"
)

// VTODOFields contains normalized task fields extracted from a raw VTODO payload.
type VTODOFields struct {
	UID         string
	Title       string
	Description string
	Status      string
	CompletedAt *time.Time
	DueDate     *string
	DueAt       *time.Time
	Priority    *int
	RRule       string
	ParentUID   string
	Categories  []string
}

// ParseVTODOFields extracts normalized fields from a raw iCalendar VTODO payload.
func ParseVTODOFields(raw string) VTODOFields {
	fields := VTODOFields{Status: "needs-action"}
	lines := unfoldICalendarLines(raw)
	inVTODO := false

	for _, line := range lines {
		name, value, params, ok := splitPropertyLine(line)
		if !ok {
			continue
		}

		switch name {
		case "BEGIN":
			if strings.EqualFold(value, "VTODO") {
				inVTODO = true
			}
		case "END":
			if strings.EqualFold(value, "VTODO") {
				return fields
			}
		}

		if !inVTODO {
			continue
		}

		switch name {
		case "UID":
			if fields.UID == "" {
				fields.UID = strings.TrimSpace(value)
			}
		case "SUMMARY":
			if fields.Title == "" {
				fields.Title = strings.TrimSpace(value)
			}
		case "DESCRIPTION":
			if fields.Description == "" {
				fields.Description = strings.TrimSpace(value)
			}
		case "STATUS":
			status := strings.ToLower(strings.TrimSpace(value))
			if status != "" {
				fields.Status = status
			}
		case "COMPLETED":
			if ts, ok := parseICalDateTime(value); ok {
				fields.CompletedAt = &ts
			}
		case "DUE":
			if propertyParamEquals(params, "VALUE", "DATE") {
				if date := strings.TrimSpace(value); date != "" {
					if formatted, ok := parseICalDate(date); ok {
						fields.DueDate = &formatted
					}
				}
				break
			}
			if ts, ok := parseICalDateTime(value); ok {
				fields.DueAt = &ts
			}
		case "PRIORITY":
			if priority, ok := parseICalInt(value); ok {
				fields.Priority = &priority
			}
		case "RRULE":
			if fields.RRule == "" {
				fields.RRule = strings.TrimSpace(value)
			}
		case "RELATED-TO":
			if propertyParamEquals(params, "RELTYPE", "PARENT") && fields.ParentUID == "" {
				fields.ParentUID = strings.TrimSpace(value)
			}
		case "CATEGORIES":
			for _, category := range strings.Split(value, ",") {
				category = strings.TrimSpace(category)
				if category != "" {
					fields.Categories = append(fields.Categories, category)
				}
			}
		}
	}

	return fields
}

func unfoldICalendarLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	split := strings.Split(raw, "\n")
	lines := make([]string, 0, len(split))
	for _, line := range split {
		if len(lines) > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			lines[len(lines)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func splitPropertyLine(line string) (name, value string, params map[string]string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx <= 0 {
		return "", "", nil, false
	}

	rawName := line[:idx]
	params = parsePropertyParams(rawName)
	if semi := strings.IndexByte(rawName, ';'); semi >= 0 {
		rawName = rawName[:semi]
	}

	return strings.ToUpper(strings.TrimSpace(rawName)), strings.TrimSpace(line[idx+1:]), params, true
}

func parsePropertyParams(rawName string) map[string]string {
	parts := strings.Split(rawName, ";")
	if len(parts) <= 1 {
		return nil
	}

	params := make(map[string]string, len(parts)-1)
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, hasValue := strings.Cut(part, "=")
		if !hasValue {
			continue
		}
		params[strings.ToUpper(strings.TrimSpace(key))] = strings.ToUpper(strings.Trim(strings.TrimSpace(value), `"`))
	}
	return params
}

func propertyParamEquals(params map[string]string, key string, expected string) bool {
	if len(params) == 0 {
		return false
	}
	value, ok := params[strings.ToUpper(strings.TrimSpace(key))]
	return ok && value == strings.ToUpper(strings.TrimSpace(expected))
}

func parseICalDateTime(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	formats := []string{"20060102T150405Z", "20060102T150405", "20060102T1504Z", "20060102T1504"}
	for _, format := range formats {
		ts, err := time.Parse(format, value)
		if err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func parseICalDate(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	ts, err := time.Parse("20060102", value)
	if err != nil {
		return "", false
	}
	return ts.Format("2006-01-02"), true
}

func parseICalInt(raw string) (int, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}
	var parsed int
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
		parsed = parsed*10 + int(r-'0')
	}
	return parsed, true
}
