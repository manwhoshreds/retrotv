═══════════════════════════════════════════════════════════════════════
RETROTV ROADMAP
═══════════════════════════════════════════════════════════════════════

PHASE 1: VIDEO PLAYBACK
───────────────────────────────────────────────────────────────────────
✓ Qt window
✓ MPV integration
✓ Embedded video playback
✓ Local media files

STATUS: COMPLETE

PHASE 2: CHANNELS
───────────────────────────────────────────────────────────────────────
✓ Channel model
✓ Hardcoded channels
✓ Channel switching
✓ Channel numbers
✓ Keyboard navigation

STATUS: COMPLETE

PHASE 3: INFO OVERLAY
───────────────────────────────────────────────────────────────────────
□ Show current channel number
□ Show channel logo
□ Show current program
□ Show episode title
□ Show start/end times
□ Show next program
□ Auto-hide after 5 seconds
□ Show automatically on channel change
□ Fade in/out animations

PHASE 4: FULL TV GUIDE (EPG)
───────────────────────────────────────────────────────────────────────
□ Traditional cable-style guide
□ Time grid across the top
□ Channel list on the left
□ Program navigation
□ Program descriptions
□ Jump to current time
□ Search functionality
□ Channel logos
□ Poster artwork

PHASE 5: M3U SUPPORT
───────────────────────────────────────────────────────────────────────
□ Load channels from M3U files
□ Support local media paths
□ Support network streams
□ tvg-id support
□ tvg-name support
□ Channel grouping
□ Channel logos

PHASE 6: XMLTV SUPPORT
───────────────────────────────────────────────────────────────────────
□ Parse XMLTV files
□ Current program information
□ Next program information
□ Episode descriptions
□ Season/episode numbers
□ Artwork support
□ Ratings
□ Categories

PHASE 7: REMOTE-CONTROL UX
───────────────────────────────────────────────────────────────────────
□ Up/Down = Channel switching
□ Left/Right = Volume
□ G = Guide
□ I = Info
□ ESC = Back
□ 0-9 = Direct channel entry
□ Enter = Confirm channel
□ Full keyboard navigation
□ Gamepad support
□ LIRC support

PHASE 8: SERVER MODE
───────────────────────────────────────────────────────────────────────
□ RetroTV Server
□ Shared media library
□ Shared channel lineup
□ Shared XMLTV data
□ Multiple clients
□ Watched status sync
□ User profiles

PHASE 9: SMART CHANNELS (THE KILLER FEATURE)
───────────────────────────────────────────────────────────────────────
□ Kodi-style smart playlists
□ Genre rules
□ Studio rules
□ Year filters
□ Random playback
□ Shuffle episodes
□ No-repeat windows
□ Seasonal channels
□ Dynamic channel generation

Examples:

Adult Swim
    Genre = Animation
    Studio = Adult Swim
    Shuffle = True
    No Repeats = 30 Days

Star Trek
    Franchise = Star Trek
    Chronological = False
    Random Episodes = True

Christmas
    Tag = Christmas
    Enabled = December Only

PHASE 10: THE COOL SHIT
───────────────────────────────────────────────────────────────────────
□ CRT shaders
□ Channel static transitions
□ Old-school TV startup animation
□ Channel logos
□ Commercial injection
□ Adult Swim bumpers
□ Seasonal events
□ Weather widget
□ Clock overlay
□ Screensavers
□ Retro themes
□ Multiple skins
□ Sound effects
□ Fake TV guide advertisements

═══════════════════════════════════════════════════════════════════════
CURRENT STATUS
═══════════════════════════════════════════════════════════════════════

✓ Phase 1 Complete
✓ Phase 2 Complete
→ Next: Phase 3 (Info Overlay)

GOAL:
Build a self-hosted, open-source, Kodi-inspired virtual cable box
that uses smart playlists, M3U, and XMLTV to recreate traditional TV.
═══════════════════════════════════════════════════════════════════════