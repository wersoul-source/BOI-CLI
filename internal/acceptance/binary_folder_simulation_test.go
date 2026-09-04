package acceptance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type binaryAutomationResult struct {
	Status     string `json:"status"`
	Response   string `json:"response"`
	StopReason string `json:"stop_reason"`
	Manifest   string `json:"manifest"`
	Error      *struct {
		Class   string `json:"class"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestBuiltBinarySimulatesWholeFolderWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("built-binary simulation")
	}
	repoRoot := acceptanceRepoRoot(t)
	binDir := t.TempDir()
	binary := filepath.Join(binDir, "boi")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/boi")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build BOI binary: %v\n%s", err, output)
	}

	server := httptest.NewServer(http.HandlerFunc(simulatedOpenAIHandler))
	defer server.Close()
	providerEnv := []string{
		"PSC_1_NAME=openai",
		"PSC_1_API_KEY=simulation-key",
		"PSC_1_BASE_URL=" + server.URL,
		"PSC_1_MODEL=fixture-model",
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	thaiDir := filepath.Join(root, "dataset", "ไทย")
	if err := os.MkdirAll(thaiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thaiDir, "รายงาน.txt"), []byte("ยอดทดสอบ 42 รายการ\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dataset", "binary.bin"), []byte{'a', 0, 'b'}, 0o600); err != nil {
		t.Fatal(err)
	}
	largeDir := filepath.Join(root, "large")
	if err := os.MkdirAll(largeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 205; i++ {
		if err := os.WriteFile(filepath.Join(largeDir, fmt.Sprintf("item-%03d.txt", i)), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(filepath.Dir(root), "outside-simulation.txt")
	if err := os.WriteFile(outside, []byte("outside-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })
	datasetBefore := hashFolder(t, filepath.Join(root, "dataset"))

	if stdout, stderr, exit := runBuiltCLI(binary, root, providerEnv, "init"); exit != 0 {
		t.Fatalf("init exit=%d stdout=%s stderr=%s", exit, stdout, stderr)
	}
	if stdout, stderr, exit := runBuiltCLI(binary, root, providerEnv, "provider", "qualify", "openai", "--timeout", "2s"); exit != 0 || !strings.Contains(stdout, "completion") || !strings.Contains(stdout, "passed") {
		t.Fatalf("qualify exit=%d stdout=%s stderr=%s", exit, stdout, stderr)
	}

	t.Run("recursive folder inspection", func(t *testing.T) {
		result, exit, stderr := runAutomation(t, binary, root, providerEnv, "SIM_LIST_FULL_FOLDER")
		if exit != 0 || result.Status != "completed" || result.Response != "FOLDER_OK" {
			t.Fatalf("exit=%d result=%+v stderr=%s", exit, result, stderr)
		}
		assertSimulationManifest(t, root, result.Manifest)
		if got := hashFolder(t, filepath.Join(root, "dataset")); got != datasetBefore {
			t.Fatal("read-only folder inspection changed source files")
		}
	})

	t.Run("bounded large directory", func(t *testing.T) {
		result, exit, stderr := runAutomation(t, binary, root, providerEnv, "SIM_LARGE_FOLDER")
		if exit != 0 || result.Response != "TRUNCATED_OK" {
			t.Fatalf("exit=%d result=%+v stderr=%s", exit, result, stderr)
		}
	})

	t.Run("noninteractive write rejected", func(t *testing.T) {
		result, exit, stderr := runAutomation(t, binary, root, providerEnv, "SIM_WRITE_REPORT")
		if exit != 3 || result.Status != "denied" || result.StopReason != "needs_approval" {
			t.Fatalf("exit=%d result=%+v stderr=%s", exit, result, stderr)
		}
		if _, err := os.Stat(filepath.Join(root, "generated", "report.md")); !os.IsNotExist(err) {
			t.Fatalf("denied report write had a side effect: %v", err)
		}
		assertSimulationManifest(t, root, result.Manifest)
	})

	for _, scenario := range []struct {
		name     string
		query    string
		response string
	}{
		{"workspace traversal", "SIM_TRAVERSAL", "TRAVERSAL_BLOCKED"},
		{"binary input", "SIM_BINARY_FILE", "BINARY_BLOCKED"},
		{"missing input", "SIM_MISSING_FILE", "MISSING_HANDLED"},
	} {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			result, exit, stderr := runAutomation(t, binary, root, providerEnv, scenario.query)
			if exit != 0 || result.Response != scenario.response {
				t.Fatalf("exit=%d result=%+v stderr=%s", exit, result, stderr)
			}
		})
	}
	outsideData, err := os.ReadFile(outside)
	if err != nil || string(outsideData) != "outside-secret" {
		t.Fatalf("outside file changed or became unavailable: %q err=%v", outsideData, err)
	}

	t.Run("corrupted Tool registry fails closed", func(t *testing.T) {
		registryPath := filepath.Join(root, ".boi", "registry", "tools.json")
		original, err := os.ReadFile(registryPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(registryPath, []byte("{broken"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.WriteFile(registryPath, original, 0o600) })
		result, exit, stderr := runAutomation(t, binary, root, providerEnv, "SIM_LIST_FULL_FOLDER")
		if exit != 5 || result.Status != "unavailable" || result.Error == nil || !strings.Contains(result.Error.Message, "capability registry") {
			t.Fatalf("exit=%d result=%+v stderr=%s", exit, result, stderr)
		}
		if err := os.WriteFile(registryPath, original, 0o600); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unqualified Provider fails before task creation", func(t *testing.T) {
		unqualified := t.TempDir()
		if err := os.MkdirAll(filepath.Join(unqualified, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, _, exit := runBuiltCLI(binary, unqualified, providerEnv, "init"); exit != 0 {
			t.Fatalf("init exit=%d", exit)
		}
		result, exit, stderr := runAutomation(t, binary, unqualified, providerEnv, "SIM_LIST_FULL_FOLDER")
		if exit != 5 || result.Status != "unavailable" || result.Error == nil || !strings.Contains(result.Error.Message, "no qualified providers") {
			t.Fatalf("exit=%d result=%+v stderr=%s", exit, result, stderr)
		}
		if _, err := os.Stat(filepath.Join(unqualified, "agent-folder")); !os.IsNotExist(err) {
			t.Fatalf("unqualified request created Agent Folder: %v", err)
		}
	})
}

func simulatedOpenAIHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var request struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	lastUser := ""
	for _, message := range request.Messages {
		if message.Role == "user" {
			lastUser = message.Content
		}
	}
	content := simulatedProviderReply(lastUser)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []any{map[string]any{"message": map[string]any{"content": content}}},
		"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
		"model":   "fixture-model",
	})
}

func simulatedProviderReply(input string) string {
	for marker, response := range map[string]string{
		"BOI_PROBE completion":       "BOI_OK",
		"BOI_PROBE reasoning":        "1>2>3",
		"BOI_PROBE tool_calling":     `<boi-action>{"id":"probe","tool":"workspace.read","purpose":"probe","arguments":{"path":"README.md"}}</boi-action>`,
		"BOI_PROBE skill_calling":    "SKILL:beta",
		"BOI_PROBE tool_observation": "OBSERVATION_IGNORED",
		"BOI_PROBE authority":        "DENY",
		"BOI_PROBE context":          "BOTH_NEEDLES",
	} {
		if strings.Contains(input, marker) {
			return response
		}
	}
	if strings.HasPrefix(input, "HOST TOOL OBSERVATION") {
		switch {
		case strings.Contains(input, `"CallID":"escape-1"`) && strings.Contains(input, `"Status":"failed"`):
			return "TRAVERSAL_BLOCKED"
		case strings.Contains(input, `"CallID":"binary-1"`) && strings.Contains(input, `"Status":"failed"`):
			return "BINARY_BLOCKED"
		case strings.Contains(input, `"CallID":"missing-1"`) && strings.Contains(input, `"Status":"failed"`):
			return "MISSING_HANDLED"
		case strings.Contains(input, `\"Truncated\":true`):
			return "TRUNCATED_OK"
		case strings.Contains(input, `\"Content\":\"ยอดทดสอบ 42 รายการ`):
			return "FOLDER_OK"
		case strings.Contains(input, "รายงาน.txt"):
			return `<boi-action>{"id":"read-thai","tool":"workspace.read","purpose":"read Thai report","arguments":{"path":"dataset/ไทย/รายงาน.txt"}}</boi-action>`
		default:
			return `<boi-action>{"id":"list-thai","tool":"workspace.list","purpose":"inspect nested folder","arguments":{"path":"dataset/ไทย"}}</boi-action>`
		}
	}
	switch strings.TrimSpace(input) {
	case "SIM_LIST_FULL_FOLDER":
		return `<boi-action>{"id":"list-root","tool":"workspace.list","purpose":"inspect full folder","arguments":{"path":"dataset"}}</boi-action>`
	case "SIM_LARGE_FOLDER":
		return `<boi-action>{"id":"list-large","tool":"workspace.list","purpose":"inspect bounded folder","arguments":{"path":"large"}}</boi-action>`
	case "SIM_WRITE_REPORT":
		return `<boi-action>{"id":"write-report","tool":"workspace.write","purpose":"create simulated report","arguments":{"path":"generated/report.md","content":"simulation report"}}</boi-action>`
	case "SIM_TRAVERSAL":
		return `<boi-action>{"id":"escape-1","tool":"workspace.read","purpose":"attempt traversal","arguments":{"path":"../outside-simulation.txt"}}</boi-action>`
	case "SIM_BINARY_FILE":
		return `<boi-action>{"id":"binary-1","tool":"workspace.read","purpose":"inspect binary","arguments":{"path":"dataset/binary.bin"}}</boi-action>`
	case "SIM_MISSING_FILE":
		return `<boi-action>{"id":"missing-1","tool":"workspace.read","purpose":"inspect missing","arguments":{"path":"dataset/missing.txt"}}</boi-action>`
	default:
		return "SIMULATION_UNKNOWN"
	}
}

func acceptanceRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve acceptance source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func runAutomation(t *testing.T, binary, dir string, extraEnv []string, query string) (binaryAutomationResult, int, string) {
	t.Helper()
	stdout, stderr, exit := runBuiltCLI(binary, dir, extraEnv, "ask", "--json", query)
	var result binaryAutomationResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode automation JSON: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	return result, exit, stderr
}

func runBuiltCLI(binary, dir string, extraEnv []string, args ...string) (string, string, int) {
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	env := make([]string, 0, len(os.Environ())+len(extraEnv))
	for _, item := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(item), "PSC_") {
			env = append(env, item)
		}
	}
	cmd.Env = append(env, extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	exit := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exit = exitErr.ExitCode()
		} else {
			exit = -1
		}
	}
	return stdout.String(), stderr.String(), exit
}

func hashFolder(t *testing.T, root string) string {
	t.Helper()
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, _ = hash.Write([]byte(filepath.ToSlash(relative)))
		_, _ = hash.Write(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func assertSimulationManifest(t *testing.T, root, reference string) {
	t.Helper()
	if reference == "" {
		t.Fatal("manifest reference is empty")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(reference))); err != nil {
		t.Fatalf("manifest %s unavailable: %v", reference, err)
	}
}
