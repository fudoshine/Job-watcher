package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

// AIConfig holds settings for AI provider
type AIConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Provider  string `yaml:"provider"` // "ollama" or "gemini"
	Model     string `yaml:"model"`
	Threshold int    `yaml:"threshold"` // Score out of 100
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"` // json
}

type OllamaResponse struct {
	Response string `json:"response"`
}

type MatchResult struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

var (
	resumeText string
	aiMutex    sync.Mutex
)

// loadResume loads resume.txt into memory
func loadResume() error {
	data, err := os.ReadFile("resume.txt")
	if err != nil {
		return err
	}
	resumeText = string(data)
	if len(resumeText) < 50 {
		return fmt.Errorf("resume.txt seems too short, please paste your resume content")
	}
	return nil
}

// scoreJobWithAI sends job details to the configured AI provider
func scoreJobWithAI(job Job, cfg AIConfig) (int, string, error) {
	company := strings.TrimSpace(extractCompany(job))
	if company == "" {
		company = job.Source
	}
	description := strings.TrimSpace(job.Description)
	if description == "" {
		description = "N/A"
	} else {
		description = truncateText(description, 1200)
	}

	prompt := fmt.Sprintf(`Role: Hiring Manager. 
Task: Evaluate match for a Fresher/Entry-Level Candidate (0-2 YOE).

Candidate Resume:
%s

Job Title: %s
Company: %s
Job Description: %s

CRITICAL RULES:
1. IF Job Title contains "Senior", "Staff", "Lead", "Principal", "Architect", or requires >2 years experience: SCORE MUST BE 0.
2. IF Job is for "Intern", "New Grad", "Associate", "Junior", or "0-2 years": Score normally based on skill match.
3. IGNORE skill match if Rule #1 is violated.

Constraint: Return ONLY a JSON object with "score" (0-100) and "reason" (short string).
Example: {"score": 0, "reason": "Senior role (3+ years) not for fresher"}`, resumeText, job.Title, company, description)

	if strings.ToLower(cfg.Provider) == "gemini" {
		return callGeminiAI(cfg, prompt)
	}
	return callOllama(cfg, prompt)
}

func callOllama(cfg AIConfig, prompt string) (int, string, error) {
	reqBody := OllamaRequest{
		Model:  cfg.Model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, "", fmt.Errorf("ollama connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return 0, "", err
	}

	return parseJSONResponse(ollamaResp.Response)
}

func callGeminiAI(cfg AIConfig, prompt string) (int, string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return 0, "", fmt.Errorf("GEMINI_API_KEY not set")
	}

	model := cfg.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}

	// Ensure we don't double-prefix if user added "models/"
	model = strings.TrimPrefix(model, "models/")

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, "", fmt.Errorf("gemini connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, "", fmt.Errorf("gemini error %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return 0, "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return 0, "", fmt.Errorf("empty response from gemini")
	}

	return parseJSONResponse(geminiResp.Candidates[0].Content.Parts[0].Text)
}

func parseJSONResponse(raw string) (int, string, error) {
	var result MatchResult
	// Sometimes LLMs add markdown code blocks like ```json ... ```
	cleanResp := strings.TrimSpace(raw)
	cleanResp = strings.TrimPrefix(cleanResp, "```json")
	cleanResp = strings.TrimSuffix(cleanResp, "```")

	if err := json.Unmarshal([]byte(cleanResp), &result); err != nil {
		// Fallback: simple text parsing if JSON fails
		return 0, cleanResp, nil
	}

	return result.Score, result.Reason, nil
}

func truncateText(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}
