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

## Onboarding design and study notes

The internal/onboarding package owns the welcome/setup routes, escaped progress
panel, declared non-secret session fields, and a versioned completion preference.
Core handlers explicitly call Complete only after a successful domain operation.
A failing policy evaluation is still a successful task; a partial LogFlow report
is not marked complete. Guide status is not a security/compliance conclusion.

GET/sample/setup routes do not run scans or create tickets. POST setup checks
use bounded read-only requests; Jira identity is not project authorization. Native
folder selection is explicit, bounded and has a manual fallback. Preflight checks
never store tokens, and Jira checks preserve previously selected local fields.

Why not a JS tour library? Server-rendered forms and a small guide keep packaging
light, work offline and are keyboard accessible. Each independent repository
contains its own internal copy and tests; no cross-project service is required.
That trades centralized updates for simpler standalone distribution.

Practice: explain a fresh launch, a failed check, a successful demo, persistence
after restart, and why restarting the guide cannot delete assessment data. Read
onboarding_test.go and the packaged smoke script to distinguish a connection
check from an end-to-end live integration test.

Browser regression: use Referrer-Policy same-origin rather than no-referrer.
The latter can make a browser emit an opaque Origin on ordinary form posts,
which correctly fails strict origin checks. Same-origin preserves legitimate
local form submission while suppressing cross-origin referrers. Null and foreign
origins remain rejected. This was verified through a real browser interaction.
