# CredDrift interview guide

## Explain it in 30 seconds

CredDrift is a local Go credential-discovery and evidence-tracking tool. It groups repeated candidate values by SHA-256 while storing only metadata in SQLite. Its main distinction is explaining what a rescan actually checked: a missing match can mean source removal, exclusion, or incomplete coverage. Successful scans commit occurrence state and evidence history together. Failed scans retain prior findings. Optional Jira handoff handles uncertain delivery without blindly creating duplicate tickets.

## Follow the code

`main.go` opens the data store and configures exclusions. `internal/desktop` starts a token-protected loopback server. `internal/onboarding` checks folder readability without scanning. `internal/ui/server.go` runs the explicit scan with a cooperative deadline. `scanner/scanner.go` traverses bounded eligible files. `store.Reconcile` snapshots previous occurrences, reconciles current matches, and calls `appendTimeline` before committing the transaction. `internal/store/timeline.go` queries metadata history for the UI and JSON export.

## Explain the novelty with an example

1. A value appears in two files: one identity, two new occurrence events.
2. Remove one file and scan both locations: one event becomes absent from checked coverage; the other is still present.
3. Exclude the remaining directory: its evidence stays active but this scan records it as unverified, with an exclusion reason.
4. Restore the first location and scan: it is reappeared, preserving the same identity.
5. Cancel a later scan: history records cancellation with no partial events. Earlier evidence is retained.

A disappearance and appearance at another path are observations, not proof that a file moved. Source removal is not provider revocation. Rotation recording is a separate manual user action, never provider verification.

## Questions to practice

- **Identity versus occurrence?** SHA-256 links values; a location is identity plus path plus line. A new path is a new occurrence even when the identity was already known. A previously retired exact occurrence is reappeared.
- **Why transactions?** Updating occurrence activity without the corresponding timeline would produce contradictory evidence. Both commit or both roll back. The UI records a failed attempt separately if reconciliation fails.
- **Why record outside-scope events?** To show that earlier active evidence was retained, not checked. Such events do not imply file reads outside the selected root. Coverage reasoning uses metadata; the scanner reads only eligible in-scope files.
- **What is not reconstructed?** Existing databases gain history tables, but past scans cannot be recovered. Their first post-upgrade observation may be still present rather than newly discovered.
- **Why hash rather than encrypt?** Identity comparison does not require recovering the value. Unsalted SHA-256 still permits guessing weak values; this is a documented privacy limitation, not encryption.
- **Does detection establish validity?** No. Supported patterns and entropy provide candidates only. No provider requests occur.
- **What does failure prove?** That coverage was incomplete. No partial matches change evidence; history records failed/cancelled outcomes without raw errors or values. If history persistence fails, the UI reports it.
- **What does timeline export contain?** The latest 100 scan attempts, finish timestamps, roots, committed eligible-file counts and occurrence events. Optional hash filtering filters events, not scan outcomes. Paths and hashes remain sensitive. Older history stays in SQLite; no automatic retention cleanup exists.
- **What are the concurrency limits?** UI mutations are serialized; scans block other finding actions. SQLite uses one connection. This is a single-user desktop tool, not a distributed scanner.
- **Exactly-once Jira delivery?** No. Durable intent blocks blind repeat creates after uncertain responses. Existing issue comments can also have uncertain outcomes. Equivalent HTTPS origins share identity; older mappings migrate conservatively.
- **What are the limitations?** A changing filesystem is not an atomic snapshot. Deadlines cannot interrupt every blocked OS read. Historical metadata grows with scans and active occurrences. No detection-accuracy, user-adoption or production-performance claims have been established.

## Evidence you can show

Read `timeline_test.go`, `reconcile_test.go`, `server_test.go`, `origin_test.go` and the packaged smoke script. Run the synthetic scenario in TESTING.md. Explain why each state appears before showing it. Test coverage measures executed statements, not detection precision or operational effectiveness.

Describe AI assistance honestly and explain the code you personally reviewed. A demo and these notes support preparation; they do not replace understanding the implementation.
