package app

import (
	"testing"
	"time"

	coreblock "github.com/boi-family/boi-cli/internal/block/core"
	llmfactory "github.com/boi-family/boi-cli/internal/provider/factory"
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
		profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Unix(1, 0), Results: []coreblock.CapabilityResult{
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
