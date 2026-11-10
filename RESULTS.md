# CredDrift — measured results

Recorded 2026-11-10T15:56:06.302729+00:00. Tested source: [`da45283`](https://github.com/theprideofwolves/creddrift/commit/da45283799237a0dca91aa51ccd73aacd1ede3f4). Environment: go version go1.25.5 windows/amd64. These are local measurements, not production impact or third-party certification.

## Verification

| Gate | Result |
|---|---|
| Race-enabled tests | 43 top-level passed; 2 skipped |
| Statement coverage | 71.9% |
| go vet | PASS |
| Root executable build | PASS |
| Packaged HTTP/setup/demo/export/restart smoke | PASS |
| Synthetic evaluation | Completed; limitations reported below |

Detailed [test outcomes](results/verification.json) include table-driven subtests separately. Test coverage measures executed statements, not detection accuracy. Packaged subprocess execution is outside this coverage number.

## Detection baseline — synthetic-v1

| Metric | Result |
|---|---|
| Labeled cases | 20 (10 positive, 10 negative/decoy) |
| True positives / false positives / false negatives | 8 / 3 / 2 |
| Precision | 72.73% |
| Recall | 80.00% |

This is a small authored baseline, not a representative corpus. Scores compare exact extracted identities per case. A partial extraction counts as both a false positive and a missed expected value. No candidate validity is checked. The baseline includes deliberate blind spots rather than only examples matching the regexes. Known misses: quoted JSON keys and whitespace-containing quoted values. False positives include placeholders, suffix key names and a truncated quoted value. These results have not been used to tune detection for this report.

Corpus SHA-256: `c04143bb0b139f8fa7f101a3dd2d2446395e4dd21c4c2bb21b770393be1d0bfe`. [Per-case counts and timings](results/evaluation.json); [corpus and measurement code](cmd/evaluate/main.go).

## Scanner-only performance

- Workload: 1,000 generated files, 62.5 MiB total, 100 candidate occurrences.
- Warm-up: one scan; measurement: five sequential scans.
- Trial milliseconds: 7102.087, 6520.556, 6486.448, 6641.741, 6754.800.
- Median: **6641.741 ms**, **9.41 MiB/s**.

Filesystem cache was not flushed. Fixture generation, SQLite/history writes, UI, process startup and peak memory are excluded. Hardware is not standardized. Do not describe this as end-to-end application latency, guaranteed throughput, or cross-machine superiority. The 50,000-file test verifies a resource limit, not a performance claim.

## Functionality matrix

| Workflow | Automated evidence | Status / limit |
|---|---|---|
| Welcome, demo, folder setup and manual path | onboarding tests; packaged smoke | PASS |
| Native picker failure/cancellation fallback | injected picker test | PASS; real OS dialog manual |
| Setup never scans | onboarding/UI tests; smoke | PASS |
| Supported patterns, boundaries, CRLF locations | scanner tests | PASS for assertions; measured detection blind spots above |
| Render/export/database redaction | UI and timeline tests; smoke | PASS on synthetic fixtures |
| Repeated identities; ignore/restore across locations | store lifecycle/reconciliation tests | PASS |
| Removed/reappearing locations; scope retention | reconciliation/timeline tests | PASS |
| Failed/cancelled scans preserve evidence | store/UI tests; timeline smoke | PASS |
| Exclusions, binary files, out-of-scope coverage | scanner/store/timeline tests | PASS |
| Symlink exclusion and symlink-ancestor retention | dedicated scanner/store tests | SKIPPED locally: Windows privilege unavailable |
| Empty/retired export | UI tests; packaged smoke | PASS |
| Manual rotation wording/state | legacy/manual-action tests; smoke | PASS; not provider-verified |
| Restart and guide restart persistence | packaged smoke; timeline reopen test | PASS |
| Timeline states, identity filter, transaction rollback | timeline tests; smoke | PASS |
| Demo cannot hand off to Jira, including duplicate identities | UI tests; smoke | PASS; no live requests |
| Jira uncertain delivery, duplicate-create protection, legacy origin aliases | fake TLS server/in-memory transport tests | PASS; mocked integration only |
| Theme persistence, guarded POST, safe return paths | theme/UI tests; smoke | PASS; appearance/accessibility manual |
| Oversized file/line, 50,001 files, >100,000 matches | scanner edge tests | PASS |

Manual checks are in [TESTING.md](TESTING.md). Native dialogs, screen readers, narrow/zoomed layout, sustained load and real Jira permissions/custom fields are not validated by these results. No real secrets, private source folders, paid APIs or provider-validation endpoints were used.

## Reproduce and inspect

```sh
python .github/scripts/verify.py
```

Requires Go 1.25.5+, a race-compatible C compiler, Python 3 and Git. [Full method](results/README.md). Reports contain metadata only; private launcher URLs, local data paths and raw candidate values are not included. The tested source revision predates the documentation/results commit; measurement code and application code are identified by that revision.

[GitHub Actions](https://github.com/theprideofwolves/creddrift/actions/workflows/go.yml) repeats Windows/Linux checks and publishes synthetic evaluation artifacts. Check its actual outcome separately; this table describes the recorded local run.

An honest resume bullet: “Built a Go/SQLite credential evidence tracker with transactional rescans and uncertain-delivery-safe Jira handoff; validated with 43 passing race-enabled tests and a reproducible 1,000-file synthetic scan benchmark.” Be prepared to explain the detection misses and measurement boundaries.
