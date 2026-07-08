package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	keywords        []string
	locations       []string
	excludeKeywords []string
	maxExpYears     int
	maxDaysOld      int
)

// Experience patterns to detect years of experience
var expPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(\d+)\+?\s*(?:to\s*\d+\s*)?(?:years?|yrs?)\s*(?:of\s*)?(?:experience|exp)?`),
	regexp.MustCompile(`(?:experience|exp)\s*(?:of\s*)?(\d+)\+?\s*(?:to\s*\d+\s*)?(?:years?|yrs?)`),
	regexp.MustCompile(`(\d+)\s*-\s*\d+\s*(?:years?|yrs?)`),
}

func initFilters(cfg Config) {
	keywords = cfg.Keywords
	locations = cfg.Locations
	excludeKeywords = cfg.ExcludeKeywords
	maxExpYears = cfg.MaxExperienceYears
	maxDaysOld = cfg.MaxDaysOld // Use the dedicated config field

	// Default to 2 years if not set (suitable for 1 year experience)
	if maxExpYears == 0 {
		maxExpYears = 2
	}
}

// matchesKeyword checks if text contains any of the keywords
func matchesKeyword(text string) bool {
	text = strings.ToLower(text)
	for _, k := range keywords {
		if strings.Contains(text, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// matchesLocation checks if job location/title contains India or Remote
func matchesLocation(text string) bool {
	if len(locations) == 0 {
		return true // No location filter
	}

	text = strings.ToLower(text)
	for _, loc := range locations {
		if strings.Contains(text, strings.ToLower(loc)) {
			return true
		}
	}
	return false
}

// hasExcludedKeyword checks if title contains senior/lead/etc.
func hasExcludedKeyword(title string) bool {
	title = strings.ToLower(title)
	for _, k := range excludeKeywords {
		if strings.Contains(title, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// extractExperience extracts required years from text
func extractExperience(text string) int {
	text = strings.ToLower(text)

	for _, pattern := range expPatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) > 1 {
			// Parse the first number found
			var years int
			fmt.Sscanf(matches[1], "%d", &years)
			return years
		}
	}
	return 0 // Unknown/not specified = assume entry level
}

// isEligibleJob checks all filters
func isEligibleJob(job Job) bool {
	combined := strings.ToLower(job.Title + " " + job.Link + " " + job.Description)
	keywordText := strings.ToLower(job.Title + " " + job.Description)

	// Must match at least one keyword
	if !matchesKeyword(keywordText) {
		return false
	}

	// Must not have excluded keywords (senior, lead, etc.)
	if hasExcludedKeyword(combined) {
		return false
	}

	// Date filter (if available)
	if !isRecentJob(job.Date) {
		return false
	}

	// Check experience requirement if detectable
	expYears := extractExperience(combined)
	if expYears > maxExpYears {
		return false
	}

	// Location check - RemoteOK jobs are remote by default
	if job.Source == "RemoteOK" {
		return true
	}

	// For other sources, check location in title/link
	if !matchesLocation(combined) {
		return false
	}

	return true
}

// isRecentJob checks if job is within maxDaysOld
func isRecentJob(date time.Time) bool {
	if maxDaysOld <= 0 || date.IsZero() {
		return true // No filter or no date available
	}
	// Check if date is after (Now - maxDaysOld)
	cutoff := time.Now().AddDate(0, 0, -maxDaysOld)
	return date.After(cutoff)
}

// Legacy function for compatibility
func initKeywords(cfg Config) {
	initFilters(cfg)
}
