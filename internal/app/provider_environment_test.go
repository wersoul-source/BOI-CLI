package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	coreblock "github.com/boi-family/boi-cli/internal/core"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

func TestProviderEnvironmentFailsClosedAndRequiresEveryProvider(t *testing.T) {
	boiDir := t.TempDir()
	items := []llmfactory.ConfiguredProvider{{Name: "one", Model: "m1", EndpointClass: "official"}, {Name: "two", Model: "m2", EndpointClass: "custom"}}
	if ProviderEnvironment(boiDir, items).ToolCalling {
		t.Fatal("missing profiles must disable Tool Calling")
	}
	if len(QualifiedProviders(boiDir, items)) != 0 {
		t.Fatal("missing profiles must not enter Router")
	}
	for index, item := range items {
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: []coreblock.CapabilityResult{
			{Capability: "completion", Status: coreblock.CapabilityPassed},
			{Capability: "tool_calling", Status: coreblock.CapabilityPassed},
			{Capability: "tool_observation", Status: coreblock.CapabilityPassed},
			{Capability: "authority", Status: coreblock.CapabilityPassed},
			{Capability: "skill_calling", Status: coreblock.CapabilityPassed},
			{Capability: "context", Status: coreblock.CapabilityPassed, InputBytes: 1024 + index},
		}}
		if err := coreblock.SaveCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target), profile); err != nil {
			t.Fatal(err)
		}
		if index == 0 && len(QualifiedProviders(boiDir, items)) != 1 {
			t.Fatal("only profiled Provider should enter Router")
		}
	}
	env := ProviderEnvironment(boiDir, items)
	if !env.ToolCalling || !env.SkillCalling || env.ContextBytes != 1024 {
		t.Fatalf("unexpected environment: %#v", env)
	}
	if len(QualifiedProviders(boiDir, items)) != 2 {
		t.Fatal("both passing Providers should enter Router")
	}
}

func TestQualificationProfileIsSharedThroughGlobalDir(t *testing.T) {
	global := t.TempDir()
	t.Setenv("BOI_CONFIG_DIR", global)
	target := coreblock.ProviderTarget{Provider: "openai", Model: "m", EndpointClass: "custom"}
	profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: []coreblock.CapabilityResult{{Capability: "completion", Status: coreblock.CapabilityPassed}}}
	first, second := filepath.Join(t.TempDir(), ".boi"), filepath.Join(t.TempDir(), ".boi")
	if _, _, err := LoadProfile(second, target); err == nil {
		t.Fatal("profile found before any qualification")
	}
	if _, err := SaveProfile(first, target, profile); err != nil {
		t.Fatal(err)
	}
	_, path, err := LoadProfile(second, target)
	if err != nil || !strings.HasPrefix(path, global) {
		t.Fatalf("second workspace did not reuse the shared profile: path=%q err=%v", path, err)
	}
}

func TestExpiredQualificationKeepsProviderOut(t *testing.T) {
	boiDir := t.TempDir()
	item := llmfactory.ConfiguredProvider{Name: "old", Model: "m", EndpointClass: "custom"}
	target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
	save := func(generated time.Time) {
		profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: generated, Results: []coreblock.CapabilityResult{{Capability: "completion", Status: coreblock.CapabilityPassed}}}
		if err := coreblock.SaveCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target), profile); err != nil {
			t.Fatal(err)
		}
	}
	save(time.Now().Add(-coreblock.ProfileMaxAge - time.Hour))
	if len(QualifiedProviders(boiDir, []llmfactory.ConfiguredProvider{item})) != 0 {
		t.Fatal("an expired profile must not let the Provider into the Router")
	}
	if _, _, err := LoadProfile(boiDir, target); err == nil || !strings.Contains(err.Error(), "boi provider qualify old") {
		t.Fatalf("the error must say how to fix it: %v", err)
	}
	save(time.Now().Add(-coreblock.ProfileMaxAge + time.Hour))
	if len(QualifiedProviders(boiDir, []llmfactory.ConfiguredProvider{item})) != 1 {
		t.Fatal("a profile inside the window must still count")
	}
}
