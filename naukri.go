package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// fetchNaukriJobs tries to scrape Naukri - may be limited without JS
func fetchNaukriJobs() ([]Job, error) {
	var allJobs []Job
	var failures []string

	searches := []string{
		"software-engineer-fresher",
		"backend-developer-fresher",
		"full-stack-developer-fresher",
	}

	for _, search := range searches {
		url := fmt.Sprintf("https://www.naukri.com/%s-jobs?experience=0-1&jobAge=1", search)
		jobs, err := scrapeNaukriPage(url)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		allJobs = append(allJobs, jobs...)
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []Job
	for _, j := range allJobs {
		if !seen[j.ID] {
			seen[j.ID] = true
			unique = append(unique, j)
		}
	}

	if len(unique) == 0 {
		if len(failures) > 0 {
			return nil, fmt.Errorf("all Naukri searches failed: %s", strings.Join(failures, "; "))
		}
		return nil, fmt.Errorf("no Naukri job cards found; the page may require JavaScript or its markup changed")
	}

	return unique, nil
}

func scrapeNaukriPage(url string) ([]Job, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var jobs []Job
	jobIDRegex := regexp.MustCompile(`-(\d+)\?`)

	// Try to find job listings
	doc.Find("a[href*='job-listings']").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		if link == "" {
			return
		}

		title := strings.TrimSpace(s.Text())
		if title == "" || len(title) > 100 {
			return
		}

		var jobID string
		if matches := jobIDRegex.FindStringSubmatch(link); len(matches) > 1 {
			jobID = matches[1]
		} else {
			jobID = fmt.Sprintf("%d", i)
		}

		if !strings.HasPrefix(link, "http") {
			link = "https://www.naukri.com" + link
		}

		jobs = append(jobs, Job{
			ID:     "naukri-" + jobID,
			Title:  title,
			Link:   link,
			Source: "Naukri",
		})
	})

	if len(jobs) == 0 {
		return nil, fmt.Errorf("no job listing links found; the page may require JavaScript or its markup changed")
	}

	return jobs, nil
}
