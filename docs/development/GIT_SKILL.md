# Git & GitHub CLI Skill

> ศึกษาจาก: GitHub CLI docs, git-scm, conventional commits, semantic-release
> โดย: คำปัน (Kampun) — BOI Family

---

## 🚀 Quick Reference

```bash
# ── Daily Workflow ──
git add -A
git commit -m "type: description"
git push

# ── Tag & Release ──
git tag -a v0.1.0 -m "v0.1.0: description"
git push origin v0.1.0
# → GitHub Actions auto-builds + releases via goreleaser

# ── Check Status ──
git status --short
git log --oneline -5
git tag -l

# ── Release Management ──
gh release list --limit 5
gh release view v0.1.0
gh run list --limit 5
gh run watch <run-id>
```

---

## 📋 Commit Convention

```
type: short description

Optional body with details.

Format: type: description
Types:  feat, fix, docs, chore, refactor, test, ci, style
```

| Type | When |
|------|------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `chore` | Maintenance, cleanup |
| `refactor` | Code restructuring |
| `ci` | CI/CD changes |

---

## 🏷️ Release Flow

```
1. Commit changes
   git add -A && git commit -m "feat: description"

2. Push to master
   git push

3. Create tag
   git tag -a v0.1.0 -m "v0.1.0: description"

4. Push tag (triggers goreleaser)
   git push origin v0.1.0

5. Wait for build
   gh run list --limit 3

6. Verify release
   gh release view v0.1.0
```

---

## 🔧 Common Fixes

**Wrong commit message?**
```bash
git commit --amend -m "new message"
git push --force-with-lease  # only if not yet pulled by others
```

**Forgot to add files?**
```bash
git add forgotten-file.go
git commit --amend --no-edit
git push --force-with-lease
```

**Tag already exists locally but not pushed?**
```bash
git tag -d v0.1.0 && git tag -a v0.1.0 -m "new message"
```

**Release failed?**
```bash
gh run list --workflow release.yml
gh run view <run-id> --log
```

---

## 📊 GitHub CLI Essentials

```bash
gh auth status          # Check login
gh repo view            # View repo info
gh release list         # List releases
gh release create       # Create release
gh pr list              # List PRs
gh pr create            # Create PR
gh issue list           # List issues
gh run list             # List workflow runs
gh run watch <id>       # Watch workflow live
```
