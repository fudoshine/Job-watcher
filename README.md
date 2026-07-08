# Job Watcher

Job Watcher is a powerful, automated tool designed to help developers find job opportunities efficiently. It aggregates job listings from multiple sources, filters them based on your preferences, scores them using AI (optional), and delivers real-time notifications directly to your Telegram.

## Features

-   **Multi-Source Aggregation**: Fetches jobs from:
    -   Indeed
    -   LinkedIn
    -   Naukri
    -   Instahyre
    -   RemoteOK
    -   Razorpay
    -   Wellfound
    -   YCombinator (Work at a Startup)
    -   Hacker News ("Who's Hiring")
    -   Reddit (r/cscareerquestions, r/forhire)
    -   Triplebyte
    -   Company Career Pages (350+ tech companies including startups, mid-size, and enterprises)
    -   Shared Lists (Google Sheets / GitHub Tables)
-   **JobSpy Integration (Python)**: Optional multi-board scraper for LinkedIn/Indeed/Google/Glassdoor/ZipRecruiter.
-   **High Performance**: Fetches from all sources in parallel for maximum speed.
-   **Smart Filtering**:
    -   Keyword matching (titles, skills)
    -   Experience level filtering
    -   Location filtering (Remote/India specific)
    -   Exclusion keywords (avoid senior/manager roles if desired)
    -   Date filtering (ignore old job postings)
-   **AI Scoring**: Integrates with AI (Ollama or Google Gemini) to score jobs based on your resume and preferences.
-   **Real-time Alerts**: Sends instant notifications to Telegram with job details and links.
-   **Deduplication**: Tracks seen jobs to prevent duplicate alerts.

## Prerequisites

-   **Go**: Version 1.21 or higher.
-   **Python**: Version 3.10+ (only if you enable JobSpy).
-   **Telegram Bot**: You need a Bot Token and Chat ID to receive alerts.
-   **Ollama** (Optional): Only required if you want AI-powered job scoring and filtering.

## Installation

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/yourusername/job-watcher.git
    cd job-watcher
    ```

2.  **Install dependencies**:
    ```bash
    go mod download
    ```

3.  **(Optional) Install JobSpy (Python) dependencies**:
    ```bash
    python3 -m pip install --upgrade pip
    pip install -e ./JobSpy
    ```

4.  **Setup Configuration**:
    -   The project uses `config.yaml` to define search parameters. Modify it to match your skills and preferences.
    -   Create a `.env` file for your secrets (see configuration below).

### AI Configuration (Optional)

You can use either **Ollama** (local) or **Google Gemini API** (cloud) to score jobs.

#### Option A: Ollama (Local - Free, Private)
1.  Install [Ollama](https://ollama.com/).
2.  Pull the model: `ollama pull llama3.1:latest`
3.  In `config.yaml`:
    ```yaml
    ai:
      enabled: true
      provider: "ollama"
      model: "llama3.1:latest"
    ```

#### Option B: Gemini API (Cloud - Fast, No RAM usage)
1.  Get a free API Key from [Google AI Studio](https://aistudio.google.com/app/apikey).
2.  Add it to your `.env` file:
    ```bash
    GEMINI_API_KEY=your_api_key_here
    ```
3.  In `config.yaml`:
    ```yaml
    ai:
      enabled: true
      provider: "gemini"
      model: "gemini-2.0-flash"
    ```

### Configuration

### 1. Telegram Bot Setup

To receive notifications, you need to set up a Telegram bot:

1.  **Create a Bot**:
    -   Open Telegram and search for **@BotFather**.
    -   Send the command `/newbot`.
    -   Follow the instructions to name your bot.
    -   **BotFather** will give you a **HTTP API Token**. This is your `TG_TOKEN`.

2.  **Get Chat ID**:
    -   Start a chat with your new bot and send a message (e.g., "Hello").
    -   Visit this URL in your browser: `https://api.telegram.org/bot<YOUR_TG_TOKEN>/getUpdates`
    -   Look for the `"chat":{"id":12345678,...}` part in the JSON response.
    -   The number (e.g., `12345678`) is your `TG_CHAT`.

### 2. Environment Variables

Create a `.env` file in the root directory (you can use `.env.example` as a reference):

```bash
TG_TOKEN=your_telegram_bot_token
TG_CHAT=your_telegram_chat_id
```

### 3. Personalization

**This is the most important step!** The default configuration looks for generic software engineering jobs. To find jobs that match YOUR skills and experience, you need to customize it.

📖 **See the full guide: [CUSTOMIZATION.md](CUSTOMIZATION.md)**

**Quick customization checklist:**
- ✅ Update `keywords` in `config.yaml` with your tech stack (languages, frameworks, tools)
- ✅ Set your `locations` (cities or "remote")
- ✅ Adjust `max_experience_years` to match your level
- ✅ Add `exclude_keywords` to filter out senior roles (if you're a junior/mid-level)
- ✅ Paste your resume into `resume.txt` (if using AI scoring)
- ✅ Configure `max_days_old` to ignore old job postings (default: 5 days)

### 4. config.yaml

The `config.yaml` file controls the behavior of the scraper. For a detailed guide on how to personalize this for your specific needs (keywords, experience level, AI matching), please read **[CUSTOMIZATION.md](CUSTOMIZATION.md)**.

Key sections include:

-   `keywords`: List of technologies and roles you are interested in (e.g., "python", "react", "SDE").
-   `locations`: Target locations (e.g., "remote", "bangalore").
-   `exclude_keywords`: keywords to filter out (e.g., "senior", "lead").
-   `max_experience_years`: Maximum experience required for the job.
-   `sources`: Enable/disable specific job sources.
-   `jobspy`: JobSpy (Python) scraper settings (sites, search term, location).
-   `ai`: Configuration for AI scoring (enabled/disabled, model name, threshold).

## Usage

1.  **Run the watcher**:
    ```bash
    go run .
    ```
    Or build and run:
    ```bash
    go build -o job-watcher
    ./job-watcher
    ```

2.  **AI Feature**:
    If AI is enabled in `config.yaml`, make sure Ollama is running:
    ```bash
    ollama serve
    ```
    And ensure you have the configured model pulled (default is `llama3.1:latest`):
    ```bash
    ollama pull llama3.1:latest
    ```

## Automated Running (GitHub Actions)

Want the job watcher to run automatically every hour without keeping your computer on? Use GitHub Actions!

1.  **Add GitHub Secrets**:
    -   Go to your repository **Settings** → **Secrets and variables** → **Actions**
    -   Add these repository secrets:
        -   `TG_TOKEN` - Your Telegram bot token
        -   `TG_CHAT` - Your Telegram chat ID
        -   `GEMINI_API_KEY` - Your Gemini API key (if using AI scoring)
        -   `RESUME_TEXT` - Your full resume content (required for AI matching)

2.  **The workflow is already set up** in [`.github/workflows/job-watcher.yml`](.github/workflows/job-watcher.yml)
    -   Runs automatically every hour
    -   Completely free on public repositories
    -   You can also trigger it manually from the **Actions** tab

3.  **Verify it's working**:
    -   Check the **Actions** tab in your repository
    -   View run history and logs
    -   Job notifications will be sent to your Telegram

**Deduplication**: The workflow automatically creates and commits `jobs.json` after each run to track previously seen jobs. You'll see commits from `github-actions[bot]` - this prevents duplicate notifications.

**Resume Privacy**: Use the `RESUME_TEXT` secret to keep your resume private while enabling AI matching.

**Free Tier**: Public repos get unlimited minutes. Private repos get 2,000 free minutes/month (more than enough for hourly runs).

### Troubleshooting GitHub Actions

If some sources return 0 jobs in GitHub Actions:

1. **Check the Actions logs** - Go to Actions tab → Latest run → Expand "Run Job Watcher"
2. **JavaScript-required sites** - Some sites (Wellfound, Naukri) need JavaScript and won't work in GitHub Actions
3. **Test locally first** - Run `go run . --test-sources` on your machine to verify sources work
4. **Test individual sources** - Run `go run . --test-sources` to check each source

See **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** for detailed debugging steps.

## Testing

Test all sources individually:
```bash
go run . --test-sources
```

This will show which sources are working and which are failing, with sample jobs from each source.

## Contributing

We welcome contributions! Please see **[CONTRIBUTING.md](CONTRIBUTING.md)** for details on how to submit pull requests, report issues, and follows our code style.

## License

[MIT](LICENSE)
