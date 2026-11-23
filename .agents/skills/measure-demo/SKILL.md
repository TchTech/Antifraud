---
name: measure-demo
description: Collect or compare Antifraud demo effectiveness and latency evidence for project defence or mentor review.
---

1. Read `docs/evaluation.md` and choose the claims and denominators before measuring. Record the commit, policy, synthetic dataset and seed, time boundary, configuration and exact command. For throughput and latency also read `docs/load/README.md` and record the achieved per-stream rates.
2. Run on the isolated stand. Record both the decision and actual effect for effectiveness measurements. Retain raw counts, partial failures and measured environment. If only the fixed regression cases ran, publish their individual outcomes.
3. Link raw evidence from `docs/verification.md`. Report which acceptance target each run covers and which prerequisite it leaves open. Done means a second teammate can reproduce every figure from the recorded inputs.
