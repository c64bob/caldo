package query

import (
	"fmt"
	"strings"
	"time"
)

// CompileError indicates an invalid or unsupported AST for SQL compilation.
type CompileError struct {
	Message string
}

func (e *CompileError) Error() string {
	return e.Message
}

// CompileOptions configures SQL compilation behavior.
type CompileOptions struct {
	Now            time.Time
	UpcomingDays   int
	ResolveProject func(name string) bool
	ResolveLabel   func(name string) bool
}

// CompileFilter compiles a filter AST to a parametrized SQL where fragment and args.
func CompileFilter(node Node, opts CompileOptions) (string, []any, error) {
	if opts.UpcomingDays <= 0 {
		opts.UpcomingDays = 7
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	c := &compiler{opts: opts}
	where, err := c.compile(node)
	if err != nil {
		return "", nil, err
	}
	return where, c.args, nil
}

type compiler struct {
	opts CompileOptions
	args []any
}

func (c *compiler) compile(node Node) (string, error) {
	switch n := node.(type) {
	case AndNode:
		left, err := c.compile(n.Left)
		if err != nil {
			return "", err
		}
		right, err := c.compile(n.Right)
		if err != nil {
			return "", err
		}
		return "(" + left + " AND " + right + ")", nil
	case OrNode:
		left, err := c.compile(n.Left)
		if err != nil {
			return "", err
		}
		right, err := c.compile(n.Right)
		if err != nil {
			return "", err
		}
		return "(" + left + " OR " + right + ")", nil
	case NotNode:
		expr, err := c.compile(n.Expr)
		if err != nil {
			return "", err
		}
		return "(NOT " + expr + ")", nil
	case FilterNode:
		return c.compileLeaf(n)
	default:
		return "", &CompileError{Message: fmt.Sprintf("unknown node type %T", node)}
	}
}

func (c *compiler) compileLeaf(node FilterNode) (string, error) {
	today := c.opts.Now.Format("2006-01-02")
	switch node.Operator {
	case TokenToday:
		c.args = append(c.args, today)
		return "due_date = ?", nil
	case TokenOverdue:
		c.args = append(c.args, today)
		return "due_date < ?", nil
	case TokenUpcoming:
		end := c.opts.Now.AddDate(0, 0, c.opts.UpcomingDays).Format("2006-01-02")
		c.args = append(c.args, today, end)
		return "due_date BETWEEN ? AND ?", nil
	case TokenNoDate:
		return "due_date IS NULL", nil
	case TokenProject:
		if !c.resolve(node.Value, c.opts.ResolveProject) {
			return "1=0", nil
		}
		c.args = append(c.args, strings.TrimPrefix(node.Value, "#"))
		return "project_name = ?", nil
	case TokenLabel:
		if !c.resolve(node.Value, c.opts.ResolveLabel) {
			return "1=0", nil
		}
		c.args = append(c.args, "%"+strings.TrimPrefix(node.Value, "@")+"%")
		return "label_names LIKE ?", nil
	case TokenPriority:
		c.args = append(c.args, node.Value)
		return "priority = ?", nil
	case TokenCompleted:
		if strings.EqualFold(node.Value, "true") {
			return "(status = 'completed' OR completed_at IS NOT NULL)", nil
		}
		return "(status != 'completed' AND completed_at IS NULL)", nil
	case TokenText:
		c.args = append(c.args, node.Value+"*")
		return "rowid IN (SELECT rowid FROM tasks_fts WHERE tasks_fts MATCH ?)", nil
	case TokenBefore:
		c.args = append(c.args, node.Value)
		return "due_date < ?", nil
	case TokenAfter:
		c.args = append(c.args, node.Value)
		return "due_date > ?", nil
	default:
		return "", &CompileError{Message: fmt.Sprintf("unsupported filter operator %s", node.Operator)}
	}
}

func (c *compiler) resolve(raw string, resolver func(name string) bool) bool {
	if resolver == nil {
		return true
	}
	return resolver(strings.TrimPrefix(strings.TrimPrefix(raw, "#"), "@"))
}
