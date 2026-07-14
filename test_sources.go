package main

import (
	"fmt"
	"strings"
	"time"
)

type sourceHealthCheck struct {
	name       string
	aliases    []string
	enabled    bool
	skipReason string
	fetch      func() ([]Job, error)
}

// testSources runs each configured job-board path without sending Telegram
// notifications or changing jobs.json. An optional comma-separated source list
// can narrow the check, for example: go run . --test-sources indeed,naukri.
func testSources(config Config, filterArgs []string) int {
	checks := configuredSourceHealthChecks(config)
	requested := normalizeSourceHealthFilter(filterArgs)

	fmt.Println("🧪 Testing configured job sources")
	if len(requested) > 0 {
		fmt.Printf("   Filter: %s\n", strings.Join(filterArgs, ","))
	}
	fmt.Println("========================================================")

	matched := 0
	tested := 0
	working := 0
	failed := 0
	zeroResults := 0

	for _, check := range checks {
		if !matchesSourceHealthFilter(check, requested) {
			continue
		}
		matched++

		if check.skipReason != "" {
			fmt.Printf("⏭️  %s: %s\n", check.name, check.skipReason)
			continue
		}
		if !check.enabled {
			fmt.Printf("⏭️  %s: disabled in config\n", check.name)
			continue
		}

		tested++
		start := time.Now()
		fmt.Printf("🔍 Testing %s...\n", check.name)
		jobs, err := check.fetch()
		elapsed := time.Since(start).Round(time.Millisecond)
		if err != nil {
			failed++
			fmt.Printf("   ❌ Failed after %s: %v\n", elapsed, err)
			continue
		}

		if len(jobs) == 0 {
			zeroResults++
			fmt.Printf("   ⚠️  0 jobs after %s (site may be empty, blocked, or changed)\n", elapsed)
			continue
		}

		working++
		fmt.Printf("   ✅ %d jobs after %s\n", len(jobs), elapsed)
		for i, job := range jobs {
			if i == 3 {
				break
			}
			fmt.Printf("      - %s\n", job.Title)
		}
	}

	fmt.Println("========================================================")
	if matched == 0 {
		fmt.Println("❌ No configured source matched the requested filter.")
		return 2
	}

	fmt.Printf("Summary: %d working, %d failed, %d returned 0, %d tested\n", working, failed, zeroResults, tested)
	if tested == 0 || working == 0 {
		return 1
	}
	return 0
}

func configuredSourceHealthChecks(config Config) []sourceHealthCheck {
	jobSpySites := configuredJobSpySites(config)
	checks := make([]sourceHealthCheck, 0, len(jobSpySites)+10)

	for _, site := range jobSpySites {
		site := site
		checks = append(checks, sourceHealthCheck{
			name:    "JobSpy/" + jobSpySourceName(site),
			aliases: []string{"jobspy", site, "jobspy/" + site},
			enabled: true,
			fetch: func() ([]Job, error) {
				return fetchJobSpyJobsForSites(config, []string{site})
			},
		})
	}

	checks = append(checks,
		directSourceHealthCheck("RemoteOK", "remoteok", config.Sources["remoteok"], fetchJobs),
		directSourceHealthCheck("Razorpay", "razorpay", config.Sources["razorpay"], fetchRazorpayJobs),
		directSourceHealthCheck("Wellfound", "wellfound", config.Sources["wellfound"], fetchWellfoundJobs),
		directSourceHealthCheck("Instahyre", "instahyre", config.Sources["instahyre"], fetchInstahyreJobs),
		directSourceHealthCheck("YC Jobs", "ycjobs", config.Sources["ycjobs"], fetchYCJobs),
		directSourceHealthCheck("HN Jobs", "hnjobs", config.Sources["hnjobs"], fetchHNJobs),
		directSourceHealthCheck("Reddit", "reddit", config.Sources["reddit"], fetchRedditJobs),
		directSourceHealthCheck("Triplebyte", "triplebyte", config.Sources["triplebyte"], fetchTriplebyteJobs),
		directSourceHealthCheck("Shared Lists", "sharedlists", config.Sources["sharedlists"], fetchSharedListJobs),
		sourceHealthCheck{
			name:       "Companies",
			aliases:    []string{"companies"},
			enabled:    config.Sources["companies"],
			skipReason: "large career-page crawl; run the watcher for this source",
		},
	)

	if !hasJobSpySite(jobSpySites, "indeed") {
		checks = append(checks, directSourceHealthCheck("Indeed", "indeed", config.Sources["indeed"], func() ([]Job, error) {
			return fetchIndeedJobs(config.IndeedRSS)
		}))
	}
	if !hasJobSpySite(jobSpySites, "linkedin") {
		checks = append(checks, directSourceHealthCheck("LinkedIn", "linkedin", config.Sources["linkedin"], fetchLinkedInJobs))
	}
	if !hasJobSpySite(jobSpySites, "naukri") {
		checks = append(checks, directSourceHealthCheck("Naukri", "naukri", config.Sources["naukri"], fetchNaukriJobs))
	}

	return checks
}

func directSourceHealthCheck(name, key string, enabled bool, fetch func() ([]Job, error)) sourceHealthCheck {
	return sourceHealthCheck{
		name:    name,
		aliases: []string{key, strings.ToLower(name)},
		enabled: enabled,
		fetch:   fetch,
	}
}

func normalizeSourceHealthFilter(args []string) map[string]bool {
	filter := make(map[string]bool)
	for _, arg := range args {
		for _, value := range strings.Split(arg, ",") {
			value = strings.ToLower(strings.TrimSpace(value))
			if value != "" {
				filter[value] = true
			}
		}
	}
	return filter
}

func matchesSourceHealthFilter(check sourceHealthCheck, filter map[string]bool) bool {
	if len(filter) == 0 {
		return true
	}
	if filter[strings.ToLower(check.name)] {
		return true
	}
	for _, alias := range check.aliases {
		if filter[strings.ToLower(alias)] {
			return true
		}
	}
	return false
}
