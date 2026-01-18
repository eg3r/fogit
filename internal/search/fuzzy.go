package search

import (
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/eg3r/fogit/pkg/fogit"
)

// Match represents a fuzzy search match with similarity score
type Match struct {
	Feature   *fogit.Feature
	Score     float64 // 0-100, higher is better
	MatchType string  // "exact", "levenshtein", "token-based"
	Matched   string  // What matched (name, description, etc.)
}

// SearchConfig holds fuzzy search configuration
type SearchConfig struct {
	FuzzyMatch     bool
	MinSimilarity  float64 // 0-100
	MaxSuggestions int
}

// DefaultSearchConfig returns default search configuration
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		FuzzyMatch:     true,
		MinSimilarity:  60.0,
		MaxSuggestions: 5,
	}
}

// FindSimilar finds features similar to the query
func FindSimilar(query string, features []*fogit.Feature, config SearchConfig) []Match {
	if !config.FuzzyMatch {
		return nil
	}

	var matches []Match

	for _, feature := range features {
		// Check name similarity
		nameScore := CalculateSimilarity(query, feature.Name)
		if nameScore >= config.MinSimilarity {
			matchType := "levenshtein"
			if nameScore == 100.0 {
				matchType = "exact"
			}
			matches = append(matches, Match{
				Feature:   feature,
				Score:     nameScore,
				MatchType: matchType,
				Matched:   "name",
			})
			continue
		}

		// Check description similarity (if name didn't match well enough)
		if feature.Description != "" {
			descScore := CalculateSimilarity(query, feature.Description)
			if descScore >= config.MinSimilarity {
				matches = append(matches, Match{
					Feature:   feature,
					Score:     descScore,
					MatchType: "token-based",
					Matched:   "description",
				})
				continue
			}
		}

		// Token-based matching (word by word)
		tokenScore := TokenBasedSimilarity(query, feature.Name)
		if tokenScore >= config.MinSimilarity {
			matches = append(matches, Match{
				Feature:   feature,
				Score:     tokenScore,
				MatchType: "token-based",
				Matched:   "name",
			})
		}
	}

	// Sort by score (highest first)
	sortMatches(matches)

	// Limit to max suggestions
	if len(matches) > config.MaxSuggestions {
		matches = matches[:config.MaxSuggestions]
	}

	return matches
}

// CalculateSimilarity calculates similarity between two strings using Levenshtein distance
// Returns a score from 0-100 (100 = identical)
func CalculateSimilarity(s1, s2 string) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 100.0
	}

	if s1 == "" || s2 == "" {
		return 0.0
	}

	// Use library's Levenshtein distance
	distance := fuzzy.LevenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	// Convert distance to similarity percentage
	similarity := (1.0 - float64(distance)/float64(maxLen)) * 100.0

	if similarity < 0 {
		return 0
	}
	return similarity
}

// TokenBasedSimilarity calculates similarity based on matching words/tokens
func TokenBasedSimilarity(s1, s2 string) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 100.0
	}

	tokens1 := tokenize(s1)
	tokens2 := tokenize(s2)

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	// Count matching tokens
	matches := 0
	for _, t1 := range tokens1 {
		for _, t2 := range tokens2 {
			if t1 == t2 {
				matches++
				break
			}
		}
	}

	// Calculate Jaccard similarity
	union := len(tokens1) + len(tokens2) - matches
	if union == 0 {
		return 0.0
	}

	similarity := (float64(matches) / float64(union)) * 100.0
	return similarity
}

// tokenize splits a string into words
func tokenize(s string) []string {
	// Split on whitespace and common separators
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "/", " ")

	words := strings.Fields(s)

	// Filter out very short tokens
	var tokens []string
	for _, word := range words {
		if len(word) > 1 {
			tokens = append(tokens, word)
		}
	}

	return tokens
}

// sortMatches sorts matches by score (highest first)
func sortMatches(matches []Match) {
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}
