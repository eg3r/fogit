package storage

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	// separatorRegex matches spaces, dots, underscores to be replaced with hyphens
	separatorRegex = regexp.MustCompile(`[\s._]+`)
	// nonAlphanumericRegex matches anything that's not a-z, 0-9, or hyphen
	nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9-]+`)
	// multipleHyphensRegex matches multiple consecutive hyphens
	multipleHyphensRegex = regexp.MustCompile(`-+`)
)

const (
	maxSlugLength = 100
)

// SlugifyOptions configures slug generation behavior
type SlugifyOptions struct {
	MaxLength        int    // Maximum length (default: 100)
	AllowSlashes     bool   // Allow forward slashes (for branch names)
	NormalizeUnicode bool   // Remove accents/diacritics
	EmptyFallback    string // Fallback for empty slugs
}

// DefaultSlugifyOptions returns options for filename slugs
func DefaultSlugifyOptions() SlugifyOptions {
	return SlugifyOptions{
		MaxLength:        maxSlugLength,
		AllowSlashes:     false,
		NormalizeUnicode: false,
		EmptyFallback:    "",
	}
}

// Slugify converts text to a URL-safe slug with configurable options
func Slugify(text string, opts SlugifyOptions) string {
	slug := text

	// Optional: Normalize unicode (remove accents/diacritics)
	if opts.NormalizeUnicode {
		t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
		slug, _, _ = transform.String(t, slug)
	}

	// Rule 1: Convert to lowercase
	slug = strings.ToLower(slug)

	// Rule 2: Replace spaces and separators (., _) with hyphens
	slug = separatorRegex.ReplaceAllString(slug, "-")

	// Rule 3: Remove special characters (keep only a-z, 0-9, -, and optionally /)
	if opts.AllowSlashes {
		slug = regexp.MustCompile(`[^a-z0-9-/]+`).ReplaceAllString(slug, "")
	} else {
		slug = nonAlphanumericRegex.ReplaceAllString(slug, "")
	}

	// Rule 4: Collapse multiple hyphens into single hyphen
	slug = multipleHyphensRegex.ReplaceAllString(slug, "-")

	// Rule 5: Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Rule 7: Enforce maximum length
	maxLen := opts.MaxLength
	if maxLen == 0 {
		maxLen = maxSlugLength
	}
	if len(slug) > maxLen {
		slug = slug[:maxLen]
		// Trim trailing hyphens again after truncation
		slug = strings.TrimRight(slug, "-")
	}

	// Rule 6: Handle empty results with fallback
	if slug == "" && opts.EmptyFallback != "" {
		slug = opts.EmptyFallback
	}

	return slug
}

// slugify converts a feature name to a URL-safe slug following spec/guides/file-naming.md
// This is the legacy function kept for backward compatibility
func slugify(name string) string {
	return Slugify(name, DefaultSlugifyOptions())
}

// GenerateFeatureFilename generates a filename from feature name and ID, handling collisions
// This is exported so callers can know what filename will be used
func GenerateFeatureFilename(name, id string, existingFiles map[string]bool) string {
	slug := slugify(name)

	// Rule 6: If slug is empty, use feature prefix with UUID
	if slug == "" {
		slug = "feature-" + id[:8]
	}

	// Try the basic slug first
	filename := slug + ".yml"
	if !existingFiles[filename] {
		return filename
	}

	// Collision detected - append first 8 chars of UUID
	uuidPrefix := id[:8]
	filename = slug + "-" + uuidPrefix + ".yml"
	if !existingFiles[filename] {
		return filename
	}

	// Extremely rare: use full UUID
	filename = slug + "-" + id + ".yml"
	return filename
}

// generateFilename is the internal version used by repository
func generateFilename(name, id string, existingFiles map[string]bool) string {
	return GenerateFeatureFilename(name, id, existingFiles)
}
