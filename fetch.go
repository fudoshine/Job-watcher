package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Job struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Source      string    `json:"source,omitempty"`
	Date        time.Time `json:"date,omitempty"`        // For date filtering
	Description string    `json:"description,omitempty"` // Optional long text
}

func fetchJobs() ([]Job, error) {
	req, _ := http.NewRequest("GET", "https://remoteok.com/api", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	client := &http.Client{
		Timeout: 15 * time.Second, // Reduced from 30s to 15s
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var jobs []Job
	for _, j := range raw {
		if j["id"] == nil || j["position"] == nil {
			continue
		}

		// Handle ID which can be string or number
		var id string
		switch v := j["id"].(type) {
		case float64:
			id = fmt.Sprintf("%.0f", v)
		case string:
			id = v
		default:
			continue
		}

		// Handle URL - API returns full URLs or relative paths
		urlPath, ok := j["url"].(string)
		if !ok {
			continue
		}

		link := urlPath
		if len(urlPath) > 0 && urlPath[0] == '/' {
			link = "https://remoteok.com" + urlPath
		}

		jobs = append(jobs, Job{
			ID:     "remoteok-" + id,
			Title:  j["position"].(string),
			Link:   link,
			Source: "RemoteOK",
			Date:   parseRemoteOKDate(j["date"]),
		})
	}

	return jobs, nil
}

func parseRemoteOKDate(v interface{}) time.Time {
	if str, ok := v.(string); ok {
		// RemoteOK usually sends ISO strings like "2023-10-25T..."
		if t, err := time.Parse(time.RFC3339, str); err == nil {
			return t
		}
		// Fallback for simple dates YYYY-MM-DD
		if t, err := time.Parse("2006-01-02", str); err == nil {
			return t
		}
	}
	return time.Time{} // Zero value if parsing fails
}
