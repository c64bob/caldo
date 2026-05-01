package handler

import "testing"

func TestIsSupportedUILanguage(t *testing.T) {
	if !isSupportedUILanguage("de") || !isSupportedUILanguage("en") {
		t.Fatal("expected de and en to be supported")
	}
	if isSupportedUILanguage("fr") {
		t.Fatal("fr must not be supported")
	}
}

func TestIsSupportedDarkMode(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system"} {
		if !isSupportedDarkMode(mode) {
			t.Fatalf("expected supported mode %q", mode)
		}
	}
	if isSupportedDarkMode("amoled") {
		t.Fatal("unexpected custom mode support")
	}
}
