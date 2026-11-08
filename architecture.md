# CredDrift: implemented architecture

```mermaid
flowchart TD
    A[Launcher and OS data directory] --> B[SQLite schema migration]
    B --> C[Protected localhost UI]
    C --> D[Explicit directory scan]
    D --> E[Bounded traversal and pattern matching]
    E --> F{Scan succeeded?}
    F -->|No| G[Retain prior findings]
    F -->|Yes| H[Transactional location reconciliation]
    H --> I[Review and redacted JSON export]
    I --> J[Explicit Jira handoff with durable intent]
```

## Source map

- main.go opens the user-data database and declares default exclusions. Config
  holds in-memory scan targets; no YAML file is read and no startup scan runs.
- internal/desktop launches loopback HTTP with a random token/port, host/origin
  guards, CSP and one data-directory launcher lock.
- internal/onboarding provides demo/own-folder selection, native picker with
  manual fallback, readability check and optional read-only Jira identity check.
  Setup never scans; selected paths remain in session memory only.
- internal/ui/server.go serializes scans and review actions through a mutex.
  The scanner gets a cooperative three-minute deadline. A failed scan never
  calls Reconcile. Complete guide progress is saved only after reconciliation.
- scanner/scanner.go traverses explicit targets, applies exclusions, matches
  supported patterns and calculates entropy. No provider API validates a match.
- internal/store/store.go uses one SQLite connection, migrates manual-action
  timestamps, hashes candidate values, and reconciles covered locations in a
  transaction. Ignoring a hash applies across its locations. Exports are metadata.
- internal/ticket stores create intent and delivery mappings separately in
  tickets.json; it handles uncertain responses without blind automatic retries.

## State and boundaries

creddrift.db stores candidates, occurrences and review actions. onboarding.json
contains a version/completion flag. Demo source lives under synthetic-demo and
uses the same database as real scans; it does not clear existing findings.
Restarting the guide never resets the database. Jira credentials are request-only.
Candidate strings exist transiently during scanning; hashes are not encryption.

This is a local single-user tool. No network deployment, shared authentication,
provider rotation, continuous monitoring or enterprise deployment is established.
Scopes are observations of a changing filesystem, not an atomic filesystem snapshot.
Tests cover state transitions, redaction, reconciliation failures, local request
guards, mocked Jira contracts and packaged launch/demo/export/restart flows.
Live Jira permissions and native-picker behavior require separate validation.
