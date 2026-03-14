---
name: Release tracking
about: Track packaging, installability, smoke checks, and public release notes for a specific Sheaft release
title: "[REL] vX.Y.Z release tracking"
labels: ["area: ci", "area: docs", "type: task"]
---

## Release scope

- Version: `vX.Y.Z`
- Channel: technical preview / experimental / stable
- Planned tag date:
- Target release URL:

## Packaging

- [ ] `go build ./cmd/sheaft` works from a clean checkout
- [ ] checked-in sample `run` smoke path passes
- [ ] checked-in sample `serve` config starts successfully
- [ ] reproducible release packaging path is green
- [ ] release archives/checksums/SBOMs are attached

## First-user surface

- [ ] README install path is explicit
- [ ] Quickstart is copy-paste friendly
- [ ] release status is clearly stated
- [ ] compatibility with supported upstream contracts is documented
- [ ] changelog / release notes are ready

## Out of scope

- Do not close broader roadmap epics from this issue alone.
- Use linked implementation issues for product capability changes.

## Links

- Release notes source:
- Release workflow run:
- Follow-up milestone:
