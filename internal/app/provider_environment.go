package app

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"

	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

// ConfigureProviderProfileReferences gives task manifests a credential-free
// pointer to the exact qualification profile used by each Router candidate.
func ConfigureProviderProfileReferences(service *agent.Service, workspaceRoot, boiDir string, qualified []llmfactory.ConfiguredProvider) {
	if service == nil {
		return
	}
	for _, item := range qualified {
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		_, path, loadErr := LoadProfile(boiDir, target)
		if loadErr != nil {
			path = coreblock.CapabilityProfilePath(boiDir, target)
		}
		reference, err := filepath.Rel(workspaceRoot, path)
		if err != nil {
			reference = path
		}
		service.SetProviderProfileReference(item.Name, item.Model, filepath.ToSlash(reference))
	}
}

// verifiedCompensations are the compensations this build has proven. The
// approval floor is on the list because the Runtime's enforcement tests say
// so (agent.ApprovalFloorEnforced); Core only believes it through here.
func verifiedCompensations() coreblock.VerifiedCompensations {
	return coreblock.VerifiedCompensations{ApprovalFloor: agent.ApprovalFloorEnforced}
}

// RecipeForProfile composes the Recipe of a profile just measured, with the
// compensations this build has proven.
func RecipeForProfile(profile *coreblock.CapabilityProfile) coreblock.Recipe {
	return coreblock.ComposeRecipe(profile, verifiedCompensations())
}

// ProviderRecipe composes the Recipe of one configured Provider from its
// measured profile. A Provider with no valid profile gets the Recipe that
// has nothing verified.
func ProviderRecipe(boiDir string, item llmfactory.ConfiguredProvider) coreblock.Recipe {
	target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
	profile, _, err := LoadProfile(boiDir, target)
	if err != nil {
		profile = nil
	}
	return coreblock.ComposeRecipe(profile, verifiedCompensations())
}

// ProviderEnvironment sets the Runtime from the Recipes of every Provider in
// the Router. Failover can select any of them, so a capability is on only
// when every candidate's Recipe has it, and the approval floor is on when
// any candidate needs it: the Policy Floor is shared and takes the strictest
// (ADR-006, D3). A Recipe per endpoint inside the Router is a later step.
func ProviderEnvironment(boiDir string, configured []llmfactory.ConfiguredProvider) coreblock.AgentEnvironment {
	qualified := QualifiedProviders(boiDir, configured)
	if len(qualified) == 0 {
		return coreblock.AgentEnvironment{}
	}
	combined := coreblock.AgentEnvironment{Completion: true, ToolCalling: true, NativeTools: true, SkillCalling: true}
	for _, item := range qualified {
		env := ProviderRecipe(boiDir, item).Environment()
		combined.Completion = combined.Completion && env.Completion
		combined.ToolCalling = combined.ToolCalling && env.ToolCalling
		combined.NativeTools = combined.NativeTools && env.NativeTools
		combined.SkillCalling = combined.SkillCalling && env.SkillCalling
		combined.ApprovalFloor = combined.ApprovalFloor || env.ApprovalFloor
		if combined.ContextBytes == 0 || env.ContextBytes < combined.ContextBytes {
			combined.ContextBytes = env.ContextBytes
		}
	}
	// The floor guards Tool calls; with none offered there is nothing to guard.
	combined.ApprovalFloor = combined.ApprovalFloor && combined.ToolCalling
	return combined
}

// RecipeVersion names the Recipes in force for a set of Providers: the one
// Recipe's version, or a hash over all of them in Router order.
func RecipeVersion(boiDir string, qualified []llmfactory.ConfiguredProvider) string {
	switch len(qualified) {
	case 0:
		return ""
	case 1:
		return ProviderRecipe(boiDir, qualified[0]).Version
	}
	hash := sha256.New()
	for _, item := range qualified {
		hash.Write([]byte(ProviderRecipe(boiDir, item).Version + "\x00"))
	}
	return "multi-" + hex.EncodeToString(hash.Sum(nil)[:8])
}

// QualifiedProviders removes candidates without a valid profile and candidates
// whose basic completion probe failed: the two cases ADR-006 keeps out of the
// Router. They can still be re-qualified directly through the Provider
// transport.
func QualifiedProviders(boiDir string, configured []llmfactory.ConfiguredProvider) []llmfactory.ConfiguredProvider {
	qualified := make([]llmfactory.ConfiguredProvider, 0, len(configured))
	for _, item := range configured {
		if ProviderRecipe(boiDir, item).Usable() {
			qualified = append(qualified, item)
		}
	}
	return qualified
}
