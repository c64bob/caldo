package model

import (
	"strings"
	"testing"
)

func TestMergeVTODOFieldsNoBaseDisablesAutoMerge(t *testing.T) {
	result := MergeVTODOFields("", "BEGIN:VTODO\nSUMMARY:l\nEND:VTODO", "BEGIN:VTODO\nSUMMARY:r\nEND:VTODO")
	if !result.Conflict || result.Merged {
		t.Fatalf("expected conflict without base")
	}
}

func TestMergeVTODOFieldsAutoMergesConflictFreeChanges(t *testing.T) {
	base := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:one\nDESCRIPTION:old\nEND:VTODO\nEND:VCALENDAR"
	local := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:local\nDESCRIPTION:old\nEND:VTODO\nEND:VCALENDAR"
	remote := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:one\nDESCRIPTION:remote\nEND:VTODO\nEND:VCALENDAR"
	result := MergeVTODOFields(base, local, remote)
	if !result.Merged || result.Conflict {
		t.Fatalf("expected merged result")
	}
	if !strings.Contains(result.MergedVTODO, "SUMMARY:local") || !strings.Contains(result.MergedVTODO, "DESCRIPTION:remote") {
		t.Fatalf("unexpected merged payload: %q", result.MergedVTODO)
	}
}

func TestMergeVTODOFieldsDetectsFieldConflict(t *testing.T) {
	base := "BEGIN:VTODO\nSUMMARY:one\nEND:VTODO"
	local := "BEGIN:VTODO\nSUMMARY:local\nEND:VTODO"
	remote := "BEGIN:VTODO\nSUMMARY:remote\nEND:VTODO"
	result := MergeVTODOFields(base, local, remote)
	if !result.Conflict || result.Merged {
		t.Fatalf("expected conflict for same-field divergence")
	}
}

func TestMergeVTODOFieldsDoesNotConflictOnEqualPointerFieldValues(t *testing.T) {
	base := "BEGIN:VTODO\nSUMMARY:one\nDUE:20260501T120000Z\nPRIORITY:5\nEND:VTODO"
	local := "BEGIN:VTODO\nSUMMARY:local\nDUE:20260501T120000Z\nPRIORITY:5\nEND:VTODO"
	remote := "BEGIN:VTODO\nSUMMARY:one\nDUE:20260501T120000Z\nPRIORITY:5\nDESCRIPTION:remote\nEND:VTODO"
	result := MergeVTODOFields(base, local, remote)
	if !result.Merged || result.Conflict {
		t.Fatalf("expected merged result for equal pointer-backed values")
	}
	if !strings.Contains(result.MergedVTODO, "SUMMARY:local") || !strings.Contains(result.MergedVTODO, "DESCRIPTION:remote") {
		t.Fatalf("unexpected merged payload: %q", result.MergedVTODO)
	}
}

func TestMergeVTODOFieldsClearsOptionalFieldWhenMergedToNil(t *testing.T) {
	base := "BEGIN:VTODO\nSUMMARY:one\nDUE:20260501T120000Z\nEND:VTODO"
	local := "BEGIN:VTODO\nSUMMARY:one\nEND:VTODO"
	remote := "BEGIN:VTODO\nSUMMARY:one\nEND:VTODO"
	result := MergeVTODOFields(base, local, remote)
	if !result.Merged || result.Conflict {
		t.Fatalf("expected merged result")
	}
	if strings.Contains(result.MergedVTODO, "DUE:") {
		t.Fatalf("expected merged payload to clear DUE field: %q", result.MergedVTODO)
	}
}
