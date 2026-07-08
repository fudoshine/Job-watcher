package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type JobSpyConfig struct {
	Sites                    []string `yaml:"sites"`
	SearchTerm               string   `yaml:"search_term"`
	GoogleSearchTerm         string   `yaml:"google_search_term"`
	Location                 string   `yaml:"location"`
	Distance                 int      `yaml:"distance"`
	IsRemote                 bool     `yaml:"is_remote"`
	JobType                  string   `yaml:"job_type"`
	EasyApply                *bool    `yaml:"easy_apply"`
	ResultsWanted            int      `yaml:"results_wanted"`
	HoursOld                 int      `yaml:"hours_old"`
	CountryIndeed            string   `yaml:"country_indeed"`
	LinkedInFetchDescription bool     `yaml:"linkedin_fetch_description"`
	Offset                   int      `yaml:"offset"`
	UserAgent                string   `yaml:"user_agent"`
	Proxies                  []string `yaml:"proxies"`
	Verbose                  int      `yaml:"verbose"`
	Python                   string   `yaml:"python"`
}

type jobSpyRecord struct {
	ID           string `json:"id"`
	Site         string `json:"site"`
	JobURL       string `json:"job_url"`
	JobURLDirect string `json:"job_url_direct"`
	Title        string `json:"title"`
	Company      string `json:"company"`
	Location     string `json:"location"`
	DatePosted   string `json:"date_posted"`
	IsRemote     *bool  `json:"is_remote"`
	Description  string `json:"description"`
}

var jobSpySourceNames = map[string]string{
	"linkedin":      "LinkedIn",
	"indeed":        "Indeed",
	"glassdoor":     "Glassdoor",
	"google":        "Google Jobs",
	"zip_recruiter": "ZipRecruiter",
	"bayt":          "Bayt",
	"naukri":        "Naukri",
	"bdjobs":        "BDJobs",
}

func fetchJobSpyJobs(cfg Config) ([]Job, error) {
	return fetchJobSpyJobsForSites(cfg, cfg.JobSpy.Sites)
}

func fetchJobSpyJobsForSites(cfg Config, sites []string) ([]Job, error) {
	scriptPath, err := filepath.Abs("jobspy_runner.py")
	if err != nil {
		return nil, fmt.Errorf("jobspy runner path error: %w", err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("jobspy runner missing at %s", scriptPath)
	}

	python := strings.TrimSpace(cfg.JobSpy.Python)
	if python == "" {
		python = strings.TrimSpace(os.Getenv("JOBSPY_PYTHON"))
	}
	if python == "" {
		python = "python3"
	}

	normalizedSites := normalizeJobSpySites(sites)

	args := []string{scriptPath}
	if len(normalizedSites) > 0 {
		args = append(args, "--sites", strings.Join(normalizedSites, ","))
	}
	if cfg.JobSpy.SearchTerm != "" {
		args = append(args, "--search-term", cfg.JobSpy.SearchTerm)
	}
	if cfg.JobSpy.GoogleSearchTerm != "" {
		args = append(args, "--google-search-term", cfg.JobSpy.GoogleSearchTerm)
	}
	if cfg.JobSpy.Location != "" {
		args = append(args, "--location", cfg.JobSpy.Location)
	}
	if cfg.JobSpy.Distance > 0 {
		args = append(args, "--distance", strconv.Itoa(cfg.JobSpy.Distance))
	}
	if cfg.JobSpy.IsRemote {
		args = append(args, "--is-remote")
	}
	if cfg.JobSpy.JobType != "" {
		args = append(args, "--job-type", cfg.JobSpy.JobType)
	}
	if cfg.JobSpy.EasyApply != nil {
		args = append(args, "--easy-apply", strconv.FormatBool(*cfg.JobSpy.EasyApply))
	}
	if cfg.JobSpy.ResultsWanted > 0 {
		args = append(args, "--results-wanted", strconv.Itoa(cfg.JobSpy.ResultsWanted))
	}
	if cfg.JobSpy.CountryIndeed != "" {
		args = append(args, "--country-indeed", cfg.JobSpy.CountryIndeed)
	}
	hoursOld := cfg.JobSpy.HoursOld
	if hoursOld == 0 && cfg.MaxDaysOld > 0 {
		hoursOld = cfg.MaxDaysOld * 24
	}
	if hoursOld > 0 {
		args = append(args, "--hours-old", strconv.Itoa(hoursOld))
	}
	if cfg.JobSpy.LinkedInFetchDescription {
		args = append(args, "--linkedin-fetch-description")
	}
	if cfg.JobSpy.Offset > 0 {
		args = append(args, "--offset", strconv.Itoa(cfg.JobSpy.Offset))
	}
	if cfg.JobSpy.UserAgent != "" {
		args = append(args, "--user-agent", cfg.JobSpy.UserAgent)
	}
	if len(cfg.JobSpy.Proxies) > 0 {
		args = append(args, "--proxies", strings.Join(cfg.JobSpy.Proxies, ","))
	}
	if cfg.JobSpy.Verbose > 0 {
		args = append(args, "--verbose", strconv.Itoa(cfg.JobSpy.Verbose))
	}

	cmd := exec.Command(python, args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("jobspy runner failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var records []jobSpyRecord
	if err := json.Unmarshal(output, &records); err != nil {
		return nil, fmt.Errorf("jobspy output parse failed: %w", err)
	}

	jobs := make([]Job, 0, len(records))
	for _, record := range records {
		if job, ok := jobFromJobSpyRecord(record); ok {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

func normalizeJobSpySites(sites []string) []string {
	seen := make(map[string]bool)
	var normalized []string
	for _, site := range sites {
		site = strings.ToLower(strings.TrimSpace(site))
		if site == "" || seen[site] {
			continue
		}
		seen[site] = true
		normalized = append(normalized, site)
	}
	return normalized
}

func configuredJobSpySites(cfg Config) []string {
	sites := []string{}
	if cfg.Sources["jobspy"] {
		sites = append(sites, cfg.JobSpy.Sites...)
	}

	for _, site := range []string{"indeed", "linkedin", "naukri"} {
		if cfg.Sources[site] {
			sites = append(sites, site)
		}
	}

	return normalizeJobSpySites(sites)
}

func hasJobSpySite(sites []string, site string) bool {
	site = strings.ToLower(strings.TrimSpace(site))
	for _, candidate := range sites {
		if strings.ToLower(strings.TrimSpace(candidate)) == site {
			return true
		}
	}
	return false
}

func jobFromJobSpyRecord(record jobSpyRecord) (Job, bool) {
	site := strings.ToLower(strings.TrimSpace(record.Site))
	link := strings.TrimSpace(record.JobURLDirect)
	if link == "" {
		link = strings.TrimSpace(record.JobURL)
	}
	title := strings.TrimSpace(record.Title)
	if title == "" || link == "" {
		return Job{}, false
	}

	if record.Company != "" {
		title = fmt.Sprintf("%s @ %s", title, strings.TrimSpace(record.Company))
	}

	location := strings.TrimSpace(record.Location)
	if location == "" && record.IsRemote != nil && *record.IsRemote {
		location = "Remote"
	}
	if location != "" {
		title = fmt.Sprintf("%s (%s)", title, location)
	}

	source := jobSpySourceName(site)
	date := parseJobSpyDate(record.DatePosted)

	idSeed := strings.TrimSpace(record.ID)
	if idSeed == "" {
		idSeed = link
	}
	if idSeed == "" {
		idSeed = title
	}
	jobID := fmt.Sprintf("jobspy-%s-%s", site, shortHash(idSeed))
	if site == "" {
		jobID = fmt.Sprintf("jobspy-%s", shortHash(idSeed))
	}

	return Job{
		ID:          jobID,
		Title:       title,
		Link:        link,
		Source:      source,
		Date:        date,
		Description: strings.TrimSpace(record.Description),
	}, true
}

func jobSpySourceName(site string) string {
	normalized := strings.ToLower(strings.TrimSpace(site))
	if normalized == "" {
		return "JobSpy"
	}
	if name, ok := jobSpySourceNames[normalized]; ok {
		return name
	}
	return normalized
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func parseJobSpyDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}
