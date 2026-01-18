package interactive

import (
	"github.com/eg3r/fogit/internal/search"
	"github.com/eg3r/fogit/pkg/fogit"
)

// MockPrompter is a mock implementation of Prompter for testing
type MockPrompter struct {
	// Function stubs for customizing behavior
	SelectFromSimilarFeaturesFunc func(query string, matches []search.Match) (*fogit.Feature, bool, error)
	SelectVersionIncrementFunc    func(currentVersion, versionFormat string) (fogit.VersionIncrement, error)
	ConfirmFunc                   func(message string) (bool, error)
	ConfirmDeletionFunc           func(feature *fogit.Feature, incomingRels []IncomingRelationship) (bool, error)
	ReadLineFunc                  func(prompt string) (string, error)
	SelectFeatureFunc             func(features []*fogit.Feature, title string) (*fogit.Feature, error)
	InputTextFunc                 func(title, defaultValue string) (string, error)

	// Call tracking
	Calls []MockCall
}

// MockCall records a method call for verification
type MockCall struct {
	Method string
	Args   []interface{}
}

// NewMockPrompter creates a new mock prompter with default behaviors
func NewMockPrompter() *MockPrompter {
	return &MockPrompter{
		Calls: make([]MockCall, 0),
	}
}

// SelectFromSimilarFeatures implements Prompter
func (m *MockPrompter) SelectFromSimilarFeatures(query string, matches []search.Match) (*fogit.Feature, bool, error) {
	m.Calls = append(m.Calls, MockCall{
		Method: "SelectFromSimilarFeatures",
		Args:   []interface{}{query, matches},
	})

	if m.SelectFromSimilarFeaturesFunc != nil {
		return m.SelectFromSimilarFeaturesFunc(query, matches)
	}

	// Default: select first match if available, otherwise create new
	if len(matches) > 0 {
		return matches[0].Feature, false, nil
	}
	return nil, true, nil
}

// SelectVersionIncrement implements Prompter
func (m *MockPrompter) SelectVersionIncrement(currentVersion, versionFormat string) (fogit.VersionIncrement, error) {
	m.Calls = append(m.Calls, MockCall{
		Method: "SelectVersionIncrement",
		Args:   []interface{}{currentVersion, versionFormat},
	})

	if m.SelectVersionIncrementFunc != nil {
		return m.SelectVersionIncrementFunc(currentVersion, versionFormat)
	}

	// Default: minor increment
	return fogit.VersionIncrementMinor, nil
}

// Confirm implements Prompter
func (m *MockPrompter) Confirm(message string) (bool, error) {
	m.Calls = append(m.Calls, MockCall{
		Method: "Confirm",
		Args:   []interface{}{message},
	})

	if m.ConfirmFunc != nil {
		return m.ConfirmFunc(message)
	}

	// Default: confirm
	return true, nil
}

// ConfirmDeletion implements Prompter
func (m *MockPrompter) ConfirmDeletion(feature *fogit.Feature, incomingRels []IncomingRelationship) (bool, error) {
	m.Calls = append(m.Calls, MockCall{
		Method: "ConfirmDeletion",
		Args:   []interface{}{feature, incomingRels},
	})

	if m.ConfirmDeletionFunc != nil {
		return m.ConfirmDeletionFunc(feature, incomingRels)
	}

	// Default: confirm
	return true, nil
}

// ReadLine implements Prompter
func (m *MockPrompter) ReadLine(prompt string) (string, error) {
	m.Calls = append(m.Calls, MockCall{
		Method: "ReadLine",
		Args:   []interface{}{prompt},
	})

	if m.ReadLineFunc != nil {
		return m.ReadLineFunc(prompt)
	}

	// Default: empty string
	return "", nil
}

// SelectFeature implements Prompter
func (m *MockPrompter) SelectFeature(features []*fogit.Feature, title string) (*fogit.Feature, error) {
	m.Calls = append(m.Calls, MockCall{
		Method: "SelectFeature",
		Args:   []interface{}{features, title},
	})

	if m.SelectFeatureFunc != nil {
		return m.SelectFeatureFunc(features, title)
	}

	// Default: select first
	if len(features) > 0 {
		return features[0], nil
	}
	return nil, nil
}

// InputText implements Prompter
func (m *MockPrompter) InputText(title, defaultValue string) (string, error) {
	m.Calls = append(m.Calls, MockCall{
		Method: "InputText",
		Args:   []interface{}{title, defaultValue},
	})

	if m.InputTextFunc != nil {
		return m.InputTextFunc(title, defaultValue)
	}

	// Default: return default value
	return defaultValue, nil
}

// Reset clears all recorded calls
func (m *MockPrompter) Reset() {
	m.Calls = make([]MockCall, 0)
}

// GetCallCount returns the number of times a method was called
func (m *MockPrompter) GetCallCount(method string) int {
	count := 0
	for _, call := range m.Calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// WasCalled returns true if the method was called at least once
func (m *MockPrompter) WasCalled(method string) bool {
	return m.GetCallCount(method) > 0
}
