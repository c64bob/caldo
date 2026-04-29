package query

import (
	"errors"
	"testing"
	"time"
)

type unknownNode struct{}

func (unknownNode) node() {}

func TestCompileFilter_LeafOperators(t *testing.T) {
	now := time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		node  FilterNode
		where string
		args  []any
	}{
		{"today", FilterNode{Operator: TokenToday}, "due_date = ?", []any{"2026-04-29"}},
		{"overdue", FilterNode{Operator: TokenOverdue}, "due_date < ?", []any{"2026-04-29"}},
		{"upcoming", FilterNode{Operator: TokenUpcoming}, "due_date BETWEEN ? AND ?", []any{"2026-04-29", "2026-05-06"}},
		{"no-date", FilterNode{Operator: TokenNoDate}, "due_date IS NULL", nil},
		{"project", FilterNode{Operator: TokenProject, Value: "#Work"}, "project_name = ?", []any{"Work"}},
		{"label", FilterNode{Operator: TokenLabel, Value: "@Home"}, "label_names LIKE ?", []any{"%Home%"}},
		{"priority", FilterNode{Operator: TokenPriority, Value: "high"}, "priority = ?", []any{"high"}},
		{"completed-true", FilterNode{Operator: TokenCompleted, Value: "true"}, "(status = 'completed' OR completed_at IS NOT NULL)", nil},
		{"completed-false", FilterNode{Operator: TokenCompleted, Value: "false"}, "(status != 'completed' AND completed_at IS NULL)", nil},
		{"text", FilterNode{Operator: TokenText, Value: "invoice"}, "rowid IN (SELECT rowid FROM tasks_fts WHERE tasks_fts MATCH ?)", []any{"invoice*"}},
		{"before", FilterNode{Operator: TokenBefore, Value: "2026-05-01"}, "due_date < ?", []any{"2026-05-01"}},
		{"after", FilterNode{Operator: TokenAfter, Value: "2026-05-01"}, "due_date > ?", []any{"2026-05-01"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			where, args, err := CompileFilter(tc.node, CompileOptions{Now: now})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if where != tc.where {
				t.Fatalf("where=%q want %q", where, tc.where)
			}
			if len(args) != len(tc.args) {
				t.Fatalf("args len=%d want %d", len(args), len(tc.args))
			}
			for i := range tc.args {
				if args[i] != tc.args[i] {
					t.Fatalf("arg[%d]=%v want %v", i, args[i], tc.args[i])
				}
			}
		})
	}
}

func TestCompileFilter_LogicNodes(t *testing.T) {
	now := time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC)
	node := OrNode{Left: AndNode{Left: FilterNode{Operator: TokenToday}, Right: NotNode{Expr: FilterNode{Operator: TokenNoDate}}}, Right: FilterNode{Operator: TokenPriority, Value: "low"}}
	where, args, err := CompileFilter(node, CompileOptions{Now: now})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if where != "((due_date = ? AND (NOT due_date IS NULL)) OR priority = ?)" {
		t.Fatalf("where=%q", where)
	}
	if len(args) != 2 || args[0] != "2026-04-29" || args[1] != "low" {
		t.Fatalf("args=%v", args)
	}
}

func TestCompileFilter_UnknownProjectAndLabelBecomeFalse(t *testing.T) {
	projectFalse := func(string) bool { return false }
	labelFalse := func(string) bool { return false }

	where, args, err := CompileFilter(FilterNode{Operator: TokenProject, Value: "#missing"}, CompileOptions{ResolveProject: projectFalse})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if where != "1=0" || len(args) != 0 {
		t.Fatalf("where=%q args=%v", where, args)
	}

	where, args, err = CompileFilter(FilterNode{Operator: TokenLabel, Value: "@missing"}, CompileOptions{ResolveLabel: labelFalse})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if where != "1=0" || len(args) != 0 {
		t.Fatalf("where=%q args=%v", where, args)
	}
}

func TestCompileFilter_UnknownNodeAndOperatorErrors(t *testing.T) {
	_, _, err := CompileFilter(unknownNode{}, CompileOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("error type=%T want *CompileError", err)
	}

	_, _, err = CompileFilter(FilterNode{Operator: TokenString, Value: "x"}, CompileOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &compileErr) {
		t.Fatalf("error type=%T want *CompileError", err)
	}
}
