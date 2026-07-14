package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Keywords           []string        `yaml:"keywords"`
	Locations          []string        `yaml:"locations"`
	ExcludeKeywords    []string        `yaml:"exclude_keywords"`
	MaxExperienceYears int             `yaml:"max_experience_years"`
	IndeedRSS          []string        `yaml:"indeed_rss"`
	Sources            map[string]bool `yaml:"sources"`
	JobSpy             JobSpyConfig    `yaml:"jobspy"`
	AI                 AIConfig        `yaml:"ai"`             // New AI config
	RetentionDays      int             `yaml:"retention_days"` // Days to keep job history
	MaxDaysOld         int             `yaml:"max_days_old"`   // Filter jobs older than X days
}

var cfg Config

func loadConfig() Config {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Warning: Could not read config.yaml, using defaults")
		return Config{
			Sources: map[string]bool{
				"remoteok": true,
			},
			RetentionDays: 30,
		}
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		fmt.Println("Warning: Could not parse config.yaml")
	}

	// Default retention if not set
	if c.RetentionDays == 0 {
		c.RetentionDays = 30
	}

	return c
}

// JobRecord stores job with timestamp for tracking when it was first seen
type JobRecord struct {
	Job       Job   `json:"job"`
	FirstSeen int64 `json:"first_seen"` // Unix timestamp
}

func loadOldJobs() map[string]int64 {
	file, err := os.ReadFile("jobs.json")
	if err != nil {
		return map[string]int64{}
	}

	// Try new format first (with timestamps)
	var records []JobRecord
	if err := json.Unmarshal(file, &records); err == nil && len(records) > 0 {
		seen := map[string]int64{}
		for _, r := range records {
			seen[r.Job.ID] = r.FirstSeen
		}
		return seen
	}

	// Fallback: old format (just jobs without timestamps)
	var jobs []Job
	json.Unmarshal(file, &jobs)

	seen := map[string]int64{}
	now := time.Now().Unix()
	for _, j := range jobs {
		seen[j.ID] = now // Assume they were seen now
	}
	return seen
}

func saveJobRecords(jobs []Job, existingRecords map[string]int64) {
	now := time.Now().Unix()
	var records []JobRecord

	// Calculate cutoff time for retention
	retentionDays := cfg.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30 // Safety fallback
	}
	cutoff := now - int64(retentionDays*24*60*60)

	// Add existing jobs with their original timestamps
	for id, ts := range existingRecords {
		// Cleanup: Skip references older than retention period
		if ts < cutoff {
			continue
		}

		// Keep record even if not in current fetch
		records = append(records, JobRecord{
			Job:       Job{ID: id},
			FirstSeen: ts,
		})
	}

	// Add new jobs
	for _, j := range jobs {
		if _, exists := existingRecords[j.ID]; !exists {
			records = append(records, JobRecord{
				Job:       j,
				FirstSeen: now,
			})
		}
	}

	data, _ := json.MarshalIndent(records, "", "  ")
	os.WriteFile("jobs.json", data, 0644)
}

func sendTelegram(msg string) {
	token := os.Getenv("TG_TOKEN")
	chat := os.Getenv("TG_CHAT")

	if token == "" || chat == "" {
		fmt.Println("Warning: TG_TOKEN or TG_CHAT not set, skipping Telegram notification")
		fmt.Println(msg)
		return
	}

	// Telegram has a 4096 character limit per message
	const maxLen = 4000

	// Split message if too long
	messages := []string{}
	if len(msg) <= maxLen {
		messages = append(messages, msg)
	} else {
		// Split by job entries
		for len(msg) > maxLen {
			// Find a good split point (newline before maxLen)
			splitIdx := maxLen
			for i := maxLen; i > maxLen-500; i-- {
				if msg[i] == '\n' {
					splitIdx = i
					break
				}
			}
			messages = append(messages, msg[:splitIdx])
			msg = msg[splitIdx:]
		}
		if len(msg) > 0 {
			messages = append(messages, msg)
		}
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	for i, m := range messages {
		data := fmt.Sprintf("chat_id=%s&text=%s&disable_web_page_preview=true",
			chat,
			url.QueryEscape(m))

		resp, err := http.Post(apiURL, "application/x-www-form-urlencoded", bytes.NewBufferString(data))
		if err != nil {
			fmt.Printf("Error sending Telegram message %d: %v\n", i+1, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("Telegram API returned status %d for message %d\n", resp.StatusCode, i+1)
		}
	}
	fmt.Printf("Sent %d Telegram message(s)\n", len(messages))
}

func main() {
	// Health checks deliberately avoid notifications and jobs.json writes.
	if len(os.Args) > 1 && os.Args[1] == "--test-sources" {
		cfg = loadConfig()
		if len(cfg.Keywords) > 0 {
			initKeywords(cfg)
		}
		os.Exit(testSources(cfg, os.Args[2:]))
	}

	// Load .env file for Telegram credentials
	if err := godotenv.Load(); err != nil {
		fmt.Println("Note: No .env file found, using environment variables")
	}

	// Load configuration
	cfg = loadConfig()

	// Initialize keywords from config
	if len(cfg.Keywords) > 0 {
		initKeywords(cfg)
	}

	old := loadOldJobs()
	fmt.Printf("Loaded %d previously seen jobs for deduplication\n", len(old))

	// Fetch from all sources in parallel for speed
	fmt.Println("🚀 Fetching jobs in parallel...")
	startTime := time.Now()

	var jobs []Job
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Helper to add jobs safely
	addJobs := func(source string, newJobs []Job) {
		mu.Lock()
		jobs = append(jobs, newJobs...)
		fmt.Printf("  ✓ %s: %d jobs\n", source, len(newJobs))
		mu.Unlock()
	}

	jobSpySites := configuredJobSpySites(cfg)
	if len(jobSpySites) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if jobSpyJobs, err := fetchJobSpyJobsForSites(cfg, jobSpySites); err == nil {
				addJobs(fmt.Sprintf("JobSpy (%s)", strings.Join(jobSpySites, ", ")), jobSpyJobs)
			} else {
				fmt.Printf("  Warning: JobSpy failed: %v\n", err)
			}
		}()
	}

	// RemoteOK
	if cfg.Sources["remoteok"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if remoteJobs, err := fetchJobs(); err == nil {
				addJobs("RemoteOK", remoteJobs)
			}
		}()
	}

	// Razorpay
	if cfg.Sources["razorpay"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if razorpayJobs, err := fetchRazorpayJobs(); err == nil {
				addJobs("Razorpay", razorpayJobs)
			}
		}()
	}

	// Wellfound
	if cfg.Sources["wellfound"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if wellfoundJobs, err := fetchWellfoundJobs(); err == nil {
				addJobs("Wellfound", wellfoundJobs)
			} else {
				fmt.Printf("  Warning: Wellfound failed: %v\n", err)
			}
		}()
	}

	// Indeed
	if cfg.Sources["indeed"] && !hasJobSpySite(jobSpySites, "indeed") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if indeedJobs, err := fetchIndeedJobs(cfg.IndeedRSS); err == nil {
				addJobs("Indeed", indeedJobs)
			} else {
				fmt.Printf("  Warning: Indeed failed: %v\n", err)
			}
		}()
	}

	// LinkedIn
	if cfg.Sources["linkedin"] && !hasJobSpySite(jobSpySites, "linkedin") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if linkedinJobs, err := fetchLinkedInJobs(); err == nil {
				addJobs("LinkedIn", linkedinJobs)
			}
		}()
	}

	// Naukri
	if cfg.Sources["naukri"] && !hasJobSpySite(jobSpySites, "naukri") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if naukriJobs, err := fetchNaukriJobs(); err == nil {
				addJobs("Naukri", naukriJobs)
			} else {
				fmt.Printf("  Warning: Naukri failed: %v\n", err)
			}
		}()
	}

	// Instahyre
	if cfg.Sources["instahyre"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if instahyreJobs, err := fetchInstahyreJobs(); err == nil {
				addJobs("Instahyre", instahyreJobs)
			}
		}()
	}

	// Company Career Pages (parallel within)
	if cfg.Sources["companies"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("  📋 Scanning company career pages...")
			if companyJobs, err := fetchAllCompanyJobsParallel(); err == nil {
				addJobs("Companies", companyJobs)
			}
		}()
	}

	// YCombinator Jobs (workatastartup.com)
	if cfg.Sources["ycjobs"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if ycJobs, err := fetchYCJobs(); err == nil {
				addJobs("YC Jobs", ycJobs)
			}
		}()
	}

	// Hacker News Who's Hiring
	if cfg.Sources["hnjobs"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if hnJobs, err := fetchHNJobs(); err == nil {
				addJobs("HN Jobs", hnJobs)
			}
		}()
	}

	// Reddit (r/cscareerquestions, r/forhire)
	if cfg.Sources["reddit"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if redditJobs, err := fetchRedditJobs(); err == nil {
				addJobs("Reddit", redditJobs)
			}
		}()
	}

	// Triplebyte/Karat
	if cfg.Sources["triplebyte"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tbJobs, err := fetchTriplebyteJobs(); err == nil {
				addJobs("Triplebyte", tbJobs)
			}
		}()
	}

	// Shared Lists (Google Sheets / GitHub Tables)
	if cfg.Sources["sharedlists"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if listJobs, err := fetchSharedListJobs(); err == nil {
				addJobs("Shared Lists", listJobs)
			}
		}()
	}

	// Wait for all sources
	wg.Wait()
	elapsed := time.Since(startTime)
	fmt.Printf("\n⏱️  Fetched in %.1f seconds\n", elapsed.Seconds())

	// Filter for new eligible jobs (not seen before + matches filters)
	var newOnes []Job
	for _, j := range jobs {
		_, alreadySeen := old[j.ID]
		if !alreadySeen && isEligibleJob(j) {
			newOnes = append(newOnes, j)
		}
	}

	fmt.Printf("\nTotal jobs fetched: %d\n", len(jobs))
	fmt.Printf("New jobs matching keywords: %d\n", len(newOnes))

	var finalJobs []Job

	// AI Matching Pass
	if cfg.AI.Enabled && len(newOnes) > 0 {
		fmt.Printf("\n🤖 Running AI Matcher on %d candidate jobs...\n", len(newOnes))
		if err := loadResume(); err != nil {
			fmt.Printf("⚠️ AI skipped: %v\n", err)
			finalJobs = newOnes // Fallback to all
		} else {
			// Buffered channel to limit concurrency (even M3 Pro shouldn't do 100 at once)
			// A reasonable limit is 4-8 parallel models for 4B parameters
			concurrency := 5
			results := make(chan Job, len(newOnes))
			var wg sync.WaitGroup
			sem := make(chan struct{}, concurrency)

			fmt.Printf("⚡ Parallel AI Scoring enabled (Concurrency: %d)\n", concurrency)

			for i, j := range newOnes {
				wg.Add(1)
				go func(idx int, job Job) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release

					fmt.Printf("[%d/%d] Scoring: %s...\n", idx+1, len(newOnes), job.Title)
					score, _, err := scoreJobWithAI(job, cfg.AI)

					if err != nil {
						fmt.Printf("Error scoring %s: %v\n", job.Title, err)
						results <- job // Keep on error
						return
					}

					if score >= cfg.AI.Threshold {
						// ENRICHMENT STEP: If very high score (e.g. >= 90), try to find recruiters
						// Just show their LinkedIn profile link, no fancy email guessing (user request)
						job.Title = fmt.Sprintf("[AI: %d] %s", score, job.Title)

						results <- job
					}
				}(i, j)
			}

			go func() {
				wg.Wait()
				close(results)
			}()

			for j := range results {
				finalJobs = append(finalJobs, j)
			}
		}
	} else {
		finalJobs = newOnes
	}

	if len(finalJobs) > 0 {
		msg := "🚨 New Jobs Found:\n\n"
		for _, j := range finalJobs {
			msg += fmt.Sprintf("• %s\n%s\n\n", j.Title, j.Link)
		}
		sendTelegram(msg)
	}

	// Save all jobs with timestamps to track what we've seen
	saveJobRecords(jobs, old)
}

func loadAllJobs() []Job {
	file, err := os.ReadFile("jobs.json")
	if err != nil {
		return []Job{}
	}

	var jobs []Job
	json.Unmarshal(file, &jobs)
	return jobs
}
