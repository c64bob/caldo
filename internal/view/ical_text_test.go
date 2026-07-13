package view

import (
	"testing"

	"caldo/internal/model"
)

func TestICalendarTextDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "lowercase newline", input: `line 1\nline 2`, want: "line 1\nline 2"},
		{name: "uppercase newline", input: `line 1\Nline 2`, want: "line 1\nline 2"},
		{name: "text punctuation", input: `one\, two\; three`, want: "one, two; three"},
		{name: "backslash", input: `path\\segment`, want: `path\segment`},
		{name: "all escapes", input: `one\ntwo\Nthree\, four\; five\\six`, want: "one\ntwo\nthree, four; five\\six"},
		{name: "physical line endings", input: "one\r\ntwo\rthree", want: "one\ntwo\nthree"},
		{name: "unknown escape", input: `one\qtwo\:three`, want: `one\qtwo\:three`},
		{name: "trailing backslash", input: `one\`, want: `one\`},
		{name: "escaped backslash before n", input: `one\\n two`, want: `one\n two`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := iCalendarTextDisplay(tt.input); got != tt.want {
				t.Fatalf("iCalendarTextDisplay(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTaskTextUsesDecodedDisplayAndRawEditValues(t *testing.T) {
	t.Parallel()

	task := TaskRowView{
		Title:       `Plan\, phase 2\; team\\blue`,
		Description: `First\nSecond\, item`,
		ParentTitle: `Parent\, task`,
		IsSubtask:   true,
	}

	if got, want := taskRowTitle(task), `Plan, phase 2; team\blue`; got != want {
		t.Fatalf("taskRowTitle() = %q, want %q", got, want)
	}
	if got := taskEditableTitle(task); got != task.Title {
		t.Fatalf("taskEditableTitle() = %q, want raw %q", got, task.Title)
	}
	if got, want := taskDescriptionDisplayText(task.Description), "First\nSecond, item"; got != want {
		t.Fatalf("taskDescriptionDisplayText() = %q, want %q", got, want)
	}
	if got, want := taskRelationshipChips(task)[0].Label, "Unteraufgabe von Parent, task"; got != want {
		t.Fatalf("relationship label = %q, want %q", got, want)
	}
}

func TestConflictTextUsesDecodedDisplayAndRawManualValues(t *testing.T) {
	t.Parallel()

	fields := model.VTODOFields{
		Title:       `Plan\, phase 2`,
		Description: `First\nSecond\; item`,
	}

	if got, want := conflictTitleValue(fields), "Plan, phase 2"; got != want {
		t.Fatalf("conflictTitleValue() = %q, want %q", got, want)
	}
	if got, want := conflictDescriptionValue(fields), "First\nSecond; item"; got != want {
		t.Fatalf("conflictDescriptionValue() = %q, want %q", got, want)
	}
	for _, field := range []string{"title", "description"} {
		input := conflictManualInputForField(field, fields)
		if input.Value != conflictManualInputValue(field, fields) {
			t.Fatalf("manual %s value = %q, want raw %q", field, input.Value, conflictManualInputValue(field, fields))
		}
		if input.DisplayValue == input.Value {
			t.Fatalf("manual %s display value should be decoded: %q", field, input.DisplayValue)
		}
	}
}
