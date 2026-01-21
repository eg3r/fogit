package fogit

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FeatureVersion represents a single version/iteration of a feature
// Per spec 06-data-model.md: Each version tracks its own timestamps for state derivation,
// which branch it was on, who worked on it, and optional notes
type FeatureVersion struct {
	CreatedAt  time.Time  `yaml:"created_at"`            // When this version was started
	ModifiedAt time.Time  `yaml:"modified_at,omitempty"` // When this version was last modified (for state derivation)
	ClosedAt   *time.Time `yaml:"closed_at,omitempty"`   // When this version was completed (null if open)
	Branch     string     `yaml:"branch,omitempty"`      // Git branch name (e.g., feature/login-endpoint-v2)
	Authors    []string   `yaml:"authors,omitempty"`     // All unique authors in this version
	Notes      string     `yaml:"notes,omitempty"`       // Optional description/rationale for version
}

// Feature represents a trackable item in FoGit
// Per spec 06-data-model.md (v1.0):
// - Core fields: id, name, description, tags, versions, relationships
// - User fields: metadata (organization-specific key-value pairs)
// - Computed fields (NOT stored): state, creator, contributors, current_version
type Feature struct {
	// Identity - Core fields stored in YAML
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`

	// Classification - Core field
	Tags []string `yaml:"tags,omitempty"`

	// Versioning - Core field
	// Per spec: timestamps live per-version, state is derived from current version
	// Current version = max(versions.keys())
	Versions map[string]*FeatureVersion `yaml:"versions,omitempty"`

	// Relationships - Core field (embedded with source feature)
	Relationships []Relationship `yaml:"relationships,omitempty"`

	// Files associated with feature (optional)
	Files []string `yaml:"files,omitempty"`

	// User-defined fields - All organization fields go here
	// Examples: type, priority, category, domain, team, epic, module, jira_ticket, etc.
	// FoGit preserves these but does not validate or index them
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

// State represents the feature state
// Per spec 02-concepts.md and 06-data-model.md, state is DERIVED from timestamps:
// - open: closed_at == null AND created_at == modified_at (no changes since creation)
// - in-progress: closed_at == null AND created_at < modified_at (has been modified)
// - closed: closed_at != null (feature complete, merged)
// The State field is kept for backward compatibility but should match DeriveState()
type State string

const (
	StateOpen       State = "open"
	StateInProgress State = "in-progress"
	StateClosed     State = "closed"
)

// GetCurrentVersionKey returns the current version key (highest version number)
// Per spec 06-data-model.md: current_version = max(versions.keys())
func (f *Feature) GetCurrentVersionKey() string {
	if len(f.Versions) == 0 {
		return ""
	}

	var maxVersion string
	var maxNum int = -1

	for key := range f.Versions {
		// Try to parse as integer first (simple versioning: "1", "2", "3")
		if num, err := strconv.Atoi(key); err == nil {
			if num > maxNum {
				maxNum = num
				maxVersion = key
			}
		} else if strings.Contains(key, ".") {
			// Semantic versioning: compare major.minor.patch
			var major, minor, patch int
			if _, err := fmt.Sscanf(key, "%d.%d.%d", &major, &minor, &patch); err == nil {
				// Convert to comparable integer: major*1000000 + minor*1000 + patch
				num := major*1000000 + minor*1000 + patch
				if num > maxNum {
					maxNum = num
					maxVersion = key
				}
			}
		}
	}

	return maxVersion
}

// GetCurrentVersion returns the current FeatureVersion (highest version)
// Per spec 06-data-model.md: current version = versions[max(versions.keys())]
func (f *Feature) GetCurrentVersion() *FeatureVersion {
	key := f.GetCurrentVersionKey()
	if key == "" {
		return nil
	}
	return f.Versions[key]
}

// DeriveState calculates the feature state from current version timestamps
// Per spec 06-data-model.md:
//
//	current_version = max(versions.keys())
//	version = versions[current_version]
//	if version.created_at == version.modified_at: state = "open"
//	elif version.closed_at == null: state = "in-progress"
//	else: state = "closed"
func (f *Feature) DeriveState() State {
	// Get current version
	currentVersion := f.GetCurrentVersion()

	// If no versions exist, default to open
	if currentVersion == nil {
		return StateOpen
	}

	// Per spec: derive state from current version timestamps
	if currentVersion.ClosedAt != nil {
		return StateClosed
	}
	if currentVersion.CreatedAt.Equal(currentVersion.ModifiedAt) {
		return StateOpen
	}
	return StateInProgress
}

// Priority represents feature priority
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// NewFeature creates a new feature with a name
// Per spec 06-data-model.md: Initializes with version 1, timestamps in version
func NewFeature(name string) *Feature {
	now := time.Now().UTC()
	return &Feature{
		ID:    uuid.New().String(),
		Name:  name,
		Tags:  []string{},
		Files: []string{},
		Metadata: map[string]interface{}{
			"priority": string(PriorityMedium), // Default priority per spec
		},
		Versions: map[string]*FeatureVersion{
			"1": {
				CreatedAt:  now,
				ModifiedAt: now, // Same as created = open state
				ClosedAt:   nil, // Open by default
			},
		},
	}
}

// =============================================================================
// Metadata Accessor Methods
// Per spec 06-data-model.md: organization fields are stored in metadata
// These methods provide convenient access with type safety and defaults
// =============================================================================

// GetMetadataString returns a metadata value as string, or empty string if not found
func (f *Feature) GetMetadataString(key string) string {
	if f.Metadata == nil {
		return ""
	}
	if val, ok := f.Metadata[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// SetMetadata sets a metadata value
func (f *Feature) SetMetadata(key string, value interface{}) {
	if f.Metadata == nil {
		f.Metadata = make(map[string]interface{})
	}
	f.Metadata[key] = value
}

// GetType returns the feature type from metadata
func (f *Feature) GetType() string {
	return f.GetMetadataString("type")
}

// SetType sets the feature type in metadata
func (f *Feature) SetType(t string) {
	f.SetMetadata("type", t)
}

// GetPriority returns the feature priority from metadata
func (f *Feature) GetPriority() Priority {
	if val := f.GetMetadataString("priority"); val != "" {
		return Priority(val)
	}
	return "" // Empty priority if not set
}

// SetPriority sets the feature priority in metadata
func (f *Feature) SetPriority(p Priority) {
	f.SetMetadata("priority", string(p))
}

// GetCategory returns the category from metadata
func (f *Feature) GetCategory() string {
	return f.GetMetadataString("category")
}

// SetCategory sets the category in metadata
func (f *Feature) SetCategory(c string) {
	f.SetMetadata("category", c)
}

// GetDomain returns the domain from metadata
func (f *Feature) GetDomain() string {
	return f.GetMetadataString("domain")
}

// SetDomain sets the domain in metadata
func (f *Feature) SetDomain(d string) {
	f.SetMetadata("domain", d)
}

// GetTeam returns the team from metadata
func (f *Feature) GetTeam() string {
	return f.GetMetadataString("team")
}

// SetTeam sets the team in metadata
func (f *Feature) SetTeam(t string) {
	f.SetMetadata("team", t)
}

// GetEpic returns the epic from metadata
func (f *Feature) GetEpic() string {
	return f.GetMetadataString("epic")
}

// SetEpic sets the epic in metadata
func (f *Feature) SetEpic(e string) {
	f.SetMetadata("epic", e)
}

// GetModule returns the module from metadata
func (f *Feature) GetModule() string {
	return f.GetMetadataString("module")
}

// SetModule sets the module in metadata
func (f *Feature) SetModule(m string) {
	f.SetMetadata("module", m)
}

// GetCreatedAt returns the created timestamp from current version
func (f *Feature) GetCreatedAt() time.Time {
	if cv := f.GetCurrentVersion(); cv != nil {
		return cv.CreatedAt
	}
	return time.Time{} // Zero time if no version exists
}

// GetModifiedAt returns the modified timestamp from current version
func (f *Feature) GetModifiedAt() time.Time {
	if cv := f.GetCurrentVersion(); cv != nil {
		return cv.ModifiedAt
	}
	return time.Time{} // Zero time if no version exists
}

// GetClosedAt returns the closed timestamp from current version
func (f *Feature) GetClosedAt() *time.Time {
	if cv := f.GetCurrentVersion(); cv != nil {
		return cv.ClosedAt
	}
	return nil
}

// UpdateModifiedAt updates the modified timestamp on current version
func (f *Feature) UpdateModifiedAt() {
	now := time.Now().UTC()
	if cv := f.GetCurrentVersion(); cv != nil {
		cv.ModifiedAt = now
	}
}

// Validate checks if the feature is valid
func (f *Feature) Validate() error {
	if f.Name == "" {
		return ErrEmptyName
	}

	// Validate priority from metadata
	priority := f.GetPriority()
	if priority != "" && !priority.IsValid() {
		return ErrInvalidPriority
	}

	return nil
}

// IsValid checks if the state is valid (per spec: only open, in-progress, closed)
func (s State) IsValid() bool {
	switch s {
	case StateOpen, StateInProgress, StateClosed:
		return true
	}
	return false
}

// IsValid checks if the priority is valid
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	}
	return false
}

// CanTransitionTo checks if state transition is allowed
// Per spec 02-concepts.md: open -> in-progress -> closed (and closed -> open for reopen)
func (s State) CanTransitionTo(target State) bool {
	// Define allowed transitions per spec
	transitions := map[State][]State{
		StateOpen:       {StateInProgress, StateClosed},
		StateInProgress: {StateClosed},
		StateClosed:     {StateOpen}, // Can reopen (create new version)
	}

	allowed, ok := transitions[s]
	if !ok {
		return false
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == target {
			return true
		}
	}
	return false
}

// UpdateState updates the feature state by manipulating timestamps
// Per spec 06-data-model.md, state is derived from current version timestamps:
//   - To transition to closed: set ClosedAt timestamp on current version
//   - To transition to in-progress: ensure ModifiedAt > CreatedAt on current version
//   - To transition to open: clear ClosedAt (only valid for reopening via new version)
func (f *Feature) UpdateState(newState State) error {
	if !newState.IsValid() {
		return ErrInvalidState
	}

	currentState := f.DeriveState()
	if !currentState.CanTransitionTo(newState) {
		return errors.New("invalid state transition")
	}

	now := time.Now().UTC()
	currentVersion := f.GetCurrentVersion()
	if currentVersion == nil {
		return errors.New("no version exists to update state")
	}

	switch newState {
	case StateClosed:
		// Set closed_at to mark as closed
		currentVersion.ClosedAt = &now
		currentVersion.ModifiedAt = now
	case StateInProgress:
		// Ensure modified_at > created_at to derive in-progress state
		if !currentVersion.CreatedAt.Before(currentVersion.ModifiedAt) {
			// Add 1 nanosecond to guarantee ModifiedAt > CreatedAt
			currentVersion.ModifiedAt = currentVersion.CreatedAt.Add(time.Nanosecond)
		}
		currentVersion.ClosedAt = nil
	case StateOpen:
		// This typically only happens via ReopenFeature (new version)
		currentVersion.ClosedAt = nil
	}

	return nil
}

// VersionIncrement represents how to increment a version
type VersionIncrement string

const (
	VersionIncrementPatch VersionIncrement = "patch" // 1.0.0 -> 1.0.1 (semantic only)
	VersionIncrementMinor VersionIncrement = "minor" // 1.0.0 -> 1.1.0 or 1 -> 2
	VersionIncrementMajor VersionIncrement = "major" // 1.0.0 -> 2.0.0 or 1 -> 2
)

// IncrementVersion returns the next version string based on format and increment type
// Format can be "simple" (1, 2, 3) or "semantic" (1.0.0, 1.1.0, 2.0.0)
func IncrementVersion(current string, format string, increment VersionIncrement) (string, error) {
	if format == "semantic" {
		// Parse semantic version (e.g., "1.0.0")
		var major, minor, patch int
		n, err := fmt.Sscanf(current, "%d.%d.%d", &major, &minor, &patch)
		if err != nil || n != 3 {
			// If parsing fails, assume simple format and convert
			simpleVer, err := strconv.Atoi(current)
			if err != nil {
				return "", fmt.Errorf("invalid version format: %s", current)
			}
			major = simpleVer
			minor = 0
			patch = 0
		}

		// Increment based on type
		switch increment {
		case VersionIncrementPatch:
			patch++
		case VersionIncrementMinor:
			minor++
			patch = 0
		case VersionIncrementMajor:
			major++
			minor = 0
			patch = 0
		}

		return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
	}

	// Simple format (1, 2, 3)
	simpleVer, err := strconv.Atoi(current)
	if err != nil {
		return "", fmt.Errorf("invalid simple version: %s", current)
	}

	// For simple format, both minor and major increment by 1, patch is no-op
	if increment == VersionIncrementPatch {
		// Patch doesn't apply to simple versioning, keep same
		return current, nil
	}

	return strconv.Itoa(simpleVer + 1), nil
}

// ReopenFeature reopens a closed feature with a new version
// Per spec 02-concepts.md and 06-data-model.md:
// - Closes current version (sets ClosedAt)
// - Creates new version with the provided version string
// - New version starts in OPEN state (created_at == modified_at)
// - Transitions to in-progress after first commit/modification
// Returns error if feature is not closed
func (f *Feature) ReopenFeature(currentVersionStr string, newVersionStr string, branch string, notes string) error {
	if f.DeriveState() != StateClosed {
		return errors.New("can only reopen closed features")
	}

	now := time.Now().UTC()

	// Close current version (ensure it has ClosedAt set)
	if currentVer, ok := f.Versions[currentVersionStr]; ok {
		if currentVer.ClosedAt == nil {
			currentVer.ClosedAt = &now
		}
	}

	// Create new version entry
	// Per spec: reopened features start in OPEN state (created_at == modified_at)
	// They transition to in-progress after first commit updates modified_at
	if f.Versions == nil {
		f.Versions = make(map[string]*FeatureVersion)
	}
	f.Versions[newVersionStr] = &FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now, // Same as created = open state
		ClosedAt:   nil, // Not closed
		Branch:     branch,
		Notes:      notes,
	}

	return nil
}

// AddTag adds a tag if not already present
func (f *Feature) AddTag(tag string) {
	for _, t := range f.Tags {
		if t == tag {
			return // Already exists
		}
	}
	f.Tags = append(f.Tags, tag)
	f.UpdateModifiedAt()
}

// RemoveTag removes a tag
func (f *Feature) RemoveTag(tag string) {
	for i, t := range f.Tags {
		if t == tag {
			f.Tags = append(f.Tags[:i], f.Tags[i+1:]...)
			f.UpdateModifiedAt()
			return
		}
	}
}

// AddFile associates a file with the feature
func (f *Feature) AddFile(filepath string) {
	for _, file := range f.Files {
		if file == filepath {
			return // Already exists
		}
	}
	f.Files = append(f.Files, filepath)
	f.UpdateModifiedAt()
}

// RemoveFile removes a file association
func (f *Feature) RemoveFile(filepath string) {
	for i, file := range f.Files {
		if file == filepath {
			f.Files = append(f.Files[:i], f.Files[i+1:]...)
			f.UpdateModifiedAt()
			return
		}
	}
}

// AddRelationship adds a relationship if not already present
func (f *Feature) AddRelationship(rel Relationship) error {
	// Validate the relationship
	if rel.TargetID == "" {
		return ErrEmptyTargetID
	}

	// Check for duplicates (same type and target)
	for _, existing := range f.Relationships {
		if existing.Type == rel.Type && existing.TargetID == rel.TargetID {
			return ErrDuplicateRelationship
		}
	}

	f.Relationships = append(f.Relationships, rel)
	f.UpdateModifiedAt()
	return nil
}

// RemoveRelationship removes a relationship by type and target ID
func (f *Feature) RemoveRelationship(relType RelationshipType, targetID string) error {
	for i, rel := range f.Relationships {
		if rel.Type == relType && rel.TargetID == targetID {
			f.Relationships = append(f.Relationships[:i], f.Relationships[i+1:]...)
			f.UpdateModifiedAt()
			return nil
		}
	}
	return ErrRelationshipNotFound
}

// RemoveRelationshipByID removes a relationship by its ID
func (f *Feature) RemoveRelationshipByID(relID string) error {
	for i, rel := range f.Relationships {
		if rel.ID == relID {
			f.Relationships = append(f.Relationships[:i], f.Relationships[i+1:]...)
			f.UpdateModifiedAt()
			return nil
		}
	}
	return ErrRelationshipNotFound
}

// HasRelationship checks if a relationship exists
func (f *Feature) HasRelationship(relType RelationshipType, targetID string) bool {
	for _, rel := range f.Relationships {
		if rel.Type == relType && rel.TargetID == targetID {
			return true
		}
	}
	return false
}

// GetRelationships returns relationships filtered by type (empty type returns all)
func (f *Feature) GetRelationships(relType RelationshipType) []Relationship {
	if relType == "" {
		return f.Relationships
	}

	var filtered []Relationship
	for _, rel := range f.Relationships {
		if rel.Type == relType {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}

// Note: Per spec 06-data-model.md, creator and contributors are computed from Git history.
// These are no longer stored in the feature file. Use git commands to retrieve this info.

// GetSortedVersionKeys returns version keys sorted by creation date (ascending).
// This consolidates the repeated version sorting logic across commands.
func (f *Feature) GetSortedVersionKeys() []string {
	if len(f.Versions) == 0 {
		return nil
	}

	keys := make([]string, 0, len(f.Versions))
	for k := range f.Versions {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		vi := f.Versions[keys[i]]
		vj := f.Versions[keys[j]]
		return vi.CreatedAt.Before(vj.CreatedAt)
	})

	return keys
}
