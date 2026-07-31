# README Study — 10 Top CLI Repos

> Analysis of README patterns from 10 most popular CLI tools.
> Used to derive universal README standards.

---

## Repositories Analyzed

| # | Repo | Stars | Key Pattern |
|---|------|-------|-------------|
| 1 | **gh** (cli/cli) | 45.5k | Multi-platform install table + screenshot + SLSA verification |
| 2 | **uv** (astral-sh/uv) | 88.2k | Badges + Highlights bullet list + Features with subsections + FAQ |
| 3 | **ollama** (ollama/ollama) | 177k | Logo image + Download section first + Get Started + Community + Integrations |
| 4 | **bun** (oven-sh/bun) | 75k | Speed claim first + npm install one-liner + Benchmarks |
| 5 | **ripgrep** (BurntSushi/ripgrep) | 50k | One-liner description + Quick install + Feature table |
| 6 | **fzf** (junegunn/fzf) | 65k | GIF demo first + Simple install + Minimal text |
| 7 | **bat** (sharkdp/bat) | 50k | Screenshot first + Syntax highlighting highlight |
| 8 | **fd** (sharkdp/fd) | 35k | Comparison to find + Benchmarks + Simple |
| 9 | **lazygit** (jesseduffield/lazygit) | 55k | GIF demo + Elevator pitch + Simple |
| 10 | **zoxide** (ajeetdsouza/zoxide) | 25k | One-liner + Simple table + Minimal |

---

## 10 Universal Patterns

### 1. Install Command First
Install command (`curl`, `brew`, `npm install`, `go install`) appears within the first 20 lines of every top README. Users who arrive from search results must see how to install immediately.

**Example (bun):**
```bash
npm install -g bun
```

**Example (uv):**
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

### 2. Visual Above Fold
Every high-star repo uses a screenshot, GIF demo, or logo image visible without scrolling. Visuals establish credibility and show what the tool actually looks like — faster than any paragraph of text.

**Example (lazygit):** Animated GIF of full TUI interaction — shows value in 3 seconds.
**Example (bat):** Side-by-side screenshot comparing `bat` vs `cat` output.
**Example (fzf):** GIF of fuzzy-finding files in real time.

### 3. One-Liner Description
A single sentence that answers "what does this do?" The tagline must fit on one line at any screen width. No marketing fluff — pure function.

**Example (ripgrep):** "ripgrep is a line-oriented search tool that recursively searches the current directory for a regex pattern."
**Example (fd):** "fd is a program to find entries in your filesystem."

### 4. Badges
Version, downloads, license, build status, and platform badges displayed as a compact row. Badges signal: "this is real, maintained, and trustworthy."

**Example (uv):**
```
[![PyPI](...)](...) [![License](...)](...) [![Downloads](...)](...)
```

### 5. Features as Bullet List
Features presented as bullet points, not paragraphs. Each bullet is one line (≤80 chars) with an emoji or icon prefix. No nested paragraphs.

**Example pattern:**
```
- Lightning fast — 10-100x faster than alternatives
- Lazy loading — only install what you use
- Drop-in replacement — compatible with existing tooling
```

### 6. Quick Start Section
A "Quick Start" or "Get Started" section with copy-paste-ready commands. 3-5 commands maximum. Each command must work as-is with no edits required (placeholders marked clearly).

**Example:**
```bash
git clone https://github.com/user/repo.git
cd repo
go build -o bin/tool ./cmd/tool
./bin/tool init
```

### 7. Community Section
Stars count, contributor count, Discord/community link, license badge. Signals that the project is alive and welcomes contributions.

**Example (ollama):** "Community" section with Discord invite, Twitter link, and contribution guide.

### 8. FAQ
3-5 common questions with short answers. Handles the "what about X?" and "do I need Y?" questions that every new user has.

**Example pattern:**
```
### Do I need a server?
No. Runs entirely locally as a single binary.

### Is it free?
Yes. Open source under MIT license.
```

### 9. Minimal Text
Short sentences, high information density. No walls of prose. Every word earns its place. The README is skimmed, not read.

**Rule:** If a sentence can be cut without losing essential information, cut it.

### 10. Footer with Credits
Who built it, what license, when. Signals ownership and legal clarity.

**Example:**
```
Built with ❤️ by [Author Name] and [community/team].
Licensed under MIT. © 2026.
```

---

## Pattern Decision Matrix

| Pattern | Priority | Where | Why |
|---------|----------|-------|-----|
| Install first | CRITICAL | Before fold | Users leave if they can't install in 10 seconds |
| Visual above fold | CRITICAL | Before fold | Screenshot > 1000 words |
| One-liner | HIGH | Line 1-3 | Must answer "what is this" instantly |
| Badges | HIGH | Line 1-3 | Trust signal |
| Bullet features | HIGH | After fold | Scannable value proposition |
| Quick Start | HIGH | After features | Actionable next step |
| Community | MEDIUM | Near bottom | Social proof |
| FAQ | MEDIUM | After community | Handle objections |
| Minimal text | CONTINUOUS | Everywhere | Skimmed, not read |
| Footer credits | LOW | Last lines | Legal clarity |

---

## Anti-Patterns (What NOT to do)

| Anti-Pattern | Why It Fails |
|--------------|--------------|
| Long installation docs buried at bottom | Users leave before finding them |
| Walls of prose before any code | Nobody reads paragraphs on GitHub |
| No visual element | Looks abandoned or untrustworthy |
| Feature paragraphs instead of bullets | Can't scan for value |
| "Contributing" as first section | Readers haven't decided to use it yet |
| Too many badges (> 6) | Visual noise, slow page load |
| Generic "Fast, Modern, Simple" without proof | Empty marketing speak |

---

*Study compiled: 31 July 2026 — Kampun (คำปัน)*
