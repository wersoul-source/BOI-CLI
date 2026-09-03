package acceptance_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
	"github.com/boi-family/boi-cli/internal/block/agentfolder"
	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
	"github.com/boi-family/boi-cli/internal/workspace"
)

type scriptedProvider struct {
	responses []string
	errors    []error
	dynamic   func(int, llm.CompletionRequest) (string, error)
	calls     int
}

func (p *scriptedProvider) Name() string { return "acceptance" }
func (p *scriptedProvider) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	index := p.calls
	p.calls++
	if p.dynamic != nil {
		content, err := p.dynamic(index, request)
		if err != nil {
			return nil, err
		}
		return &llm.CompletionResponse{Content: content, Provider: p.Name(), Model: "fixture-model"}, nil
	}
	if index < len(p.errors) && p.errors[index] != nil {
		return nil, p.errors[index]
	}
	if index >= len(p.responses) {
		return nil, errors.New("acceptance fixture exhausted")
	}
	return &llm.CompletionResponse{Content: p.responses[index], Provider: p.Name(), Model: "fixture-model"}, nil
}
func (p *scriptedProvider) Stream(context.Context, llm.CompletionRequest) (<-chan llm.Token, error) {
	return nil, nil
}

type blockingProvider struct{ started chan struct{} }

func (p *blockingProvider) Name() string { return "blocking" }
func (p *blockingProvider) Complete(ctx context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
	select {
	case <-p.started:
	default:
		close(p.started)
	}
	<-ctx.Done()
	return nil, ctx.Err()
}
func (p *blockingProvider) Stream(context.Context, llm.CompletionRequest) (<-chan llm.Token, error) {
	return nil, nil
}

func newService(t *testing.T, provider llm.Provider) (*agent.Service, string) {
	t.Helper()
	root := t.TempDir()
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := agentfolder.NewStore(filepath.Join(root, "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	service := agent.NewService(persona.CorePersona(), llm.NewRouter([]llm.Provider{provider}), nil, sandbox)
	service.SetTaskRecorder(store)
	service.SetProviderProfileReference(provider.Name(), "fixture-model", ".boi/provider-profiles/fixture.json")
	return service, root
}

func runInteractive(t *testing.T, service *agent.Service, task string, decision agent.ApprovalState) (*agent.AgentResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var result *agent.AgentResult
	var runErr error
	for event := range service.Start(ctx, task) {
		if event.Approval != nil {
			event.Approval.Decisions <- agent.ApprovalDecision{RequestID: event.Approval.Request.ID, State: decision, DecidedAt: time.Now()}
		}
		if event.Result != nil || event.Err != nil {
			result, runErr = event.Result, event.Err
		}
	}
	return result, runErr
}

func TestWork1TaskAcceptance(t *testing.T) {
	t.Run("explanation", func(t *testing.T) {
		service, root := newService(t, &scriptedProvider{responses: []string{"bounded explanation"}})
		result, err := service.Run(context.Background(), "explain the runtime")
		if err != nil || result.Response != "bounded explanation" || result.StopReason != agent.StopCompleted {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		assertManifestExists(t, root, result.Manifest)
	})

	t.Run("repository inspection", func(t *testing.T) {
		provider := &scriptedProvider{responses: []string{`<boi-action>{"id":"read-1","tool":"workspace.read","purpose":"inspect README","arguments":{"path":"README.md"}}</boi-action>`, "repository inspected"}}
		service, root := newService(t, provider)
		if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
		result, err := service.Run(context.Background(), "inspect repository")
		if err != nil || result.StopReason != agent.StopCompleted || result.Usage.ToolCalls != 1 {
			t.Fatalf("result=%#v err=%v", result, err)
		}
	})

	t.Run("report creation", func(t *testing.T) {
		outputPattern := regexp.MustCompile(`Standalone user deliverables belong under "([^"]+)"`)
		provider := &scriptedProvider{dynamic: func(index int, request llm.CompletionRequest) (string, error) {
			if index > 0 {
				return "report created", nil
			}
			match := outputPattern.FindStringSubmatch(request.Messages[0].Content)
			if len(match) != 2 {
				return "", errors.New("Agent Folder output scope missing")
			}
			payload, _ := json.Marshal(map[string]any{"id": "report-1", "tool": "workspace.write", "purpose": "create report", "arguments": map[string]any{"path": filepath.ToSlash(filepath.Join(match[1], "report.md")), "content": "verified report"}})
			return "<boi-action>" + string(payload) + "</boi-action>", nil
		}}
		service, root := newService(t, provider)
		result, err := runInteractive(t, service, "create report", agent.ApprovalApproved)
		if err != nil || result == nil || len(result.Artifacts) != 1 || !strings.HasSuffix(result.Artifacts[0].Path, "/report.md") {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		assertManifestExists(t, root, result.Manifest)
	})

	t.Run("approved write", func(t *testing.T) {
		provider := &scriptedProvider{responses: []string{`<boi-action>{"id":"write-1","tool":"workspace.write","purpose":"save note","arguments":{"path":"approved.txt","content":"approved"}}</boi-action>`, "write verified"}}
		service, root := newService(t, provider)
		result, err := runInteractive(t, service, "write approved note", agent.ApprovalApproved)
		content, readErr := os.ReadFile(filepath.Join(root, "approved.txt"))
		if err != nil || readErr != nil || result.StopReason != agent.StopCompleted || string(content) != "approved" {
			t.Fatalf("result=%#v run=%v read=%v content=%q", result, err, readErr, content)
		}
	})

	t.Run("rejection", func(t *testing.T) {
		provider := &scriptedProvider{responses: []string{`<boi-action>{"id":"reject-1","tool":"workspace.write","purpose":"save denied note","arguments":{"path":"denied.txt","content":"denied"}}</boi-action>`}}
		service, root := newService(t, provider)
		result, err := runInteractive(t, service, "reject this write", agent.ApprovalRejected)
		if err == nil || result == nil || result.StopReason != agent.StopRejected {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		if _, statErr := os.Stat(filepath.Join(root, "denied.txt")); !os.IsNotExist(statErr) {
			t.Fatalf("rejected write executed: %v", statErr)
		}
		assertManifestExists(t, root, result.Manifest)
	})

	t.Run("provider failure", func(t *testing.T) {
		failure := &llm.ProviderError{Provider: "acceptance", Class: llm.ErrorAuth, Message: "fixture auth failure"}
		service, root := newService(t, &scriptedProvider{errors: []error{failure}})
		result, err := service.Run(context.Background(), "provider failure")
		if err == nil || result == nil || result.StopReason != agent.StopProviderFailed {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		assertManifestExists(t, root, result.Manifest)
	})

	t.Run("recovery", func(t *testing.T) {
		service, _ := newService(t, &scriptedProvider{responses: []string{"", "recovered safely"}})
		result, err := service.Run(context.Background(), "recover invalid decision")
		if err != nil || result.Response != "recovered safely" || result.Plan.Revision != 2 {
			t.Fatalf("result=%#v err=%v", result, err)
		}
	})
}

func TestWork1CancellationAcceptance(t *testing.T) {
	provider := &blockingProvider{started: make(chan struct{})}
	service, root := newService(t, provider)
	ctx, cancel := context.WithCancel(context.Background())
	events := service.Start(ctx, "cancel provider")
	select {
	case <-provider.started:
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("Provider did not start")
	}
	var result *agent.AgentResult
	var runErr error
	for event := range events {
		if event.Result != nil || event.Err != nil {
			result, runErr = event.Result, event.Err
		}
	}
	if !errors.Is(runErr, context.Canceled) || result == nil || result.StopReason != agent.StopCancelled {
		t.Fatalf("result=%#v err=%v", result, runErr)
	}
	assertManifestExists(t, root, result.Manifest)
}

func assertManifestExists(t *testing.T, root, reference string) {
	t.Helper()
	if reference == "" {
		t.Fatal("manifest reference is empty")
	}
	path := filepath.Join(root, filepath.FromSlash(reference))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("manifest %s missing: %v", reference, err)
	}
}
