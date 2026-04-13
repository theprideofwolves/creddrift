# CredDrift Full Specification

## Overview
CredDrift is a self hosted security tool. It monitors the life of secrets. It does not just find them. It tracks how they change over time. 

## Core Logic
* Fingerprinting: When a secret is found, hash it with SHA 256. Never save the raw string. 
* Entropy Scoring: Use Shannon entropy. Any string with a score above 3.5 is a potential secret. Strings with an entropy of 3.5 or lower should be flagged as 'Weak_Secret'.
* Blast Radius: Count how many files contains the same hash. 
* Drift: Alert when a hash appears in a new location.
* Rotation: Alert when a hash has not changed for 90 days.

## Database Schema (SQLite)
* credentials table: ID, hash, secret type, entropy score, first seen, last seen, last rotated.
* locations table: ID, credential ID, file path, line number, active status.

## Scanner Patterns
* AWS: AKIA prefix followed by 16 characters.
* GitHub: ghp_ prefix followed by 36 characters.
* Generic: Look for keys named api_key or password with high entropy values.

## Architecture
* Go Backend: Dynamic API handlers interacting with config dependencies.
* SQLite: Pure Go driver for storage.
* GRC Metrics: Blast Radius evaluates unique instances, Drift flags relocated hashes, and stale secrets >5m are automatically removed from view.
* Embedded UI: Deeply interactive dashboard supporting Bulk Action iterations across Multi-selected table checkboxes.
* Actions Toolbar: Dedicated GUI region enabling Mark as Rotated, Ignore, Restore, and natively hooked Copy/Reveal paths.
* Cross Platform Infrastructure: Contextual bindings via runtime.GOOS supporting osascript, zenity, and powershell folder logic automatically natively in the browser depending on host OS.
* Navigation: Hamburger menu to view Main Dashboard, Ignored Keys, and Rotation History.
* Sorting: Table columns like Type, Score, and Status can be sorted ascending or descending.

## Project Status
* Phase 1 (Scanner): Complete
* Phase 2 (Persistence): Complete
* Phase 3 (Web Dashboard): In Progress