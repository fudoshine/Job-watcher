# Personalizing Your Job Watcher

This guide explains how to tweak the project to find jobs that are relevant to YOU. All the magic happens in `config.yaml` and `resume.txt`.

## 1. Defining Your Skills & Roles
**File:** `config.yaml` -> `keywords`

This is the most important part. The watcher looks for these words in job titles and descriptions.

-   **Languages**: Add the languages you know (e.g., "python", "java", "golang").
-   **Frameworks**: specific tools you use (e.g., "react", "django", "kubernetes").
-   **Roles**: Titles you are looking for (e.g., "SDE", "DevOps Engineer").

**Example:**
```yaml
keywords:
  - python
  - aws
  - devops engineer
  - backend developer
```

## 2. Choosing Locations
**File:** `config.yaml` -> `locations`

Filter where you want to work.
-   **Specific Cities**: "bangalore", "london", "nyc".
-   **Remote**: "remote", "wfh".

**Example:**
```yaml
locations:
  - remote
  - berlin
  - munich
```

## 3. Experience Level
**File:** `config.yaml` -> `max_experience_years` & `exclude_keywords`

-   **`max_experience_years`**: Helps simple number-based filtering.
-   **`exclude_keywords`**: Removes roles that are too senior. If you are a fresher, keep "Senior", "Lead", "Principal" here. If you are a Senior, remove them!

## 4. AI Matching (The "Smart" Part)
**File:** `resume.txt` & `config.yaml`

If you enabled AI in `config.yaml`, the system reads `resume.txt` to understand *who you are*.

**Choosing a Provider:**
-   **Ollama**: Runs locally. Good for privacy. Needs 8GB+ RAM.
-   **Gemini**: Runs in cloud. Fast & lightweight. Needs API Key.

**How to get a Gemini API Key:**
1.  Go to [Google AI Studio](https://aistudio.google.com/app/apikey).
2.  Click **Create API Key**.
3.  Copy the key and add it to your `.env` file: `GEMINI_API_KEY=your_key`

**Action:**
1.  Open `resume.txt` and paste your resume text.
2.  In `config.yaml`, set `provider` to either `"ollama"` or `"gemini"`.

## 5. Enabling/Disabling Sources
**File:** `config.yaml` -> `sources`

Don't care about "Hacker News" jobs? Disable them to save time.

```yaml
sources:
  linkedin: true
  hnjobs: false  # Disabled
```

## 6. JobSpy (Python) Settings
**File:** `config.yaml` -> `jobspy`

If you enable `sources.jobspy: true`, configure the JobSpy search input:

```yaml
jobspy:
  sites: [indeed, linkedin, google]
  search_term: "software engineer OR backend developer"
  location: "India"
  country_indeed: "India"
  hours_old: 120
```

Install Python deps once:
```bash
python3 -m pip install --upgrade pip
pip install -e ./JobSpy
```

## 7. Date Filtering
You can filter out old jobs (only for sources that provide dates, like RemoteOK).

In `config.yaml`:
```yaml
# Date Filtering
max_days_old: 5   # Jobs older than 5 days will be ignored
```
*Note: Scraped sites (career pages) often don't provide reliable dates, so they may default to "new".*

## 8. Adding New Companies
To add more career pages to scrape:

1.  Open `companies.go`.
2.  Scroll to the `companyCareerPages` list.
3.  Add a new entry:
    ```go
    {Name: "NewCompany", URL: "https://company.com/careers", Selector: "a[href*='job']", LinkAttr: "href"},
    ```
    -   **Selector**: The CSS selector to find the job link (e.g., `a.job-link`).
    -   **LinkAttr**: The attribute containing the URL (usually `href`).

## 9. Advanced Constraints
You can tweak hardcoded constraints in `filter.go` or `main.go` if you know Go.
