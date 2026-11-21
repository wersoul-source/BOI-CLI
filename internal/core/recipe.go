package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// CapabilityState is what BOI may do with one capability of one endpoint
// (ADR-006). It is derived from measured evidence, never assumed.
type CapabilityState string

const (
	// StateNative: the model does it alone and passed the test.
	StateNative CapabilityState = "NATIVE"
	// StateAssisted: the model does it with a compensation and passed the
	// test with it. It is used, and always reported as assisted.
	StateAssisted CapabilityState = "ASSISTED"
	// StateUnverified: there is not enough evidence yet. It is not used.
	StateUnverified CapabilityState = "UNVERIFIED"
	// StateBlocked: a safety risk, or nothing that meets the criteria can
	// close the gap. It is not used, and the evidence is kept.
	StateBlocked CapabilityState = "BLOCKED"
)

// Usable reports whether real work may rely on the capability.
func (s CapabilityState) Usable() bool { return s == StateNative || s == StateAssisted }

// Recipe capabilities. They are what the Runtime decides on; a Recipe also
// lists every probed capability under its probe name.
const (
	RecipeCompletion = "completion"
	RecipeTools      = "tools"
	RecipeSkills     = "skills"
	RecipeContext    = "context"
)

// Tool protocols a Recipe can choose.
const (
	ProtocolNone     = "none"
	ProtocolNative   = "native"
	ProtocolEnvelope = "envelope"
	// ProtocolDialectPrefix is followed by the dialect name, as in
	// "dialect:dsml".
	ProtocolDialectPrefix = "dialect:"
)

// Approval floors (D1).
const (
	FloorNone    = "none"
	FloorConfirm = "confirm"
)

// Compensation names used in a Recipe.
const (
	CompensationEnvelope      = "envelope"
	CompensationApprovalFloor = "approval-floor"
)

// RecipeSchemaVersion is the version of the Recipe contract (v0 draft).
const RecipeSchemaVersion = 0

// RecipeCapability is the state of one capability with what it rests on.
type RecipeCapability struct {
	Name  string          `json:"name"`
	State CapabilityState `json:"state"`
	// Compensation names what makes an ASSISTED capability work, or what
	// would make an UNVERIFIED one work once it is proven. Empty for NATIVE.
	Compensation string `json:"compensation,omitempty"`
	// Evidence says why the capability has this state, in the probe's words
	// where there are any.
	Evidence string `json:"evidence,omitempty"`
}

// Recipe is how this Runtime equips one model on one endpoint. Core composes
// it from the measured profile; the Runtime follows it. It belongs to one
// endpoint and one profile and is never shared between Providers (D3).
type Recipe struct {
	SchemaVersion int `json:"schema_version"`
	// Version identifies the content: two Recipes with the same decisions
	// have the same version, and any changed decision changes it.
	Version            string             `json:"version"`
	ProfileFingerprint string             `json:"profile_fingerprint,omitempty"`
	Endpoint           ProviderTarget     `json:"endpoint"`
	Capabilities       []RecipeCapability `json:"capabilities"`
	// ToolProtocol is how Tool calls travel: "native", "envelope",
	// "dialect:<name>", or "none" when no protocol was measured to work.
	ToolProtocol string `json:"tool_protocol"`
	// ApprovalFloor is "confirm" when a safety probe did not pass: every
	// Tool call then needs approval through the Broker (D1).
	ApprovalFloor string `json:"approval_floor"`
	ContextBytes  int    `json:"context_bytes,omitempty"`
}

// VerifiedCompensations says which safety compensations this build has
// proven with enforcement tests. A compensation that is not listed here is
// never switched on: the capability it would carry stays UNVERIFIED (ADR-006,
// "How D1 meets Rule 3").
type VerifiedCompensations struct {
	// ApprovalFloor: every Tool call on a floored route reaches approval, a
	// non-interactive run refuses it, and no retry or Provider switch
	// escapes it.
	ApprovalFloor bool
}

// State returns the state of a capability, UNVERIFIED when the Recipe does
// not name it.
func (r Recipe) State(name string) CapabilityState {
	for _, capability := range r.Capabilities {
		if capability.Name == name {
			return capability.State
		}
	}
	return StateUnverified
}

// Capability returns the named capability and whether the Recipe has it.
func (r Recipe) Capability(name string) (RecipeCapability, bool) {
	for _, capability := range r.Capabilities {
		if capability.Name == name {
			return capability, true
		}
	}
	return RecipeCapability{}, false
}

// Usable reports whether the endpoint may be used at all: exactly when
// completion works. A Provider with no measured profile, or whose completion
// failed, is the only kind kept out of the Router (ADR-006, Exclusion).
func (r Recipe) Usable() bool { return r.State(RecipeCompletion).Usable() }

// ComposeRecipe turns a measured profile into a Recipe. It only reads
// evidence: nothing here grants a right the Policy Floor withholds, and a
// capability that rests on an unproven compensation stays UNVERIFIED.
func ComposeRecipe(profile *CapabilityProfile, verified VerifiedCompensations) Recipe {
	recipe := Recipe{SchemaVersion: RecipeSchemaVersion, ToolProtocol: ProtocolNone, ApprovalFloor: FloorNone}
	if profile == nil || profile.Validate() != nil {
		recipe.Capabilities = []RecipeCapability{
			{Name: RecipeCompletion, State: StateUnverified, Evidence: "no measured profile"},
			{Name: RecipeTools, State: StateUnverified, Evidence: "no measured profile"},
			{Name: RecipeSkills, State: StateUnverified, Evidence: "no measured profile"},
			{Name: RecipeContext, State: StateUnverified, Evidence: "no measured profile"},
		}
		recipe.Version = recipeVersion(recipe)
		return recipe
	}
	recipe.ProfileFingerprint, recipe.Endpoint = profile.Fingerprint, profile.Target

	completion := probeState(profile, "completion")
	completion.Name = RecipeCompletion
	tools := composeTools(profile, completion.State, verified, &recipe)
	skills := composeSkills(profile, completion.State)
	context := probeState(profile, "context")
	context.Name = RecipeContext
	if context.State == StateBlocked {
		// A failed context probe measures a limit, not a danger: the default
		// budget applies.
		context.State = StateUnverified
	}
	if context.State == StateNative {
		for _, result := range profile.Results {
			if result.Capability == "context" {
				recipe.ContextBytes = result.InputBytes
			}
		}
	}
	recipe.Capabilities = []RecipeCapability{completion, tools, skills, context}
	// Every probe is also listed under its own name, so the Recipe carries
	// the whole measurement and nothing is summarised away.
	for _, result := range profile.Results {
		if result.Capability == "completion" || result.Capability == "context" {
			continue
		}
		recipe.Capabilities = append(recipe.Capabilities, probeState(profile, result.Capability))
	}
	recipe.Version = recipeVersion(recipe)
	return recipe
}

// probeState maps one probe result to a state: passed alone is NATIVE, passed
// with help is ASSISTED, a failure with nothing to close it is BLOCKED, and
// anything not measured is UNVERIFIED.
func probeState(profile *CapabilityProfile, capability string) RecipeCapability {
	out := RecipeCapability{Name: capability, State: StateUnverified, Evidence: "not measured"}
	for _, result := range profile.Results {
		if result.Capability != capability {
			continue
		}
		out.Evidence = result.Detail
		switch result.Status {
		case CapabilityPassed:
			out.State = StateNative
			if result.Assisted != "" {
				out.State, out.Compensation = StateAssisted, result.Assisted
			}
		case CapabilityFailed, CapabilityUnsupported:
			out.State = StateBlocked
			if out.Evidence == "" {
				out.Evidence = "probe " + string(result.Status)
			}
		default:
			out.State = StateUnverified
			if out.Evidence == "" {
				out.Evidence = "probe " + string(result.Status)
			}
		}
	}
	return out
}

// composeTools decides the Tool route: which protocol carries the calls, and
// whether the two safety probes let it run freely, under the approval floor,
// or not yet.
func composeTools(profile *CapabilityProfile, completion CapabilityState, verified VerifiedCompensations, recipe *Recipe) RecipeCapability {
	tools := RecipeCapability{Name: RecipeTools}
	if !completion.Usable() {
		tools.State, tools.Evidence = StateUnverified, "completion does not work, so no Tool route was measured"
		if completion == StateBlocked {
			tools.State = StateBlocked
		}
		return tools
	}
	native, envelope := probeState(profile, "native_tool_schema"), probeState(profile, "tool_calling")
	switch {
	case native.State == StateNative:
		recipe.ToolProtocol, tools.State = ProtocolNative, StateNative
	case native.State == StateAssisted:
		// The Connector recovered the call from markup: the dialect is the
		// protocol and the compensation.
		recipe.ToolProtocol = native.Compensation
		tools.State, tools.Compensation = StateAssisted, native.Compensation
		tools.Evidence = joinEvidence("the model writes Tool calls as markup in its text and the Connector recovers them", native.Evidence)
	case envelope.State.Usable():
		// The text envelope is BOI's own protocol, given to the model in
		// the prompt: a compensation for a model with no native tool calls.
		recipe.ToolProtocol = ProtocolEnvelope
		tools.State, tools.Compensation = StateAssisted, CompensationEnvelope
		tools.Evidence = joinEvidence("native tool calls did not work; the model uses BOI's text envelope", native.Evidence)
	case native.State == StateBlocked || envelope.State == StateBlocked:
		tools.State = StateBlocked
		tools.Evidence = joinEvidence("no Tool protocol works", native.Evidence, envelope.Evidence)
		return tools
	default:
		tools.State, tools.Evidence = StateUnverified, "no Tool protocol was measured"
		return tools
	}

	observation, authority := probeState(profile, "tool_observation"), probeState(profile, "authority")
	if observation.State == StateNative && authority.State == StateNative {
		return tools
	}
	if !measured(profile, "tool_observation") || !measured(profile, "authority") {
		// Without the safety measurements there is nothing to decide on.
		tools.State, tools.Compensation = StateUnverified, ""
		tools.Evidence = "the safety probes (tool_observation, authority) were not measured"
		return tools
	}
	// A safety probe did not pass. The route is not cut: it runs under the
	// approval floor (D1), and only once that floor is proven (Rule 3).
	recipe.ApprovalFloor = FloorConfirm
	tools.Compensation = joinCompensation(tools.Compensation, CompensationApprovalFloor)
	tools.Evidence = joinEvidence(strings.TrimPrefix(tools.Evidence+"; a safety probe did not pass", "; "), labelled("tool_observation", observation), labelled("authority", authority))
	if verified.ApprovalFloor {
		tools.State = StateAssisted
	} else {
		tools.State = StateUnverified
		tools.Evidence += "; the approval floor is not verified in this build, so the route stays closed"
	}
	return tools
}

// composeSkills: a Skill is instructions the model chooses to load. It needs
// the model to pick correctly and to refuse a request to bypass approval.
func composeSkills(profile *CapabilityProfile, completion CapabilityState) RecipeCapability {
	skills := probeState(profile, "skill_calling")
	skills.Name = RecipeSkills
	if !completion.Usable() {
		skills.State, skills.Compensation = StateUnverified, ""
		if completion == StateBlocked {
			skills.State = StateBlocked
		}
		skills.Evidence = "completion does not work"
		return skills
	}
	if !skills.State.Usable() {
		return skills
	}
	if authority := probeState(profile, "authority"); authority.State != StateNative {
		// No compensation for this exists yet, so it is not used.
		skills.State, skills.Compensation = StateUnverified, ""
		skills.Evidence = joinEvidence("Skill selection passed but the authority probe did not", authority.Evidence)
	}
	return skills
}

// measured reports whether the profile holds a result for the probe at all.
func measured(profile *CapabilityProfile, capability string) bool {
	for _, result := range profile.Results {
		if result.Capability == capability {
			return true
		}
	}
	return false
}

func labelled(name string, capability RecipeCapability) string {
	if capability.State == StateNative {
		return ""
	}
	return strings.TrimSuffix(name+": "+capability.Evidence, ": ")
}

func joinEvidence(lead string, parts ...string) string {
	out := lead
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out += "; " + part
		}
	}
	return out
}

func joinCompensation(existing, added string) string {
	if existing == "" {
		return added
	}
	return existing + "+" + added
}

// recipeVersion is a short hash of every decision in the Recipe.
func recipeVersion(recipe Recipe) string {
	recipe.Version = ""
	encoded, err := json.Marshal(recipe)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:8])
}
