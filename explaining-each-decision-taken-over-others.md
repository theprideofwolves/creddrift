# CredDrift: accepted decisions

1. Keep SQLite: transactional reconciliation and legacy data compatibility without a server.
2. Reconcile only complete coverage: a failed read cannot prove a secret disappeared.
3. Preserve SHA256 identity: compatible ignore/history; hashes of weak values remain guessable.
4. Add manual-action column: first discovery is not rotation; legacy values are not relabeled as verified actions.
5. Remove automatic startup scanning and arbitrary native execution endpoints: explicit scope and fewer local attack surfaces.
6. Request-only Jira token: no vault dependency; user re-enters credentials for each action.
7. Jira durable intent over blind retries: a successful remote write can lose its response.
Manual reconciliation handles ambiguity; evidence comments are not exactly-once.

Common choice: standard-library embedded UI over a SPA/framework. Fewer runtime
dependencies and straightforward packaging; trades rich asynchronous interactions
for full-page forms and serialized actions. Revisit after measured user need.

Alternatives deferred: ServiceNow adapters, cloud hosting, user accounts, automatic
remediation, generalized rule engines and AI. They expand operational scope without
improving the two selected workflows. Revisit only with a specific requirement.

Performance tradeoffs: bounds are intentional; startup and real-workload latency
must be measured before making claims. Source and tests are authoritative.

## CredDrift onboarding decisions

Keep the native folder picker because source directories are CredDrift's input.
Use manual absolute-path entry when the platform dialog is missing or cancelled.
Readability checks do not scan or mark onboarding complete; the selected path
only prefills the explicit scan form. No generic file-upload/sample endpoint is
needed for this directory workflow.

Use a local synthetic fixture and the real scanner/reconciliation path for the
demo. Keep its candidate value hidden and block its Jira handoff. Demo findings
share the database so history is preserved; the demo does not reset real work.

Keep guide progress separate from the database and save it only after a scan
reconciles successfully. Server-rendered forms keep offline distribution simple.
Use Referrer-Policy same-origin to preserve ordinary local form submissions
while strict guards continue rejecting foreign and opaque origins.
