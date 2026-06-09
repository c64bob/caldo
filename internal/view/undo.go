package view

import "strings"

// UndoSnapshotView contains the visible state for one available undo action.
type UndoSnapshotView struct {
	ActionType   string
	ExpiresAtISO string
}

func undoSnapshotTitle(snapshot UndoSnapshotView) string {
	switch strings.TrimSpace(snapshot.ActionType) {
	case "task_deleted":
		return "Aufgabe gelöscht."
	default:
		return "Letzte Änderung gespeichert."
	}
}

func undoSnapshotDescription(snapshot UndoSnapshotView) string {
	switch strings.TrimSpace(snapshot.ActionType) {
	case "task_deleted":
		return "Wiederherstellung ist kurz verfügbar."
	default:
		return "Rückgängig ist kurz verfügbar."
	}
}
