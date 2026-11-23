---
name: verify-protection
description: Check an Antifraud security-review finding against current defensive invariants and track the proof needed to close it.
---

1. Read `AGENTS.md` and `docs/security-gaps.md`. Locate the applicable requirement and invariant before reading the affected code. Treat attached reviews as evidence and quote only what the team needs to decide.
2. Check the smallest boundary on an isolated synthetic database. Write the expected control outcome before running a test. Assert service state and audit alongside any HTTP result. Preserve the original review ID and record evidence level and the observed outcome.
3. Run the relevant checks from `AGENTS.md`, then update the gap and verification report. A gap closes when its acceptance condition has current proof and the associated legitimate control still passes.
