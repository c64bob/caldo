package model

import (
	"strings"
	"testing"
)

func TestPatchVTODOPreservesUnknownValarmAttachAndRRULE(t *testing.T) {
	raw := strings.Join([]string{
		"BEGIN:VCALENDAR",
		"BEGIN:VTODO",
		"UID:uid-1",
		"SUMMARY:old",
		"RRULE:FREQ=WEEKLY;BYDAY=MO",
		"X-UNKNOWN:keep-me",
		"ATTACH:https://example.com/file.txt",
		"BEGIN:VALARM",
		"ACTION:DISPLAY",
		"TRIGGER:-PT15M",
		"END:VALARM",
		"END:VTODO",
		"END:VCALENDAR",
	}, "\n")

	newSummary := "new summary"
	patched := PatchVTODO(raw, VTODOPatch{Summary: &newSummary})

	for _, expected := range []string{
		"X-UNKNOWN:keep-me",
		"ATTACH:https://example.com/file.txt",
		"BEGIN:VALARM",
		"ACTION:DISPLAY",
		"TRIGGER:-PT15M",
		"END:VALARM",
		"RRULE:FREQ=WEEKLY;BYDAY=MO",
		"SUMMARY:new summary",
	} {
		if !strings.Contains(patched, expected) {
			t.Fatalf("expected patched VTODO to contain %q\npatched=%s", expected, patched)
		}
	}
}

func TestPatchVTODOChangesRRULEOnlyWhenExplicitlyPatched(t *testing.T) {
	raw := strings.Join([]string{
		"BEGIN:VTODO",
		"UID:uid-1",
		"SUMMARY:old",
		"RRULE:FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1",
		"END:VTODO",
	}, "\n")

	summary := "updated"
	withoutRRulePatch := PatchVTODO(raw, VTODOPatch{Summary: &summary})
	if !strings.Contains(withoutRRulePatch, "RRULE:FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1") {
		t.Fatalf("expected RRULE to remain unchanged when repeat is not explicitly patched\npatched=%s", withoutRRulePatch)
	}

	newRRule := "FREQ=DAILY"
	withRRulePatch := PatchVTODO(raw, VTODOPatch{Summary: &summary, RRule: &newRRule})
	if strings.Contains(withRRulePatch, "RRULE:FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1") {
		t.Fatalf("expected old RRULE to be replaced when repeat is explicitly patched\npatched=%s", withRRulePatch)
	}
	if !strings.Contains(withRRulePatch, "RRULE:FREQ=DAILY") {
		t.Fatalf("expected new RRULE to be present\npatched=%s", withRRulePatch)
	}
}
