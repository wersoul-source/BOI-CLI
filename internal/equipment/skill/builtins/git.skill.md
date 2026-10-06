---
name: git-helper
description: Git operations assistant — commit, branch, diff, merge workflow
version: "1.0"
requires:
  - shell
  - file-read
---

# Git Helper Skill

## Purpose
Automate common Git workflows with safety checks.

## Workflow

### Before Starting
1. Check current branch: `git branch --show-current`
2. Check status: `git status --short`
3. Check if there are uncommitted changes

### Commit Workflow
1. Review changes: `git diff` or `git diff --staged`
2. Stage files: `git add <files>`
3. Commit: `git commit -m "<message>"`
   - Use conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`

### Branch Workflow
1. List branches: `git branch -a`
2. Create branch: `git checkout -b <name>`
3. Switch branch: `git checkout <name>`

### Safety Rules
- NEVER force push to main/master
- NEVER commit secrets or API keys
- ALWAYS check git status before operations
- ALWAYS provide a meaningful commit message
