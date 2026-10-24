# RetroTV MVP Punch List

This is a living document. Keep items current as requirements change, and check them off only after implementation and verification.

## Client

- [ ] Periodically refresh guide data so long-running clients do not exhaust the initial 12-hour window.
  - Refresh `/guide.xml?hours=12` every 30 minutes.
  - Download and parse the guide outside the UI thread.
  - Preserve the selected channel and timeline position when applying an update.
  - Keep the last valid guide when a refresh fails.
  - Avoid rebuilding the guide UI when the returned data has not changed.

