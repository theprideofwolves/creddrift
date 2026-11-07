# CredDrift: implemented architecture

Launcher -> OS data directory -> SQLite schema upgrade -> protected local UI.
User-selected root -> bounded file traversal -> patterns/entropy -> complete
text-file coverage -> transactional occurrence reconciliation. Any scan error
prevents reconciliation. Ignored identities apply across locations. Jira uses
redacted type/hash/count/time and durable independent delivery records. A global
UI mutex serializes scans/review actions. SQLite keeps one open connection.
SHA256 identities preserve legacy compatibility but are not secret vaulting.

## Operational boundaries
Private single-user tool. Browser access requires the launcher token. No remote
deployment, authentication roles, telemetry, continuous watch, or automatic
containment. Data corruption stops load instead of silently overwriting history.
Backups and third-party credentials remain the user's responsibility.

## Tests and evidence
Feature tests are alongside implementation, local-request tests in
internal/desktop, and integration contract tests beside the adapters. CI verifies
Windows/Linux builds and race checks. Mocks verify defined contracts; live
permissions, provider behavior and clean-machine interactive UI require manual
validation. Release packaging does not establish enterprise production readiness.
