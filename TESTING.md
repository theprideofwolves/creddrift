# Quick manual check

Automated tests cover discovery, reconciliation, redaction, review state, restart, request guards and mocked Jira delivery. These steps cover the remaining browser/OS behavior. Use a temporary folder containing synthetic fixtures only. Skip live Jira; it is optional.

- [ ] Launch the new executable. Welcome offers **Try the guided demo** and **Use my data**; nothing scans automatically.
- [ ] Switch **Dark mode**, then **Light mode**. Dark background is black; light background is white. Green accent, readable inputs and visible keyboard focus remain consistent. Choice survives navigation and reload.
- [ ] Run the demo. One synthetic candidate appears with a file/line location, no raw value. Export opens valid JSON containing metadata only.
- [ ] Open setup. Choose a temporary synthetic folder with the native picker; cancel once and confirm manual path entry still works. **Check setup** reports readability without scanning.
- [ ] Scan that folder. Ignore a candidate, confirm it appears in **Ignored**, then restore it. Record a manual rotation action and confirm its wording does not claim provider verification.
- [ ] Delete the synthetic source file and rescan. Its occurrence moves to **Retired observations**; that view's export contains the retired record. Recreate it and rescan to reactivate it.
- [ ] Restart the guide, then restart the app with the same data directory. Findings remain available.
- [ ] At a narrow browser width and 200% zoom, confirm paths wrap, buttons remain usable and the page has no horizontal overflow. Tab through the theme switch, forms, navigation and Jira disclosure.

No real credentials, provider requests or live tickets are needed. Windows symlink tests may be skipped if the account lacks symlink privileges. Live Jira permissions/custom fields, macOS/Linux native dialogs and sustained performance remain unverified on this Windows run.

## Evidence walkthrough

- [ ] Scan two temporary files containing the same synthetic value. Open Evidence timeline: two new occurrences share an identity.
- [ ] Delete one file, rescan: absent from checked coverage plus still present. Export timeline and confirm metadata only.
- [ ] Make the remaining file binary, rescan: unverified with a coverage reason, not resolved.
- [ ] Recreate the removed file with the original synthetic assignment and rescan: reappeared.
- [ ] Use the per-finding identity link; confirm other identities' events are filtered while scan outcomes remain visible.
- [ ] Restart the app: scan history remains. Review dark/light mode, keyboard navigation and narrow-width wrapping on the timeline.

Source absence never proves revocation; manual rotation remains a separate user action. Do not use real credentials for this walkthrough.
