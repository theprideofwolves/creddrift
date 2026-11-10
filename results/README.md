# Reproduce the results

From the repository root with Go 1.25.5+, a C compiler for the race detector, Python 3 and Git:

```sh
python .github/scripts/verify.py
```

This runs race-enabled functional tests, statement coverage, vet, a root executable build, packaged smoke, and the synthetic evaluation. It writes redacted summaries to results/verification.json and results/evaluation.json. Test output and private launcher URLs are not published. All scanner inputs are generated temporary synthetic folders. No paid APIs, real secrets or provider validation are used; Jira tests use local mocks.

For evaluation only:

```sh
go run ./cmd/evaluate -revision YOUR_COMMIT -out results/evaluation.json
```

The corpus is defined in cmd/evaluate/main.go and versioned as synthetic-v1. Its SHA-256 is included in the report. Each of 20 single-line cases has an independently specified set of expected candidate values. Scoring compares exact candidate identities within the same case, deduplicates repeated detector output, and counts an incorrect extracted value as a false positive plus a missed expected value. Ten cases contain intended candidates; ten are negative/decoy cases. Quoted JSON, whitespace-containing values and placeholders intentionally expose limitations. This tiny authored corpus is a regression baseline, not a representative industry dataset or provider-validity evaluation. Precision = TP/(TP+FP); recall = TP/(TP+FN). No claims of superiority to other tools are made.

Performance is scanner-only: 1,000 files of exactly 64 KiB (62.5 MiB total), 100 synthetic candidate occurrences, one warm-up and five sequential timed trials. Report all timings and median throughput. Fixture generation, SQLite/history writes, UI, process startup and memory use are excluded. Filesystem cache is not flushed. Results depend on hardware, OS, Go version and cache state; they are not service latency or production scale claims.

Functional pass counts exclude skipped tests and distinguish table-driven subtests in the raw summary. Coverage is statement execution, not detection accuracy. Native picker, visual accessibility, zoom/layout and live Jira compatibility remain manual/unverified. The coverage command measures test binaries; separate packaged smoke coverage is not included.

GitHub Actions repeats tests, smoke and evaluation on Windows/Linux and uploads the evaluation with build artifacts. A local report does not establish that CI passed; check the linked Actions run separately.
