# Managing Job Sources

## Overview

Your job watcher scans **350+ companies** plus multiple job boards. Not all sources work perfectly all the time. This guide helps you manage them.

## Quick Actions

### Disable a Failing Source

If a source consistently fails or returns 0 jobs, disable it in `config.yaml`:

```yaml
sources:
  triplebyte: false  # Disabled - requires login
  wellfound: false   # Disabled - requires JavaScript
  naukri: false      # Disabled - requires JavaScript
```

### Check Which Sources Are Working

```bash
go run . --test-sources
```

This shows each source and how many jobs it returns.

## Source Categories

### 1. Job Boards (External APIs)

| Source | Status | Jobs/Run | Notes |
|--------|--------|----------|-------|
| Indeed | ✅ Working | 50-100 | RSS feeds, very reliable |
| LinkedIn | ✅ Working | 10-30 | Guest API, may rate-limit |
| JobSpy (Python) | ✅ Working | 20-100 | Multi-board scraper (LinkedIn/Indeed/Google/Glassdoor). Requires Python deps |
| RemoteOK | ⚠️ Premium | 0 | Requires premium account |
| Wellfound | ❌ JS Required | 0 | Needs JavaScript, disabled |
| Naukri | ❌ JS Required | 0 | Needs JavaScript, disabled |
| Instahyre | ⚠️ Variable | 0-10 | API may require auth |

### 2. Aggregators

| Source | Status | Jobs/Run | Notes |
|--------|--------|----------|-------|
| YC Jobs | ⚠️ Variable | 0-20 | Has fallbacks, may fail |
| HN Jobs | ✅ Working | 10-30 | Algolia API, reliable |
| Reddit | ✅ Working | 5-15 | JSON API, reliable |
| Triplebyte | ❌ Limited | 0 | Requires login, disabled |
| Shared Lists | ✅ Working | 1000-2000 | Very reliable |

### 3. Company Career Pages

| Category | Companies | Expected Jobs | Notes |
|----------|-----------|---------------|-------|
| All Companies | 350+ | 1500-3000 | Some may fail occasionally |
| Indian Startups | 100+ | 500-1000 | High success rate |
| Global Tech | 100+ | 500-1000 | Very reliable |
| Mid-Size | 80+ | 300-500 | Good success rate |
| Remote-First | 50+ | 200-400 | Excellent for remote jobs |

## Common Issues

### Issue: Source Returns 0 Jobs

**Possible Causes:**
1. Site requires JavaScript (Wellfound, Naukri)
2. Site requires login (Triplebyte)
3. API endpoint changed (YC Jobs, Instahyre)
4. Rate limiting (temporary)
5. Site is down (temporary)

**Solution:**
```yaml
# Disable in config.yaml
sources:
  problematic_source: false
```

### Issue: Too Many Companies, Slow Execution

**Solution 1: Disable Entire Company Source**
```yaml
sources:
  companies: false  # Disables all 350+ companies
```

**Solution 2: Remove Specific Companies**

Edit `companies.go` and comment out companies you don't want:

```go
// Not interested in gaming companies
// {Name: "Nazara Technologies", URL: "...", ...},
// {Name: "Winzo", URL: "...", ...},
```

**Solution 3: Keep Only Specific Categories**

In `companies.go`, comment out entire batches:

```go
// ========== Indian Gaming & Entertainment (Batch 20) ==========
// Comment out this entire section if not interested
/*
{Name: "Nazara Technologies", URL: "...", ...},
{Name: "Winzo", URL: "...", ...},
// ... rest of gaming companies
*/
```

### Issue: Duplicate Jobs

**Cause:** Same job appears from multiple sources

**Solution:** The job watcher automatically deduplicates based on job ID. If you still see duplicates:

1. Check `jobs.json` is being updated
2. Verify GitHub Actions has write permissions
3. Increase `retention_days` in config.yaml

### Issue: Rate Limiting

**Symptoms:**
- 429 status codes in logs
- Temporarily blocked from a site

**Solution:**
1. Wait 1-24 hours (usually resolves automatically)
2. Reduce frequency in GitHub Actions workflow
3. Disable the problematic source temporarily

## Optimization Strategies

### Strategy 1: Focus on High-Value Sources

Keep only the most reliable sources:

**Tip:** If you enable `jobspy`, consider disabling the built-in `linkedin`/`indeed` sources to reduce duplicates.

```yaml
sources:
  indeed: true       # 50-100 jobs
  linkedin: true     # 10-30 jobs
  companies: true    # 1500-3000 jobs
  sharedlists: true  # 1000-2000 jobs
  hnjobs: true       # 10-30 jobs
  
  # Disable others
  remoteok: false
  razorpay: false
  wellfound: false
  naukri: false
  instahyre: false
  ycjobs: false
  reddit: false
  triplebyte: false
```

**Result:** Still get 2500-4000+ jobs, faster execution

### Strategy 2: Sector-Specific Companies

If you're only interested in specific sectors, edit `companies.go`:

**Example: Only Fintech**
```go
var companyCareerPages = []CompanyCareer{
    // Keep only fintech companies
    {Name: "Razorpay", URL: "...", ...},
    {Name: "Stripe", URL: "...", ...},
    {Name: "BharatPe", URL: "...", ...},
    // ... other fintech
    
    // Comment out or remove all others
}
```

### Strategy 3: Geographic Focus

**For India-only jobs:**
```go
// In companies.go, keep only:
// - Indian Unicorns & Startups
// - Big Tech India offices
// - Indian SaaS companies
// Comment out global companies
```

**For Remote-only jobs:**
```go
// In companies.go, keep only:
// - Remote-First Global Companies
// - Developer Tools & Infrastructure
// - AI/ML Startups (often remote)
// Comment out location-specific companies
```

## Monitoring Source Health

### Check Logs Regularly

In GitHub Actions logs, look for:

```
✓ SourceName: X jobs     # Working well
⚠️ SourceName: 0 jobs    # May need attention
❌ SourceName: error     # Failing, should disable
```

### Track Success Rate

Create a simple tracking sheet:

| Source | Week 1 | Week 2 | Week 3 | Action |
|--------|--------|--------|--------|--------|
| Indeed | 80 | 75 | 82 | ✅ Keep |
| YC Jobs | 0 | 0 | 0 | ❌ Disable |
| Companies | 2000 | 1950 | 2100 | ✅ Keep |

### Set Expectations

**Reliable (>95% uptime):**
- Indeed, LinkedIn, HN Jobs, Shared Lists, Most Companies

**Variable (50-95% uptime):**
- YC Jobs, Instahyre, Reddit

**Unreliable (<50% uptime):**
- Triplebyte, Wellfound, Naukri, RemoteOK

## Advanced: Custom Source Priority

If you want to prioritize certain sources, you can modify the order in `main.go`:

```go
// High priority sources first
if cfg.Sources["indeed"] {
    // Fetch Indeed jobs
}

if cfg.Sources["linkedin"] {
    // Fetch LinkedIn jobs
}

// Lower priority sources later
if cfg.Sources["triplebyte"] {
    // Fetch Triplebyte jobs
}
```

## Recommended Configurations

### Configuration 1: Maximum Coverage (Default)
```yaml
sources:
  indeed: true
  linkedin: true
  companies: true
  sharedlists: true
  hnjobs: true
  reddit: true
  ycjobs: true
  instahyre: true
  
  # Disabled (don't work)
  remoteok: false
  wellfound: false
  naukri: false
  triplebyte: false
```

**Expected:** 2500-5000 jobs/run, 60-90 seconds

### Configuration 2: Fast & Reliable
```yaml
sources:
  indeed: true
  linkedin: true
  companies: true
  sharedlists: true
  
  # Disable all others
  hnjobs: false
  reddit: false
  ycjobs: false
  instahyre: false
  remoteok: false
  wellfound: false
  naukri: false
  triplebyte: false
```

**Expected:** 2000-4000 jobs/run, 40-60 seconds

### Configuration 3: Startup Focus
```yaml
sources:
  ycjobs: true
  hnjobs: true
  companies: true  # Keep only startup companies in companies.go
  
  # Disable corporate sources
  indeed: false
  linkedin: false
  sharedlists: false
  reddit: false
  instahyre: false
  remoteok: false
  wellfound: false
  naukri: false
  triplebyte: false
```

**Expected:** 500-1000 jobs/run, 30-45 seconds

## Troubleshooting Commands

```bash
# Test all sources
go run . --test-sources

# Test locally with full run
go run .

# Check which companies are in the list
grep "Name:" companies.go | wc -l

# Find a specific company
grep -i "company_name" companies.go

# Check config
cat config.yaml | grep -A 20 "sources:"
```

## When to Disable vs Remove

### Disable (in config.yaml)
- Temporary issues
- Want to try again later
- Easy to re-enable

### Remove (from code)
- Permanently broken
- Never coming back (like Triplebyte)
- Want cleaner codebase

### Comment Out (in companies.go)
- Not interested in that sector
- Too many companies
- Want to focus on specific types

## Summary

**Key Points:**
1. ✅ Not all sources will work 100% of the time - that's normal
2. ✅ Disable failing sources in `config.yaml`
3. ✅ Use `go run . --test-sources` to check health
4. ✅ Focus on high-value, reliable sources
5. ✅ Customize company list for your interests

**Recommended Action:**
- Keep the default config (Triplebyte already disabled)
- Monitor logs for 1 week
- Disable any source that consistently returns 0 jobs
- Enjoy 2500-5000+ jobs per run! 🚀
