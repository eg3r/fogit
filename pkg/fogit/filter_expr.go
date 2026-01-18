package fogit

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// FilterExpr represents a parsed filter expression.
type FilterExpr interface {
	// Matches checks if a feature matches this expression.
	Matches(f *Feature) bool
	// String returns a string representation of the expression.
	String() string
}

// Expression errors.
var (
	ErrEmptyExpression   = errors.New("empty filter expression")
	ErrInvalidExpression = errors.New("invalid filter expression")
	ErrUnmatchedParen    = errors.New("unmatched parenthesis")
	ErrInvalidOperator   = errors.New("invalid comparison operator")
	ErrInvalidField      = errors.New("invalid field name")
	ErrInvalidDateFormat = errors.New("invalid date format (expected YYYY-MM-DD)")
	ErrMissingOperand    = errors.New("missing operand for logical operator")
)

// CompareOp represents comparison operators.
type CompareOp string

const (
	OpEquals       CompareOp = "="
	OpGreater      CompareOp = ">"
	OpLess         CompareOp = "<"
	OpGreaterEqual CompareOp = ">="
	OpLessEqual    CompareOp = "<="
)

// FieldExpr represents a field comparison expression.
type FieldExpr struct {
	Field    string    // Field name (e.g., "state", "priority", "metadata.category")
	Operator CompareOp // Comparison operator
	Value    string    // Value to compare against
}

// Matches checks if the feature matches this field expression.
func (e *FieldExpr) Matches(f *Feature) bool {
	actual := e.getFieldValue(f)
	expected := e.Value

	switch e.Operator {
	case OpEquals:
		return e.matchEquals(actual, expected)
	case OpGreater, OpLess, OpGreaterEqual, OpLessEqual:
		return e.matchComparison(actual, expected, f)
	default:
		return e.matchEquals(actual, expected)
	}
}

func (e *FieldExpr) getFieldValue(f *Feature) string {
	field := strings.ToLower(e.Field)

	// Metadata shorthand aliases
	shorthandFields := map[string]bool{
		"priority": true,
		"type":     true,
		"category": true,
		"domain":   true,
		"team":     true,
		"epic":     true,
		"module":   true,
	}

	// Convert shorthand to metadata.* form
	if shorthandFields[field] {
		field = "metadata." + field
	}

	// Handle metadata.* fields
	if strings.HasPrefix(field, "metadata.") {
		key := strings.TrimPrefix(field, "metadata.")
		switch key {
		case "priority":
			return string(f.GetPriority())
		case "type":
			return f.GetType()
		case "category":
			return f.GetCategory()
		case "domain":
			return f.GetDomain()
		case "team":
			return f.GetTeam()
		case "epic":
			return f.GetEpic()
		case "module":
			return f.GetModule()
		default:
			// Check raw metadata
			if val, ok := f.Metadata[key]; ok {
				return fmt.Sprintf("%v", val)
			}
			return ""
		}
	}

	// Core fields
	switch field {
	case "state":
		return string(f.DeriveState())
	case "name":
		return f.Name
	case "description":
		return f.Description
	case "id":
		return f.ID
	case "tags":
		return strings.Join(f.Tags, ",")
	case "created":
		return f.GetCreatedAt().Format("2006-01-02")
	case "modified":
		return f.GetModifiedAt().Format("2006-01-02")
	default:
		return ""
	}
}

func (e *FieldExpr) matchEquals(actual, expected string) bool {
	// Handle special case for tags (array contains)
	if strings.ToLower(e.Field) == "tags" {
		return e.matchTags(actual, expected)
	}

	// Handle wildcards
	if strings.Contains(expected, "*") {
		return e.matchWildcard(actual, expected)
	}

	// Case-insensitive exact match
	return strings.EqualFold(actual, expected)
}

func (e *FieldExpr) matchTags(actual, expected string) bool {
	tags := strings.Split(actual, ",")
	expectedLower := strings.ToLower(expected)
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), expectedLower) {
			return true
		}
	}
	return false
}

func (e *FieldExpr) matchWildcard(actual, expected string) bool {
	// Convert wildcard pattern to regex
	pattern := "^" + regexp.QuoteMeta(expected) + "$"
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return false
	}
	return re.MatchString(actual)
}

func (e *FieldExpr) matchComparison(actual, expected string, f *Feature) bool {
	field := strings.ToLower(e.Field)

	// Date comparisons
	if field == "created" || field == "modified" {
		return e.matchDateComparison(actual, expected, f)
	}

	// Priority comparisons
	if field == "priority" || field == "metadata.priority" {
		return e.matchPriorityComparison(actual, expected)
	}

	// String comparison (lexicographic)
	actualLower := strings.ToLower(actual)
	expectedLower := strings.ToLower(expected)

	switch e.Operator {
	case OpGreater:
		return actualLower > expectedLower
	case OpLess:
		return actualLower < expectedLower
	case OpGreaterEqual:
		return actualLower >= expectedLower
	case OpLessEqual:
		return actualLower <= expectedLower
	default:
		return false
	}
}

func (e *FieldExpr) matchDateComparison(_, expected string, f *Feature) bool {
	var actualTime time.Time
	if strings.ToLower(e.Field) == "created" {
		actualTime = f.GetCreatedAt()
	} else {
		actualTime = f.GetModifiedAt()
	}

	expectedTime, err := time.Parse("2006-01-02", expected)
	if err != nil {
		return false
	}

	// Normalize to start of day for comparison
	actualDate := time.Date(actualTime.Year(), actualTime.Month(), actualTime.Day(), 0, 0, 0, 0, time.UTC)
	expectedDate := time.Date(expectedTime.Year(), expectedTime.Month(), expectedTime.Day(), 0, 0, 0, 0, time.UTC)

	switch e.Operator {
	case OpGreater:
		return actualDate.After(expectedDate)
	case OpLess:
		return actualDate.Before(expectedDate)
	case OpGreaterEqual:
		return !actualDate.Before(expectedDate)
	case OpLessEqual:
		return !actualDate.After(expectedDate)
	default:
		return false
	}
}

func (e *FieldExpr) matchPriorityComparison(actual, expected string) bool {
	priorityOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	actualOrder, aOk := priorityOrder[strings.ToLower(actual)]
	expectedOrder, eOk := priorityOrder[strings.ToLower(expected)]

	if !aOk || !eOk {
		return false
	}

	switch e.Operator {
	case OpGreater:
		return actualOrder > expectedOrder
	case OpLess:
		return actualOrder < expectedOrder
	case OpGreaterEqual:
		return actualOrder >= expectedOrder
	case OpLessEqual:
		return actualOrder <= expectedOrder
	default:
		return false
	}
}

func (e *FieldExpr) String() string {
	op := ":"
	switch e.Operator {
	case OpGreater:
		op = ":>"
	case OpLess:
		op = ":<"
	case OpGreaterEqual:
		op = ":>="
	case OpLessEqual:
		op = ":<="
	}
	return fmt.Sprintf("%s%s%s", e.Field, op, e.Value)
}

// AndExpr represents a logical AND expression.
type AndExpr struct {
	Left  FilterExpr
	Right FilterExpr
}

func (e *AndExpr) Matches(f *Feature) bool {
	return e.Left.Matches(f) && e.Right.Matches(f)
}

func (e *AndExpr) String() string {
	return fmt.Sprintf("(%s AND %s)", e.Left.String(), e.Right.String())
}

// OrExpr represents a logical OR expression.
type OrExpr struct {
	Left  FilterExpr
	Right FilterExpr
}

func (e *OrExpr) Matches(f *Feature) bool {
	return e.Left.Matches(f) || e.Right.Matches(f)
}

func (e *OrExpr) String() string {
	return fmt.Sprintf("(%s OR %s)", e.Left.String(), e.Right.String())
}

// NotExpr represents a logical NOT expression.
type NotExpr struct {
	Expr FilterExpr
}

func (e *NotExpr) Matches(f *Feature) bool {
	return !e.Expr.Matches(f)
}

func (e *NotExpr) String() string {
	return fmt.Sprintf("NOT %s", e.Expr.String())
}

// TrueExpr always matches (used for empty expressions).
type TrueExpr struct{}

func (e *TrueExpr) Matches(f *Feature) bool {
	return true
}

func (e *TrueExpr) String() string {
	return "true"
}

// ParseFilterExpr parses a filter expression string into a FilterExpr.
func ParseFilterExpr(expr string) (FilterExpr, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return &TrueExpr{}, nil
	}

	parser := &exprParser{input: expr, pos: 0}
	result, err := parser.parseOr()
	if err != nil {
		return nil, err
	}

	// Ensure all input was consumed
	parser.skipWhitespace()
	if parser.pos < len(parser.input) {
		return nil, fmt.Errorf("%w: unexpected token at position %d", ErrInvalidExpression, parser.pos)
	}

	return result, nil
}

// exprParser is a recursive descent parser for filter expressions.
type exprParser struct {
	input string
	pos   int
}

func (p *exprParser) parseOr() (FilterExpr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.matchKeyword("OR") {
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &OrExpr{Left: left, Right: right}
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseAnd() (FilterExpr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.matchKeyword("AND") {
			right, err := p.parseNot()
			if err != nil {
				return nil, err
			}
			left = &AndExpr{Left: left, Right: right}
		} else {
			break
		}
	}

	return left, nil
}

func (p *exprParser) parseNot() (FilterExpr, error) {
	p.skipWhitespace()
	if p.matchKeyword("NOT") {
		expr, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &NotExpr{Expr: expr}, nil
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (FilterExpr, error) {
	p.skipWhitespace()

	// Handle parentheses
	if p.pos < len(p.input) && p.input[p.pos] == '(' {
		p.pos++ // consume '('
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return nil, ErrUnmatchedParen
		}
		p.pos++ // consume ')'
		return expr, nil
	}

	// Parse field expression
	return p.parseFieldExpr()
}

func (p *exprParser) parseFieldExpr() (FilterExpr, error) {
	p.skipWhitespace()

	// Parse field name
	field := p.parseField()
	if field == "" {
		return nil, fmt.Errorf("%w: expected field name at position %d", ErrInvalidExpression, p.pos)
	}

	// Parse operator
	op, err := p.parseOperator()
	if err != nil {
		return nil, err
	}

	// Parse value
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	return &FieldExpr{
		Field:    field,
		Operator: op,
		Value:    value,
	}, nil
}

func (p *exprParser) parseField() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == ':' || ch == ' ' || ch == '\t' || ch == '(' || ch == ')' {
			break
		}
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *exprParser) parseOperator() (CompareOp, error) {
	if p.pos >= len(p.input) || p.input[p.pos] != ':' {
		return "", fmt.Errorf("%w: expected ':' at position %d", ErrInvalidExpression, p.pos)
	}
	p.pos++ // consume ':'

	// Check for comparison operators
	if p.pos < len(p.input) {
		switch {
		case strings.HasPrefix(p.input[p.pos:], ">="):
			p.pos += 2
			return OpGreaterEqual, nil
		case strings.HasPrefix(p.input[p.pos:], "<="):
			p.pos += 2
			return OpLessEqual, nil
		case p.input[p.pos] == '>':
			p.pos++
			return OpGreater, nil
		case p.input[p.pos] == '<':
			p.pos++
			return OpLess, nil
		}
	}

	return OpEquals, nil
}

func (p *exprParser) parseValue() (string, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return "", fmt.Errorf("%w: expected value at position %d", ErrInvalidExpression, p.pos)
	}

	// Handle quoted values
	if p.input[p.pos] == '"' {
		return p.parseQuotedValue()
	}

	// Parse unquoted value
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		// Stop at whitespace or logical operators
		if ch == ' ' || ch == '\t' || ch == '(' || ch == ')' {
			break
		}
		// Check for logical operators
		remaining := p.input[p.pos:]
		if strings.HasPrefix(strings.ToUpper(remaining), " AND ") ||
			strings.HasPrefix(strings.ToUpper(remaining), " OR ") {
			break
		}
		p.pos++
	}

	if p.pos == start {
		return "", fmt.Errorf("%w: expected value at position %d", ErrInvalidExpression, p.pos)
	}

	return p.input[start:p.pos], nil
}

func (p *exprParser) parseQuotedValue() (string, error) {
	p.pos++ // consume opening quote
	var result strings.Builder
	escaped := false

	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if escaped {
			result.WriteByte(ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else if ch == '"' {
			p.pos++ // consume closing quote
			return result.String(), nil
		} else {
			result.WriteByte(ch)
		}
		p.pos++
	}

	return "", fmt.Errorf("%w: unterminated quoted value", ErrInvalidExpression)
}

func (p *exprParser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}

func (p *exprParser) matchKeyword(keyword string) bool {
	if p.pos+len(keyword) > len(p.input) {
		return false
	}

	// Check if keyword matches (case-insensitive)
	substr := p.input[p.pos : p.pos+len(keyword)]
	if !strings.EqualFold(substr, keyword) {
		return false
	}

	// Ensure it's followed by whitespace or end of input (not part of field name)
	endPos := p.pos + len(keyword)
	if endPos < len(p.input) {
		next := p.input[endPos]
		if next != ' ' && next != '\t' && next != '(' {
			return false
		}
	}

	p.pos += len(keyword)
	return true
}
