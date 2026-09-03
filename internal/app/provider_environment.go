package app

import (
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/agent"
	coreblock "github.com/boi-family/boi-cli/internal/block/core"
	llmfactory "github.com/boi-family/boi-cli/internal/provider/factory"
)

// ConfigureProviderProfileReferences gives task manifests a credential-free
// pointer to the exact qualification profile used by each Router candidate.
func ConfigureProviderProfileReferences(service *agent.Service, workspaceRoot, boiDir string, qualified []llmfactory.ConfiguredProvider) {
	if service == nil {
		return
	}
	for _, item := range qualified {
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		path := coreblock.CapabilityProfilePath(boiDir, target)
		reference, err := filepath.Rel(workspaceRoot, path)
		if err != nil {
			reference = path
		}
		service.SetProviderProfileReference(item.Name, item.Model, filepath.ToSlash(reference))
	}
}

// ProviderEnvironment composes the intersection of qualified capabilities.
// Router failover can select any configured Provider, so Tool/Skill Calling is
// exposed only when every candidate has a valid passing profile.
func ProviderEnvironment(boiDir string, configured []llmfactory.ConfiguredProvider) coreblock.AgentEnvironment {
	qualified := QualifiedProviders(boiDir, configured)
	if len(qualified) == 0 {
		return coreblock.AgentEnvironment{}
	}
	combined := coreblock.AgentEnvironment{Completion: true, ToolCalling: true, SkillCalling: true}
	for _, item := range qualified {
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		profile, err := coreblock.LoadCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target))
		if err != nil {
			continue
		}
		env := coreblock.ComposeEnvironment(profile)
		combined.Completion = combined.Completion && env.Completion
		combined.ToolCalling = combined.ToolCalling && env.ToolCalling
		combined.SkillCalling = combined.SkillCalling && env.SkillCalling
		if combined.ContextBytes == 0 || env.ContextBytes < combined.ContextBytes {
			combined.ContextBytes = env.ContextBytes
		}
	}
	return combined
}

// QualifiedProviders removes candidates without a valid profile and candidates
// whose basic completion probe failed. They can still be re-qualified directly
// through the Provider transport, but cannot enter the Agent Router.
func QualifiedProviders(boiDir string, configured []llmfactory.ConfiguredProvider) []llmfactory.ConfiguredProvider {
	qualified := make([]llmfactory.ConfiguredProvider, 0, len(configured))
	for _, item := range configured {
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		profile, err := coreblock.LoadCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target))
		if err == nil && coreblock.ComposeEnvironment(profile).Completion {
			qualified = append(qualified, item)
		}
	}
	return qualified
}
