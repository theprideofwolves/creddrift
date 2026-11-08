# CredDrift v1.2.1

Portable single-user Windows/Linux directory secret discovery and triage: pattern
matches, hashed SQLite identities, scope-aware rescans, ignore/restore/manual
review actions, metadata export and optional reviewed Jira Cloud handoff.

This patch removes unrelated project docs and test branches, unused upload/sample
onboarding code, obsolete YAML loading and stale diagrams/spec claims. Native
folder selection and its manual fallback remain. Launcher overrides are now
CREDDRIFT_DATA_DIR (parent directory) and CREDDRIFT_NO_BROWSER. Default data
locations and existing database/ticket formats are unchanged. Direct scan requests
must use an absolute directory, matching setup guidance.
Fixed retired view/export for candidates whose last active location was removed.

Automated gates include race-enabled Go tests, vet, Windows/Linux builds and
packaged setup/demo/redaction/export/restart checks. Scanner/store tests cover
coverage, rollback and review state; Jira contracts use TLS fake servers.
Live Jira project permissions/custom fields, native OS dialogs, clean-machine
interactive launch, human accessibility and sustained resource measurements
remain separate validation. Tests do not establish enterprise deployment.

Packages are unsigned and include dependency build metadata and ZIP checksums.
No developer tools, database server or paid API are needed for offline use.
