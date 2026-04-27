package model

import "testing"

func TestAppName(t *testing.T) {
	if AppName != "caldo" {
		t.Fatalf("unexpected app name: %s", AppName)
	}
}
