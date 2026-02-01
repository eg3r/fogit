// Package features provides business logic for FoGit feature management.
package features

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// RelationshipTypeUpdateOptions contains options for updating a relationship type.
type RelationshipTypeUpdateOptions struct {
	NewName        string   // Rename the type (empty = no rename)
	RenameInverse  string   // New name for inverse type (only when renaming)
	KeepOldAsAlias bool     // Add old name as alias when renaming
	Category       string   // New category (empty = no change)
	Inverse        string   // New inverse type (empty = no change)
	Description    string   // New description (empty = no change)
	SetDescription bool     // Whether to set description (allows empty string)
	Bidirectional  *bool    // Set bidirectional (nil = no change)
	AddAliases     []string // Aliases to add
	RemoveAliases  []string // Aliases to remove
}

// RelationshipTypeUpdateResult contains the result of a type update operation.
type RelationshipTypeUpdateResult struct {
	Renamed         bool
	OldName         string
	NewName         string
	OldInverse      string
	NewInverse      string
	UpdatedRelCount int
	KeptOldAsAlias  bool
	InverseRenamed  bool
}

// UpdateRelationshipType updates a relationship type's configuration and optionally renames it.
// Returns the result of the operation or an error.
func UpdateRelationshipType(fogitDir, typeName string, opts RelationshipTypeUpdateOptions) (*RelationshipTypeUpdateResult, error) {
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	typeConfig, exists := cfg.Relationships.Types[typeName]
	if !exists {
		return nil, fmt.Errorf("relationship type '%s' not found", typeName)
	}

	result := &RelationshipTypeUpdateResult{
		OldName:    typeName,
		OldInverse: typeConfig.Inverse,
	}

	isRenaming := opts.NewName != "" && opts.NewName != typeName

	// Validate rename target doesn't exist
	if isRenaming {
		if _, exists := cfg.Relationships.Types[opts.NewName]; exists {
			return nil, fmt.Errorf("type '%s' already exists", opts.NewName)
		}
	}

	// Validate category if changing
	if opts.Category != "" {
		if _, exists := cfg.Relationships.Categories[opts.Category]; !exists {
			available := make([]string, 0, len(cfg.Relationships.Categories))
			for cat := range cfg.Relationships.Categories {
				available = append(available, cat)
			}
			return nil, fmt.Errorf("category '%s' not found. Available: %s", opts.Category, strings.Join(available, ", "))
		}
		typeConfig.Category = opts.Category
	}

	// Handle bidirectional setting
	if opts.Bidirectional != nil {
		if *opts.Bidirectional {
			typeConfig.Bidirectional = true
			typeConfig.Inverse = "" // Bidirectional types don't have inverse
		} else {
			typeConfig.Bidirectional = false
		}
	}

	// Update inverse (only if not bidirectional)
	if opts.Inverse != "" && !typeConfig.Bidirectional {
		typeConfig.Inverse = opts.Inverse
	}

	// Update description
	if opts.SetDescription {
		typeConfig.Description = opts.Description
	}

	// Manage aliases - build set for deduplication
	aliasSet := make(map[string]bool)
	for _, a := range typeConfig.Aliases {
		aliasSet[a] = true
	}
	for _, a := range opts.AddAliases {
		aliasSet[a] = true
	}
	for _, a := range opts.RemoveAliases {
		delete(aliasSet, a)
	}

	// Handle renaming
	if isRenaming {
		result.Renamed = true
		result.NewName = opts.NewName

		// Keep old name as alias if requested (add to set before building slice for deduplication)
		if opts.KeepOldAsAlias {
			aliasSet[typeName] = true
			result.KeptOldAsAlias = true
		}
	} else {
		result.NewName = typeName
	}

	// Build final aliases slice from set
	typeConfig.Aliases = make([]string, 0, len(aliasSet))
	for a := range aliasSet {
		typeConfig.Aliases = append(typeConfig.Aliases, a)
	}
	sort.Strings(typeConfig.Aliases) // Ensure deterministic ordering

	if isRenaming {
		// Delete old type, add new one
		delete(cfg.Relationships.Types, typeName)
		cfg.Relationships.Types[opts.NewName] = typeConfig

		// Handle inverse type renaming
		if result.OldInverse != "" && !typeConfig.Bidirectional {
			result.NewInverse, result.InverseRenamed = handleInverseTypeRename(
				cfg, result.OldInverse, opts.NewName, opts.RenameInverse, typeConfig)
		}

		// Update all feature files
		// Only pass inverse type info if it was actually renamed to avoid unnecessary updates.
		// Note: InverseRenamed is true iff the inverse type name changed (see handleInverseTypeRename),
		// so when false, oldInverse == newInverse and no relationship updates are needed.
		oldInv, newInv := "", ""
		if result.InverseRenamed {
			oldInv, newInv = result.OldInverse, result.NewInverse
		}
		updatedCount, err := UpdateRelationshipsInFeatures(fogitDir, typeName, opts.NewName, oldInv, newInv)
		if err != nil {
			return nil, fmt.Errorf("failed to update relationships: %w", err)
		}
		result.UpdatedRelCount = updatedCount
	} else {
		// Just update the type config
		cfg.Relationships.Types[typeName] = typeConfig
	}

	// Validate and save
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	if err := config.Save(fogitDir, cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return result, nil
}

// RelationshipTypeDeleteOptions contains options for deleting a relationship type.
type RelationshipTypeDeleteOptions struct {
	MigrateTo string // Migrate relationships to this type (empty = no migration)
	Cascade   bool   // Delete all relationships using this type
	Force     bool   // Delete type only, leave relationships orphaned
}

// handleInverseTypeRename handles the renaming of an inverse type when its paired type is renamed.
// It updates the inverse type's config to point to the new name and optionally renames the inverse itself.
func handleInverseTypeRename(cfg *fogit.Config, oldInverse, newTypeName, renameInverseTo string, typeConfig fogit.RelationshipTypeConfig) (newInverse string, renamed bool) {
	// Determine new inverse name
	newInverseName := renameInverseTo
	if newInverseName == "" {
		newInverseName = oldInverse // Keep same inverse name by default
	}

	inverseConfig, exists := cfg.Relationships.Types[oldInverse]
	if !exists {
		return oldInverse, false
	}

	// Update inverse to point to new primary type name
	inverseConfig.Inverse = newTypeName

	if newInverseName != oldInverse {
		// Rename the inverse type
		delete(cfg.Relationships.Types, oldInverse)
		cfg.Relationships.Types[newInverseName] = inverseConfig
		typeConfig.Inverse = newInverseName
		cfg.Relationships.Types[newTypeName] = typeConfig
		return newInverseName, true
	}

	// Just update the inverse config without renaming
	cfg.Relationships.Types[oldInverse] = inverseConfig
	return oldInverse, false
}

// RelationshipTypeDeleteResult contains the result of a type delete operation.
type RelationshipTypeDeleteResult struct {
	TypeName        string
	InverseType     string
	AffectedRels    []string // Human-readable list of affected relationships
	MigratedCount   int
	DeletedRelCount int
	RequiresConfirm bool
	ConfirmMessage  string
}

// DeleteRelationshipType deletes a relationship type from the configuration.
// If relationships exist and no handling option is specified, returns an error with details.
func DeleteRelationshipType(fogitDir, typeName string, opts RelationshipTypeDeleteOptions) (*RelationshipTypeDeleteResult, error) {
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	typeConfig, exists := cfg.Relationships.Types[typeName]
	if !exists {
		return nil, fmt.Errorf("relationship type '%s' not found", typeName)
	}

	result := &RelationshipTypeDeleteResult{
		TypeName:    typeName,
		InverseType: typeConfig.Inverse,
	}

	// Count relationships using this type
	repo := storage.NewFileRepository(fogitDir)
	features, err := repo.List(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	for _, f := range features {
		for _, rel := range f.Relationships {
			if string(rel.Type) == typeName || (result.InverseType != "" && string(rel.Type) == result.InverseType) {
				result.AffectedRels = append(result.AffectedRels, fmt.Sprintf("%s → %s (%s)", f.Name, rel.TargetName, rel.Type))
			}
		}
	}

	relCount := len(result.AffectedRels)

	// Validate flag combinations
	if opts.MigrateTo != "" && opts.Cascade {
		return nil, fmt.Errorf("cannot use both migrate-to and cascade")
	}
	if opts.MigrateTo != "" && opts.Force {
		return nil, fmt.Errorf("cannot use both migrate-to and force")
	}
	if opts.Cascade && opts.Force {
		return nil, fmt.Errorf("cannot use both cascade and force")
	}

	// Check if relationships exist and no handling specified
	if relCount > 0 && opts.MigrateTo == "" && !opts.Cascade && !opts.Force {
		return nil, &RelationshipTypeInUseError{
			TypeName:     typeName,
			AffectedRels: result.AffectedRels,
		}
	}

	// Handle --migrate-to
	if opts.MigrateTo != "" {
		targetConfig, exists := cfg.Relationships.Types[opts.MigrateTo]
		if !exists {
			return nil, fmt.Errorf("target type '%s' not found", opts.MigrateTo)
		}

		migratedCount, err := UpdateRelationshipsInFeatures(fogitDir, typeName, opts.MigrateTo, result.InverseType, targetConfig.Inverse)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate relationships: %w", err)
		}
		result.MigratedCount = migratedCount

		delete(cfg.Relationships.Types, typeName)
		if result.InverseType != "" {
			delete(cfg.Relationships.Types, result.InverseType)
		}

		if err := config.Save(fogitDir, cfg); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		return result, nil
	}

	// Handle --cascade
	// When relationships exist, require confirmation; actual deletion happens in ExecuteRelationshipTypeDelete
	if opts.Cascade && relCount > 0 {
		result.RequiresConfirm = true
		result.ConfirmMessage = fmt.Sprintf("This will permanently delete %d relationships", relCount)
		return result, nil
	}

	// Handle --force
	if opts.Force && relCount > 0 {
		result.RequiresConfirm = true
		result.ConfirmMessage = fmt.Sprintf("%d relationships use type '%s' and will become invalid", relCount, typeName)
		return result, nil
	}

	// Delete type (and inverse if exists)
	delete(cfg.Relationships.Types, typeName)
	if result.InverseType != "" {
		delete(cfg.Relationships.Types, result.InverseType)
	}

	if err := config.Save(fogitDir, cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return result, nil
}

// ExecuteRelationshipTypeDelete performs the actual deletion after confirmation.
// Call this after DeleteRelationshipType returns RequiresConfirm=true and user confirms.
func ExecuteRelationshipTypeDelete(fogitDir, typeName string, opts RelationshipTypeDeleteOptions) (*RelationshipTypeDeleteResult, error) {
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	typeConfig, exists := cfg.Relationships.Types[typeName]
	if !exists {
		return nil, fmt.Errorf("relationship type '%s' not found", typeName)
	}

	result := &RelationshipTypeDeleteResult{
		TypeName:    typeName,
		InverseType: typeConfig.Inverse,
	}

	if opts.Cascade {
		deletedCount, err := DeleteRelationshipsByType(fogitDir, typeName, result.InverseType)
		if err != nil {
			return nil, fmt.Errorf("failed to delete relationships: %w", err)
		}
		result.DeletedRelCount = deletedCount
	}

	delete(cfg.Relationships.Types, typeName)
	if result.InverseType != "" {
		delete(cfg.Relationships.Types, result.InverseType)
	}

	if err := config.Save(fogitDir, cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return result, nil
}

// RelationshipTypeInUseError is returned when attempting to delete a type that has relationships.
type RelationshipTypeInUseError struct {
	TypeName     string
	AffectedRels []string
}

func (e *RelationshipTypeInUseError) Error() string {
	return fmt.Sprintf("relationships exist using type '%s'", e.TypeName)
}

// UpdateRelationshipsInFeatures updates relationship type names in all feature files.
func UpdateRelationshipsInFeatures(fogitDir, oldType, newType, oldInverse, newInverse string) (int, error) {
	repo := storage.NewFileRepository(fogitDir)
	features, err := repo.List(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	updatedCount := 0
	for _, feature := range features {
		modified := false
		for i, rel := range feature.Relationships {
			if string(rel.Type) == oldType {
				feature.Relationships[i].Type = fogit.RelationshipType(newType)
				modified = true
				updatedCount++
			}
			// Only update inverse if it actually changed
			if oldInverse != "" && newInverse != "" && oldInverse != newInverse && string(rel.Type) == oldInverse {
				feature.Relationships[i].Type = fogit.RelationshipType(newInverse)
				modified = true
				updatedCount++
			}
		}
		if modified {
			if err := repo.Update(context.Background(), feature); err != nil {
				return updatedCount, fmt.Errorf("failed to update feature '%s': %w", feature.Name, err)
			}
		}
	}

	return updatedCount, nil
}

// DeleteRelationshipsByType removes all relationships of a given type from features.
func DeleteRelationshipsByType(fogitDir, typeName, inverseType string) (int, error) {
	repo := storage.NewFileRepository(fogitDir)
	features, err := repo.List(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	deletedCount := 0
	for _, feature := range features {
		modified := false
		newRels := make([]fogit.Relationship, 0, len(feature.Relationships))
		for _, rel := range feature.Relationships {
			if string(rel.Type) == typeName || (inverseType != "" && string(rel.Type) == inverseType) {
				deletedCount++
				modified = true
			} else {
				newRels = append(newRels, rel)
			}
		}
		if modified {
			feature.Relationships = newRels
			if err := repo.Update(context.Background(), feature); err != nil {
				return deletedCount, fmt.Errorf("failed to update feature '%s': %w", feature.Name, err)
			}
		}
	}

	return deletedCount, nil
}

// RelationshipCategoryUpdateOptions contains options for updating a relationship category.
type RelationshipCategoryUpdateOptions struct {
	NewName         string // Rename the category (empty = no rename)
	KeepOldAsAlias  bool   // Add old name as alias when renaming
	Description     string // New description
	SetDescription  bool   // Whether to set description
	AllowCycles     *bool  // Set allow cycles (nil = no change)
	CycleDetection  string // Cycle detection mode (empty = no change)
	IncludeInImpact *bool  // Include in impact analysis (nil = no change)
}

// RelationshipCategoryUpdateResult contains the result of a category update operation.
type RelationshipCategoryUpdateResult struct {
	Renamed        bool
	OldName        string
	NewName        string
	TypesUpdated   int
	KeptOldAsAlias bool
}

// UpdateRelationshipCategory updates a relationship category's configuration and optionally renames it.
func UpdateRelationshipCategory(fogitDir, categoryName string, opts RelationshipCategoryUpdateOptions) (*RelationshipCategoryUpdateResult, error) {
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	categoryConfig, exists := cfg.Relationships.Categories[categoryName]
	if !exists {
		return nil, fmt.Errorf("relationship category '%s' not found", categoryName)
	}

	result := &RelationshipCategoryUpdateResult{
		OldName: categoryName,
		NewName: categoryName,
	}

	// Validate detection mode if provided
	if opts.CycleDetection != "" {
		if !fogit.IsValidCycleDetectionMode(opts.CycleDetection) {
			return nil, fmt.Errorf("invalid detection mode '%s', must be one of: %s", opts.CycleDetection, strings.Join(fogit.ValidCycleDetectionModes, ", "))
		}
	}

	// Handle rename
	if opts.NewName != "" && opts.NewName != categoryName {
		if _, exists := cfg.Relationships.Categories[opts.NewName]; exists {
			return nil, fmt.Errorf("category '%s' already exists", opts.NewName)
		}
		result.Renamed = true
		result.NewName = opts.NewName
	}

	// Update properties
	if opts.SetDescription {
		categoryConfig.Description = opts.Description
	}
	if opts.AllowCycles != nil {
		categoryConfig.AllowCycles = *opts.AllowCycles
	}
	if opts.CycleDetection != "" {
		categoryConfig.CycleDetection = opts.CycleDetection
	}
	if opts.IncludeInImpact != nil {
		categoryConfig.IncludeInImpact = *opts.IncludeInImpact
	}

	// Handle rename
	if result.Renamed {
		// Update all types that reference this category
		for typeName, typeConfig := range cfg.Relationships.Types {
			if typeConfig.Category == categoryName {
				updatedType := cfg.Relationships.Types[typeName]
				updatedType.Category = result.NewName
				cfg.Relationships.Types[typeName] = updatedType
				result.TypesUpdated++
			}
		}

		// Delete old entry and add new one
		delete(cfg.Relationships.Categories, categoryName)

		// Handle keep-old-as-alias
		if opts.KeepOldAsAlias {
			if categoryConfig.Metadata == nil {
				categoryConfig.Metadata = make(map[string]interface{})
			}
			aliases, _ := categoryConfig.Metadata["aliases"].([]string)
			aliases = append(aliases, categoryName)
			categoryConfig.Metadata["aliases"] = aliases
			result.KeptOldAsAlias = true
		}

		cfg.Relationships.Categories[result.NewName] = categoryConfig
	} else {
		cfg.Relationships.Categories[categoryName] = categoryConfig
	}

	if err := config.Save(fogitDir, cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return result, nil
}

// RelationshipCategoryDeleteOptions contains options for deleting a relationship category.
type RelationshipCategoryDeleteOptions struct {
	MoveTypesTo string // Move types to this category (empty = no move)
	Cascade     bool   // Delete category and all its types
	Force       bool   // Delete category only, types become uncategorized
}

// RelationshipCategoryDeleteResult contains the result of a category delete operation.
type RelationshipCategoryDeleteResult struct {
	CategoryName    string
	AffectedTypes   []string
	MovedTypesCount int
	DeletedTypes    int
	DeletedRelCount int
	RequiresConfirm bool
	ConfirmMessage  string
}

// RelationshipCategoryInUseError is returned when attempting to delete a category that has types.
type RelationshipCategoryInUseError struct {
	CategoryName  string
	AffectedTypes []string
}

func (e *RelationshipCategoryInUseError) Error() string {
	return fmt.Sprintf("types exist in category '%s'", e.CategoryName)
}

// DeleteRelationshipCategory deletes a relationship category from the configuration.
func DeleteRelationshipCategory(fogitDir, categoryName string, opts RelationshipCategoryDeleteOptions) (*RelationshipCategoryDeleteResult, error) {
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if _, exists := cfg.Relationships.Categories[categoryName]; !exists {
		return nil, fmt.Errorf("relationship category '%s' not found", categoryName)
	}

	result := &RelationshipCategoryDeleteResult{
		CategoryName: categoryName,
	}

	// Find types using this category
	for typeName, typeConfig := range cfg.Relationships.Types {
		if typeConfig.Category == categoryName {
			result.AffectedTypes = append(result.AffectedTypes, typeName)
		}
	}

	typeCount := len(result.AffectedTypes)

	// Validate flag combinations
	if opts.MoveTypesTo != "" && opts.Cascade {
		return nil, fmt.Errorf("cannot use both move-types-to and cascade")
	}
	if opts.MoveTypesTo != "" && opts.Force {
		return nil, fmt.Errorf("cannot use both move-types-to and force")
	}
	if opts.Cascade && opts.Force {
		return nil, fmt.Errorf("cannot use both cascade and force")
	}

	// Check if types exist and no handling specified
	if typeCount > 0 && opts.MoveTypesTo == "" && !opts.Cascade && !opts.Force {
		return nil, &RelationshipCategoryInUseError{
			CategoryName:  categoryName,
			AffectedTypes: result.AffectedTypes,
		}
	}

	// Handle move-types-to
	if opts.MoveTypesTo != "" {
		if _, exists := cfg.Relationships.Categories[opts.MoveTypesTo]; !exists {
			return nil, fmt.Errorf("target category '%s' not found", opts.MoveTypesTo)
		}

		for _, typeName := range result.AffectedTypes {
			typeConfig := cfg.Relationships.Types[typeName]
			typeConfig.Category = opts.MoveTypesTo
			cfg.Relationships.Types[typeName] = typeConfig
			result.MovedTypesCount++
		}

		delete(cfg.Relationships.Categories, categoryName)

		if err := config.Save(fogitDir, cfg); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		return result, nil
	}

	// Handle cascade (requires confirmation)
	if opts.Cascade && typeCount > 0 {
		result.RequiresConfirm = true
		result.ConfirmMessage = fmt.Sprintf("This will delete %d relationship types and all their relationships", typeCount)
		return result, nil
	}

	// Handle force (requires confirmation if types exist)
	if opts.Force && typeCount > 0 {
		result.RequiresConfirm = true
		result.ConfirmMessage = fmt.Sprintf("%d types will become uncategorized", typeCount)
		return result, nil
	}

	// Delete category
	delete(cfg.Relationships.Categories, categoryName)

	if err := config.Save(fogitDir, cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return result, nil
}

// ExecuteRelationshipCategoryDelete performs the actual deletion after confirmation.
func ExecuteRelationshipCategoryDelete(fogitDir, categoryName string, opts RelationshipCategoryDeleteOptions) (*RelationshipCategoryDeleteResult, error) {
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if _, exists := cfg.Relationships.Categories[categoryName]; !exists {
		return nil, fmt.Errorf("relationship category '%s' not found", categoryName)
	}

	result := &RelationshipCategoryDeleteResult{
		CategoryName: categoryName,
	}

	// Find affected types
	for typeName, typeConfig := range cfg.Relationships.Types {
		if typeConfig.Category == categoryName {
			result.AffectedTypes = append(result.AffectedTypes, typeName)
		}
	}

	if opts.Cascade {
		// Delete all types and their relationships
		for _, typeName := range result.AffectedTypes {
			typeConfig := cfg.Relationships.Types[typeName]
			inverseType := typeConfig.Inverse

			count, err := DeleteRelationshipsByType(fogitDir, typeName, inverseType)
			if err != nil {
				return nil, fmt.Errorf("failed to delete relationships for type '%s': %w", typeName, err)
			}
			result.DeletedRelCount += count

			delete(cfg.Relationships.Types, typeName)
			if inverseType != "" {
				delete(cfg.Relationships.Types, inverseType)
			}
			result.DeletedTypes++
		}
	}

	if opts.Force {
		// Clear category from affected types
		for _, typeName := range result.AffectedTypes {
			typeConfig := cfg.Relationships.Types[typeName]
			typeConfig.Category = ""
			cfg.Relationships.Types[typeName] = typeConfig
		}
	}

	delete(cfg.Relationships.Categories, categoryName)

	if err := config.Save(fogitDir, cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return result, nil
}
