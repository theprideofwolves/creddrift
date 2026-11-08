# CredDrift

A local Go secret-discovery and triage tool with SQLite, scope-aware rescans,
redacted Jira handoff, and a bundled synthetic demo.

## Two focused capabilities
1. Review potential hardcoded AWS IDs, classic GitHub tokens, and generic
api_key/password assignments without exposing candidate values in the UI.
2. Keep occurrence state accurate across successful rescans and track reviewed
remediation in Jira without leaking source contents.

Launch, choose an absolute folder, scan, inspect locations, ignore/restore
candidates, and optionally record your own manual rotation action. The app does
not contact providers to validate, revoke, or rotate credentials. Entropy is a
character statistic; it is not proof of validity or password weakness.

No automatic current-directory scan at startup. Default excluded directories:
.git, node_modules, vendor, CredDrift. Binary files and symlinks are excluded.
Limits: 50000 files, 32 MiB per file, 1 MiB per line, 100000 candidate matches.
Exceeding a limit or encountering an unreadable file fails the scan and retains
prior findings. Scans run synchronously with a three-minute cooperative deadline and request-cancellation checks between filesystem operations; there is no dedicated cancel button. A blocked OS filesystem read can delay cancellation. Choose a
specific source directory instead of a whole filesystem.

Successful reconciliation is transactional. It deactivates moved/deleted
occurrences only within successful coverage, retains excluded/binary locations
as historical observations, and reactivates candidates when they reappear.
The active view is last-observed evidence, not a guarantee of current filesystem
state. Full paths in exports are sensitive. File count means local occurrence
count, not permissions or compromise blast radius.

Raw candidate values are transient; SQLite stores SHA256 identities. Hashes are
not encryption, and weak candidates remain guessable. Protect the database and
backups. Existing SHA identities are retained for compatibility. No Git-history
scan, provider validity test, dependency CVE scan, or verified rotation tracking.

## Existing users
The new launcher uses the OS application-data directory. To carry an old
creddrift.db forward, stop both versions, back it up, copy it into the new data
directory, and launch. A migration adds an explicit manual-action timestamp;
legacy discovery/rotation timestamps are not presented as verified rotation.
Relative legacy locations should be reviewed and rescanned from an absolute
path; they cannot be confidently reconciled across arbitrary working directories.
The old config.yaml startup auto-scan is intentionally replaced by explicit UI scope.

## Installation and privacy
Download the platform ZIP from GitHub Releases and extract it. On Windows,
double-click the executable. On Linux, run the binary; if no desktop opener is
installed, copy the printed localhost URL into a browser. No development tools,
Docker, database server, or paid API is needed for the offline workflow.
The executable is unsigned: OS warnings are possible. Close the application with
Ctrl+C in its console. Closing the browser does not stop it.

The server binds to an ephemeral localhost port and uses a private per-launch
token, exact Host and Origin checks, CSP, HTML escaping and request limits.
Single-user OS trust boundary: no roles or enterprise multi-tenancy. Local
privileged processes can read memory and data. Do not expose it to a network.
No telemetry, background uploads, automatic remediation or AI dependency.

Data defaults to the OS user configuration directory / CredDrift.
Set CREDDRIFT_DATA_DIR to change its parent; CREDDRIFT_NO_BROWSER=1 suppresses
browser opening. Only one launcher may use a data directory. After a crash,
confirm the process has stopped before removing application.lock.
Stop the app before backup/restore. Replace executables to upgrade; user data
stays separate. Uninstall by deleting extracted binaries; delete the data
directory separately only if you intend to remove history.

## Release verification
CI runs unit/integration tests, race detection, vet, and builds on Windows/Linux.
Version tags or an explicit [release] commit on main publish the VERSION-selected portable ZIPs and SHA256 checksums after tests pass.
Dependency metadata is included using go version -m; it is not a full license
inventory or a signed supply-chain attestation. No unsupported scale or
enterprise-certification claim is made.

## Jira setup and delivery semantics
Use Jira Cloud's HTTPS origin, an authorized account email/API token, project
key, and a supported issue type (Task by default). Browse/Create/Comment
permissions are required as appropriate. Projects with custom mandatory fields
may reject creation: configure a suitable type or explicitly link an existing
issue. The link action validates access to the issue, not the human's choice.
Credentials are used only for the request and never saved in ticket history.
No source snippets or raw credentials are sent. Review the disclosure before send.

The app persists a create intent before contacting Jira. A missing/uncertain
response blocks repeat creates: search Jira for the shown correlation ID, then
link the existing key. Definitive 400/401/403/404/422 rejections may be corrected
and retried. No automatic retry loops. Existing mappings receive evidence
comments; an uncertain comment may have succeeded, so inspect Jira before retry.
No automatic issue closure or bidirectional/background synchronization.
Mappings are scoped to origin and local candidate hash. Losing local history
or using a different origin breaks local deduplication: link existing work first.
Live Jira permission/field compatibility requires actual authorized testing;
TLS fake-server tests are not proof of live integration.

## Developer checks
`go test -race ./...`, `go vet ./...`, `go build ./.`.
See architecture.md, interviewprep.md and explaining-each-decision-taken-over-others.md.

## Guided first run

Choose **Try the guided demo** or **Use my data**. The offline demo writes a
synthetic password fixture under the application's synthetic-demo directory and
scans it. On a fresh workspace it produces one candidate with its value hidden.
The demo uses the same local database; existing findings remain visible.
Findings located in the demo directory cannot create real Jira tickets.

For your own source, choose a folder using the native picker or paste an absolute
directory path. The picker needs Windows PowerShell on Windows, osascript on
macOS when building from source, or optional zenity on Linux. If unavailable or
cancelled, paste the path. The readability check does not scan files. The chosen
path is held for this session and prefilled in the scan form; scanning is explicit.

The optional Jira sign-in check calls GET /rest/api/3/myself. It checks identity,
not project create/comment permissions, and never creates tickets or retains your
email/API token. Re-enter credentials for a reviewed finding handoff.

A successful scan and database reconciliation remember guide completion in
onboarding.json, containing only a version and completion flag. Setup checks and
failed scans do not mark completion. **Restart the guide** preserves findings;
**Continue to existing workspace** opens saved work without scanning. Corrupt or
missing guide preferences show the guide again. Progress-save failures are shown
without discarding findings.
