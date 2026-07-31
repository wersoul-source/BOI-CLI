# README Skill — World-Class CLI README Pattern Guide

> Reusable pattern guide for creating READMEs that convert visitors into users.
> Derived from studying 10 top CLI repos (gh, uv, ollama, bun, ripgrep, fzf, bat, fd, lazygit, zoxide).

---

## When to Use

- Creating a README for a new CLI tool or library
- Rewriting an existing README that underperforms
- Auditing a README against industry best practices
- Onboarding a new project that needs documentation

---

## The 10-Second Rule

A visitor decides whether to use your tool in **10 seconds**. Your README must answer three questions before they scroll:

1. **What does this do?** (one-liner)
2. **How do I install it?** (within first 20 lines)
3. **What does it look like?** (screenshot or demo)

If any of these are missing, you lose users.

---

## Pattern 1: Install Command First (CRITICAL)

```
Placement: Lines 1-20 of README
Format:    Single-line copy-paste command
Priority:  Highest — users leave without this
```

**Template:**
```bash
curl -fsSL https://install.yourtool.dev | sh
```

```bash
brew install yourtool
```

```bash
go install github.com/you/yourcli@latest
```

**Rules:**
- Must appear above the fold (visible without scrolling).
- Must be a single command that can be copied and pasted.
- If multiple platforms, use a compact table.
- Never bury install instructions behind a "Documentation" link.

---

## Pattern 2: Visual Above Fold (CRITICAL)

```
Placement: Lines 1-30 of README
Format:    Image, GIF, or ASCII art
Purpose:   Show value instantly, build trust
```

**Options:**
| Type | When to Use | Example |
|------|-------------|---------|
| Screenshot | GUI or TUI apps | `bat` showing syntax highlighting |
| GIF demo | Interactive tools | `fzf` fuzzy-finding in action |
| ASCII art | Terminal-first tools | `cowsay` style banners |
| Logo image | Branded projects | `ollama` llama logo |

**Rules:**
- One visual is enough. More than 2 is clutter.
- GIFs should be under 5 seconds.
- Screenshots should show the tool doing its primary job.
- For terminal tools, ASCII art logos work on GitHub natively.

---

## Pattern 3: One-Liner Description

```
Placement: First 3 lines after badges
Format:    Single sentence, ≤120 characters
Tone:      What it DOES, not what it IS
```

**Bad:**
> "BOI CLI is a revolutionary AI-powered terminal runtime built with cutting-edge technology."

**Good:**
> "An AI team of 6 personas in your terminal. One binary. No server."

**Template:**
```
[Tool Name] — [verb] [what it does] in [where it runs].
```

---

## Pattern 4: Badges

```
Placement: First 2-3 lines of README
Format:    Horizontal row of shield badges
Max:       5-6 badges
```

**Essential badges:**
```
[Version] [License] [Platform] [Go/Node/Python version] [Build Status]
```

**Template (Markdown):**
```markdown
<p align="center">
  <img src="https://img.shields.io/badge/version-1.0.0-blue" alt="Version">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  <img src="https://img.shields.io/badge/platform-win%20|%20mac%20|%20linux-lightgrey" alt="Platform">
</p>
```

---

## Pattern 5: Features as Bullet List

```
Placement: After Quick Start or Logo
Format:    Bullet points with emoji/icon prefix
Count:     4-7 items maximum
```

**Template:**
```markdown
## Features

| | |
|---|---|
| Icon | **Feature Name** — One line description (≤80 chars) |
| Icon | **Feature Name** — One line description (≤80 chars) |
```

or:

```markdown
- **Feature** — What it does, in one line.
- **Feature** — What it does, in one line.
```

**Rules:**
- Each feature = one line. Never paragraph.
- Use a consistent icon/emoji prefix.
- Order by importance to user, not implementation order.
- Skip features that aren't yet built.

---

## Pattern 6: Quick Start Section

```
Placement: After Features
Format:    3-5 copy-paste commands
Purpose:   Get user to first successful run in under 60 seconds
```

**Template:**
```markdown
## Quick Start

```bash
# 1. Clone
git clone https://github.com/you/repo.git
cd repo

# 2. Build
go build -o bin/tool ./cmd/tool

# 3. Run
./bin/tool init
./bin/tool
\```
```

**Rules:**
- Every command must work as-is (no user edits required).
- Use comments to explain each step.
- 3 commands ideal, 5 maximum.
- End with a visible success (the tool running).

---

## Pattern 7: Community Section

```
Placement: Near bottom, before footer
Contents: Stars, contributors, Discord, license
```

**Template:**
```markdown
## Community

- GitHub: [you/repo](https://github.com/you/repo)
- License: MIT
- Built by: [@author](https://github.com/author)
```

**For larger projects add:**
- Discord/Slack invite link
- Contributing guide link
- Star history graph

---

## Pattern 8: FAQ

```
Placement: After Commands, before Community
Count:    3-5 questions
Format:   Question as heading, 1-2 sentence answer
```

**Common questions to answer:**
1. "Do I need a server / database / Docker?"
2. "Is it free / open source?"
3. "How does it compare to X?"
4. "What platforms does it support?"

**Template:**
```markdown
## FAQ

### Do I need a server?
No. Everything runs locally as a single binary.

### Is it free?
Yes. MIT licensed. No paid plans, no limits.
```

---

## Pattern 9: Minimal Text

```
Principle: Every word must earn its place
Rule:      If it can be cut without losing essential info, cut it
Target:    README ≤ 200 lines total
```

**Before (verbose):**
> "BOI CLI is a powerful and flexible command-line interface tool that provides users with the ability to interact with AI agents directly from their terminal. It has been designed from the ground up to be fast, efficient, and easy to use, with a focus on developer experience and productivity."

**After (minimal):**
> "AI agent runtime in your terminal. One binary, no server."

**Rules:**
- Never write a paragraph where a bullet works.
- Never write a sentence where a phrase works.
- Never write a word where nothing works.

---

## Pattern 10: Footer with Credits

```
Placement: Last 2-3 lines of README
Contents: Author, team, license, year
```

**Template:**
```markdown
<p align="center">
  <b>Tool Name</b> — Built by <b>Author</b> &amp; the <b>Team Name</b><br>
  <sub>Licensed under MIT. &copy; 2026</sub>
</p>
```

---

## Complete README Template

```markdown
[Badges — 1-2 lines]

[One-liner description — 1 line]

[Install command — 1 line]

[Logo or Visual — 8-12 lines]

---

## Quick Start

[3 copy-paste commands]

---

## Features

| | |
|---|---|
| Icon | **Feature** — Description |
| Icon | **Feature** — Description |

---

## Personas / Components

[Compact table]

---

## Commands

[Full command table]

---

## Screenshot

[ASCII art mockup or link to image]

---

## FAQ

### Question 1?
Answer.

### Question 2?
Answer.

---

## Community

[GitHub link] [License badge]

---

[Footer credits]
```

---

## Validation Checklist

Before publishing, verify:

- [ ] Install command appears within first 20 lines
- [ ] Visual element visible without scrolling
- [ ] One-liner description fits on one line at 80-char width
- [ ] Badges < 6, all functional
- [ ] Features as bullet list, not prose
- [ ] Quick Start = 3-5 commands, all copy-paste ready
- [ ] Command table complete and tested
- [ ] FAQ answers top 3 user questions
- [ ] Total README ≤ 200 lines
- [ ] Every section scannable in < 3 seconds

---

*Skill created: 31 July 2026 — Kampun (คำปัน) — BOI Family*
