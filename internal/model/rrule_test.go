package model

import "testing"

func TestIsComplexRRule(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		rule string
		want bool
	}{
		{name: "empty is simple", rule: "", want: false},
		{name: "daily simple", rule: "FREQ=DAILY", want: false},
		{name: "weekly weekdays simple", rule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", want: false},
		{name: "monthly interval count simple", rule: "FREQ=MONTHLY;INTERVAL=2;COUNT=5", want: false},
		{name: "daily byday is complex", rule: "FREQ=DAILY;BYDAY=MO,TU", want: true},
		{name: "count and until is complex", rule: "FREQ=WEEKLY;COUNT=4;UNTIL=20260307T235959Z", want: true},
		{name: "set position is complex", rule: "FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1", want: true},
		{name: "ordinal byday is complex", rule: "FREQ=MONTHLY;BYDAY=1MO,3MO", want: true},
		{name: "monthday is complex", rule: "FREQ=MONTHLY;BYMONTHDAY=15,30", want: true},
		{name: "bad interval is complex", rule: "FREQ=MONTHLY;INTERVAL=A", want: true},
		{name: "unknown freq is complex", rule: "FREQ=HOURLY", want: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := IsComplexRRule(tc.rule)
			if got != tc.want {
				t.Fatalf("IsComplexRRule(%q) = %v, want %v", tc.rule, got, tc.want)
			}
		})
	}
}
