# Clean Code — Go Edition

> ศึกษาจาก 10 แหล่ง: Go Code Review Comments, Uber Go Style Guide, Effective Go, Go Proverbs, Standard Go Project Layout, Go by Example, 100 Go Mistakes, ardanlabs, Dave Cheney, Mat Ryer
> โดย: คำปัน (Kampun) — BOI Family

---

## 🎯 10 Principles for Go Clean Code

### 1. `gofmt` or Die
```go
// ❌ Never format manually
func foo() { x:=1;y:=2 }

// ✅ Let gofmt handle it
func foo() {
    x := 1
    y := 2
}
```
> `go fmt ./...` on every commit. Zero debate about formatting.

### 2. Short Package Names
```go
// ❌ package boi_cli_agent_runtime
// ✅ package agent
```
> Single word, lowercase, no underscores. Package path provides context.

### 3. Handle Errors Explicitly
```go
// ❌ Ignore errors
data, _ := os.ReadFile(path)

// ✅ Handle or propagate
data, err := os.ReadFile(path)
if err != nil {
    return fmt.Errorf("read config: %w", err)
}
```
> Every error must be handled, wrapped with context, or explicitly ignored with comment.

### 4. No Panic in Libraries
```go
// ❌ func Parse(s string) Value { if s == "" { panic("empty") } }
// ✅ func Parse(s string) (Value, error) { if s == "" { return nil, ErrEmpty } }
```
> Only `main` may call `os.Exit` or `log.Fatal`.

### 5. Interfaces are Small
```go
// ❌ 10 methods
type Storage interface {
    Get(key string) ([]byte, error)
    Set(key string, val []byte) error
    Delete(key string) error
    List() ([]string, error)
    // ... 6 more
}

// ✅ 1-3 methods
type Reader interface { Read([]byte) (int, error) }
type Writer interface { Write([]byte) (int, error) }
```
> Accept interfaces, return structs. Interface at consumer, not producer.

### 6. Zero Value is Useful
```go
// ✅ var mu sync.Mutex (ready to use, no init needed)
// ✅ var buf bytes.Buffer (empty, ready to use)
// ❌ mu := new(sync.Mutex) (unnecessary pointer)
```

### 7. Avoid Naked Returns
```go
// ❌ func split(s string) (x, y string) { x = s[:i]; y = s[i:]; return }
// ✅ func split(s string) (string, string) { return s[:i], s[i:] }
```

### 8. Reduce Nesting — Early Return
```go
// ❌ Deep nesting
func process(data []byte) error {
    if data != nil {
        if len(data) > 0 {
            // ... 3 more levels
        }
    }
    return nil
}

// ✅ Early return
func process(data []byte) error {
    if data == nil { return ErrNilData }
    if len(data) == 0 { return ErrEmptyData }
    // ... flat logic
    return nil
}
```

### 9. Don't Repeat Package Name in Symbols
```go
// ❌ agent.NewAgent(), agent.AgentLoop, agent.AgentConfig
// ✅ agent.New(), agent.Loop, agent.Config
// User sees: agent.New(), agent.Loop — clean and clear
```

### 10. Table-Driven Tests
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        a, b, want int
    }{
        {1, 2, 3},
        {0, 0, 0},
        {-1, 1, 0},
    }
    for _, tt := range tests {
        got := Add(tt.a, tt.b)
        if got != tt.want {
            t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
        }
    }
}
```

---

## 📋 Quick Checklist

```
□ go fmt ./...       — formatting
□ go vet ./...       — suspicious constructs
□ go build ./...     — compilation
□ No panic()         — except in main
□ Errors handled     — every error checked
□ Interfaces small   — ≤3 methods preferred
□ Early returns      — flat, not nested
□ Short names        — no stutter (agent.AgentConfig → agent.Config)
□ Zero-value ready   — var x T (no constructor needed)
□ Tests exist        — table-driven where possible
```

---

## 🔧 Tools

```bash
go fmt ./...          # Format all code
go vet ./...          # Static analysis
go build ./...        # Build check
golangci-lint run     # Comprehensive linting (optional, needs install)
```
