---
name: add-fraud-scenario
description: Add a synthetic fraud or legitimate-control scenario through ingestion, risk policy and end-to-end service-state assertions.
---

Read AGENTS.md, docs/traceability.md and the protect-call-forwarding spec. Identify original requirement IDs and expected service state before changing rules. Preserve all team edits.

Add only synthetic registrations to core.Fixtures or a JSON source consumed by network-collector -file. Include stable event identity, account, source, schema version and occurrence time; receipt time comes from the server. Specify missing versus negative signals and a legitimate control case.

Keep feature calculation in collector/policy and reaction in policy. If wire semantics change, edit contracts/scripts/contracts.py and run make generate. Add an integration scenario through the gateway, asserting decision evidence/version, outbox delivery, actual service state and execution audit. Include replay or late-arrival assertions when relevant.

Run make check and TEST_DATABASE_URL=<isolated-db> make e2e. Update traceability and the policy assumption/version. Done means the new scenario and existing controls pass; unresolved policy choices go to docs/grill-with-docs.md for team review.
