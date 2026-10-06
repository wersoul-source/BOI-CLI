---
name: web-helper
description: Web search and fetch — retrieve information from the internet
version: "1.0"
requires:
  - shell
---

# Web Search & Fetch Skill

## Purpose
Search and retrieve information from the web for research and learning.

## Tools

### Web Search
Use your configured search tool or:
```bash
curl -s "https://api.duckduckgo.com/?q=<query>&format=json"
```

### Web Fetch
```bash
curl -sL "<url>" | head -n 200
```

### Documentation Lookup
For package documentation:
```bash
# Go packages
go doc <package>

# Python packages
python3 -c "help('<module>')"
```

## Best Practices
- Always verify information from multiple sources
- Cite the source URL in your response
- Respect robots.txt and rate limits
- Cache frequently accessed pages
