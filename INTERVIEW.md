# Interview walkthrough: Local secret discovery and triage

## Explain it in 45 seconds
I built pattern-based local secret discovery with entropy scores, SQLite location tracking, and dashboard ignore/restore/manual rotation actions. I improved scan error handling and kept findings visible until explicitly reviewed rather than hiding them after five minutes.

## Demonstrate it
1. Run `go run .` from the repository directory and open http://localhost:8080.
2. Select the repository's demo-workspace directory. Its passwords are deliberately synthetic fixtures, not real credentials.
3. Find the same synthetic value in two files and explain SHA-256 identity versus multiple locations.
4. Ignore it, switch to ignored findings, restore it, then record manual rotation and view history.
5. Run `go test ./...`: pattern/location checks, lifecycle, scan-error handling, and local HTTP boundaries.

## Questions to be ready for
- Pattern versus entropy? Patterns find candidates; entropy describes randomness. Neither establishes that a credential is valid.
- Why hashes? Link repeated candidate values without storing the raw values in SQLite. Unsalted hashes of weak values remain guessable; hashing is not encryption.
- What is drift? The dashboard identifies repeated credential locations. It does not establish a provider credential's real-world state.
- Rotation? A manual status timestamp; no provider API rotates or revokes credentials.
- Persistence? Findings remain reviewable; deleting a source secret does not automatically reconcile old location records.
- Security boundary? Listener binds to loopback, checks Host/Origin for local actions. It is still a local prototype without user authentication.

## Timeline and resume consistency
The submitted resume describes the original project's core behavior. These subsequent improvements strengthen the same project. If asked, distinguish the original implementation from later hardening. Practice the demo and inspect the source before describing decisions as your own. Do not claim production deployment, measured business impact, or autonomous agents from this repository.
