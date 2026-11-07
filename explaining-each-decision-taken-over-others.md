# CredDrift: accepted decisions

1. Keep SQLite: transactional reconciliation and legacy data compatibility without a server.
2. Reconcile only complete coverage: a failed read cannot prove a secret disappeared.
3. Preserve SHA256 identity: compatible ignore/history; hashes of weak values remain guessable.
4. Add manual-action column: first discovery is not rotation; legacy values are not relabeled as verified actions.
5. Remove automatic startup scanning/native execution endpoints: explicit scope and fewer local attack surfaces.
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
