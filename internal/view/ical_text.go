package view

import "strings"

func iCalendarTextDisplay(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	var builder strings.Builder
	builder.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			builder.WriteByte(value[i])
			continue
		}

		next := value[i+1]
		switch next {
		case 'n', 'N':
			builder.WriteByte('\n')
		case ',', ';', '\\':
			builder.WriteByte(next)
		default:
			builder.WriteByte(value[i])
			builder.WriteByte(next)
		}
		i++
	}
	return builder.String()
}
