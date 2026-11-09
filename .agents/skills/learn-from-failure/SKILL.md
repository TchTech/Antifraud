---
name: learn-from-failure
description: Turn an investigated failure into a minimal regression, contract correction or instruction update for team review.
---

Capture observed versus expected behavior, original requirement ID, sanitized synthetic input, environment and reproducible command. Separate evidence from hypotheses. Inspect the narrow responsible boundary before editing.

Choose the smallest durable correction: regression test for behavior, OpenAPI/SQL constraint for an ambiguous boundary, or AGENTS/skill clarification for a demonstrated workflow error. Reproduce the failure first, apply the fix, then rerun the failing check and relevant controls. Regenerate code when its source changed.

Record the lesson and proof in the PR using .github/pull_request_template.md. Human review accepts requirement/policy changes; a failure does not authorize production access, new data collection or silent policy relaxation. Done means reproduction, correction and regression evidence are reviewable. Synthetic data only; no external dependencies.
