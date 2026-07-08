# Troubleshooting Guide

## GitHub Actions Issues

### Problem: Some sources return 0 jobs

#### Common Causes:

1. **JavaScript-Required Sites**
   - Sites like Wellfound, Naukri, and some others require JavaScript to render content
   - GitHub Actions uses basic HTTP requests without a browser
   - **Solution**: These sources are disabled by default in `config.yaml`

2. **API Endpoint Changes**
   - Job sites frequently change their API endpoints
   - **Solution**: The code now includes fallback mechanisms and better error reporting

3. **Rate Limiting / Bot Detection**
   - Some sites may block requests from GitHub Actions IPs
   - **Solution**: Use proper User-Agent headers (already implemented)

#### Debugging Steps:

1. **Check GitHub Actions Logs**
   - Go to your repository → Actions tab
   - Click on the latest workflow run
   - Expand the "Run Job Watcher" step
   - Look for error messages or warnings

2. **Test Locally First**
   ```bash
   go run .
   ```
   - If it works locally but not in GitHub Actions, it's likely a network/IP issue

3. **Enable Debug Output**
   - The workflow now includes `DEBUG=true` environment variable
   - Check logs for detailed error messages

4. **Check Individual Sources**
   - Temporarily disable all sources except one in `config.yaml`
   - Run the workflow to isolate which source is failing

### Source-Specific Issues

#### YC Jobs (workatastartup.com)
- **Status**: May require JavaScript or API authentication
- **Fallback**: HTML scraping with multiple regex patterns
- **Fix Applied**: Enhanced error handling and multiple parsing strategies

#### Instahyre
- **Status**: API endpoint may have changed
- **Fallback**: Multiple API endpoint attempts
- **Fix Applied**: Try multiple API paths with better error messages

#### LinkedIn
- **Status**: Working via guest API
- **Note**: May be rate-limited if too many requests

#### Indeed
- **Status**: Working via RSS feeds
- **Note**: RSS URLs in config.yaml must be valid

#### JobSpy (Python)
- **Status**: Optional multi-board scraper
- **Error**: `JobSpy import failed` or missing module
- **Fix**: `python3 -m pip install --upgrade pip && pip install -e ./JobSpy` and ensure `python3` is on PATH (or set `JOBSPY_PYTHON`)

### Solutions Applied

1. **Better Error Handling**
   - Sources now return empty arrays instead of errors
   - Prevents one failing source from breaking the entire run

2. **Multiple Fallback Strategies**
   - Try API first, then HTML scraping
   - Multiple regex patterns for parsing

3. **Enhanced Logging**
   - Clear error messages showing which source failed and why
   - Status codes and error details in logs

4. **Graceful Degradation**
   - If a source fails, others continue working
   - Partial results are still useful

## Configuration Issues

### No Jobs Found

If you're getting 0 jobs across all sources:

1. **Check Keywords**
   - Your keywords in `config.yaml` might be too specific
   - Try broader terms like "software engineer", "developer"

2. **Check Experience Filter**
   - `max_experience_years` might be too restrictive
   - Try increasing it to 3-5 years

3. **Check Exclude Keywords**
   - You might be filtering out too many jobs
   - Review your `exclude_keywords` list

4. **Check Date Filter**
   - `max_days_old: 5` only shows very recent jobs
   - Try increasing to 7-14 days

### Telegram Not Working

1. **Check Secrets**
   - Verify `TG_TOKEN` and `TG_CHAT` are set in GitHub Secrets
   - Go to Settings → Secrets and variables → Actions

2. **Test Token**
   - Visit: `https://api.telegram.org/bot<YOUR_TOKEN>/getMe`
   - Should return bot information

3. **Test Chat ID**
   - Send a message to your bot
   - Visit: `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`
   - Find your chat ID in the response

## Performance Issues

### Workflow Takes Too Long

1. **Disable Slow Sources**
   - Company career pages can be slow (90+ sites)
   - Shared lists can be large
   - Disable in `config.yaml` if not needed

2. **Reduce AI Concurrency**
   - In `main.go`, reduce `concurrency := 5` to `concurrency := 3`

3. **Limit Job Sources**
   - Focus on 3-5 most relevant sources
   - Disable others in `config.yaml`

### Too Many Duplicate Notifications

1. **Check jobs.json**
   - Should be committed after each run
   - Verify GitHub Actions has write permissions

2. **Retention Period**
   - Adjust `retention_days` in `config.yaml`
   - Default is 30 days

## Testing Changes

### Test Locally Before Pushing

```bash
# Test the full flow
go run .

# Test specific source
# Temporarily disable others in config.yaml
```

### Manual Workflow Trigger

1. Go to Actions tab
2. Select "Job Watcher" workflow
3. Click "Run workflow"
4. Select branch and click "Run workflow"

### Check Workflow Logs

```bash
# View recent workflow runs
gh run list

# View specific run logs
gh run view <run-id> --log
```

## Common Error Messages

### "status 403" or "status 429"
- **Cause**: Rate limiting or IP blocking
- **Solution**: Add delays between requests, use better User-Agent

### "status 404"
- **Cause**: API endpoint changed
- **Solution**: Check site documentation, update URL

### "EOF" or "connection reset"
- **Cause**: Network timeout or server issue
- **Solution**: Increase timeout, retry logic (already implemented)

### "no jobs found in HTML"
- **Cause**: Site requires JavaScript
- **Solution**: Disable that source or use a headless browser (not recommended for GitHub Actions)

## Getting Help

1. **Check Logs First**
   - Most issues are visible in GitHub Actions logs

2. **Test Locally**
   - Reproduce the issue on your machine

3. **Open an Issue**
   - Include workflow logs
   - Specify which source is failing
   - Share your config.yaml (remove sensitive data)

4. **Community Support**
   - Check existing issues on GitHub
   - Search for similar problems

## Best Practices

1. **Start Simple**
   - Enable 2-3 sources first
   - Add more once working

2. **Monitor Regularly**
   - Check Actions tab weekly
   - Review job quality

3. **Update Keywords**
   - Refine based on results
   - Add new skills as you learn

4. **Respect Rate Limits**
   - Don't run workflow too frequently
   - Hourly is reasonable, every 5 minutes is not

5. **Keep Dependencies Updated**
   - Run `go get -u` periodically
   - Check for Go version updates
