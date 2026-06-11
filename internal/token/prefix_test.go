package token

import "testing"

func TestReadPrefixedStopsAtSharedDelimiters(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantPos int
	}{
		{name: "space", input: "#Work today", want: "#Work", wantPos: 5},
		{name: "colon", input: "#Work:today", want: "#Work", wantPos: 5},
		{name: "label", input: "@home#Work", want: "@home", wantPos: 5},
		{name: "empty", input: "#", want: "#", wantPos: 1},
	}

	for _, tc := range tests {
		got, gotPos := ReadPrefixed(tc.input, 0, tc.input[0])
		if got != tc.want || gotPos != tc.wantPos {
			t.Fatalf("%s: got literal=%q pos=%d want literal=%q pos=%d", tc.name, got, gotPos, tc.want, tc.wantPos)
		}
	}
}
