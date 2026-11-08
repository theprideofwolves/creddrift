# CredDrift: interview study guide

## Explain it in 30 seconds

CredDrift scans an explicitly selected local source directory for potential
hardcoded credentials, groups identical values by SHA256, and stores occurrence
and review metadata in SQLite. Successful rescans reconcile moved/deleted
locations; failed scans retain earlier evidence. A reviewed finding can be sent
to Jira without disclosing the raw credential or source path.

## Trace the implementation

Follow main.go to the desktop launcher, onboarding setup, UI scan handler,
scanner.ScanContext and store.Reconcile. Explain why scanning and reconciliation
are separate and why a read failure cannot prove that an old secret disappeared.
Then trace ignore/restore/manual-action updates and the Jira handoff guard.

## Questions and grounded answers

1. What does detection prove? Only that a supported pattern matched. Entropy is
   a character statistic; neither the pattern nor entropy validates a provider key.
2. Why hash? To link repeated values without saving raw strings in SQLite.
   Unsalted hashes of weak values are guessable and still require protection.
3. What does recorded file count mean? Distinct active recorded paths for a hash,
   not permissions, affected accounts or a confirmed compromise radius.
4. What is drift here? Additional recorded locations and rescan reconciliation.
   It is not monitoring provider credential changes or watching directories.
5. What is rotation? A user-recorded action timestamp, not provider API evidence.
   Legacy discovery timestamps are not migrated into verified rotation history.
6. Why a transaction? A reconciliation error must roll back changes rather than
   leave partially retired locations. Excluded and out-of-scope evidence stays.
7. Why synchronous scanning? A small standalone UI and serialized mutations keep
   local state simple. Long scans block other review actions; deadlines are cooperative.
8. What does setup verify? Folder readability or Jira account identity only.
   Neither runs a scan nor verifies Jira project create/comment permissions.
9. How are duplicate Jira issues reduced? Persist intent before sending. If the
   response is uncertain, inspect Jira and link its key. This is not exactly-once
   delivery; comments can also succeed without a received response.
10. What is shipped? A local Windows/Linux tool with automated regression and
    package checks. Do not claim real users or production deployment without evidence.

## Practice with source

Read scanner_test.go, reconcile_test.go, server_test.go and ticket_test.go.
Predict the result of moving/deleting a candidate, excluding a former location,
an overlong line, ignoring a repeated value and an uncertain Jira response.
Explain what stays local and why export paths/hashes remain sensitive.

For interviews, describe the AI assistance and the work you personally reviewed,
tested and can explain. These notes support study; they cannot replace practice.
