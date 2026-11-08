# CredDrift: implemented product scope

CredDrift is a single-user local source-directory scanner and candidate review
tool. The executable opens a protected localhost browser UI on a random port.
There is no automatic startup scan or monitored background directory.

## Discovery

An explicit absolute directory scan looks for AKIA IDs (16 uppercase letters or
digits after the prefix), classic ghp_ tokens (36 alphanumeric characters after
the prefix), and api_key/password assignments. Shannon entropy is recorded;
generic candidates at or below 3.5 retain the legacy Weak_Secret type. Patterns
and entropy do not establish provider validity or actual password strength.

Traversal excludes configured directory names and symlinks/non-regular files.
The binary heuristic checks the first 512 bytes for NUL. Files above 32 MiB,
lines at the scanner's 1 MiB buffer limit, more than 50000 visited regular files,
more than 100000 collected candidates, read errors or cancellation fail the
scan. The three-minute request deadline is cooperative, not a hard OS I/O timeout.

## Persistence and review

SQLite credentials store SHA256 identity, type, entropy and discovery timestamps;
locations store file/line and active state. Ignore state is per candidate hash.
A separate nullable rotation_recorded_at holds a user's manual action. Existing
legacy last_rotated values do not become verified provider-rotation evidence.

Only successful scans reconcile in one transaction. In-scope scanned locations
are replaced by current matches; eligible deleted locations are retired.
Excluded/binary/symlink and out-of-scope observations are retained. Reappearing
locations reactivate. Active, ignored, retired and manual-action views export
metadata without raw candidate values. Paths and unsalted hashes are sensitive.
There is no automatic time-based expiry, provider rotation or Git-history scan.

## Jira

A reviewed finding can create/link a Jira Cloud issue or append a comment to
its stored mapping. Only candidate type/hash, recorded file count, review time
and correlation information are disclosed. Raw candidates, paths and snippets
are not sent. A durable create intent blocks blind retries after uncertain
delivery. Demo-directory findings cannot create real Jira tickets.

## Distribution

Windows/Linux x64 portable ZIPs include the binary, docs and build metadata;
checksums are published after CI. No developer tools or paid API are needed
offline. The optional native folder picker has manual path entry as fallback.
