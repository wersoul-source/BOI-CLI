## Description

What does this PR do?

## Gap · Composition · Evidence (ADR-006)

Every PR answers all three. A PR that does not change what a model can do (documentation, tooling) may answer `N/A` with a reason.

- **Gap** — what limitation was found, and what is the evidence?
- **Composition** — what is added, which Block owns it, and how is it removed?
- **Evidence** — how do before and after differ on correctness, safety, stability and cost, which criteria set in advance did it meet, and is there a regression test?

Capability statuses used: `NATIVE` · `ASSISTED` · `UNVERIFIED` · `BLOCKED`. Never report an `ASSISTED` capability as `NATIVE`, and never claim something works without a test result.

## Related Issue

Fixes # (issue number)

## Checklist

- [ ] `go fmt ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go build ./...` passes
- [ ] Tests pass (`go test ./...` if applicable)
- [ ] README / documentation updated if needed

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Documentation
- [ ] Refactor
- [ ] Other (describe below)
