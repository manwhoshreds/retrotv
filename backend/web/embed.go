// Package web holds the configuration UI, compiled into the server binary so a
// release ships as a single file.
package web

import "embed"

// Files is the web UI rooted at this directory (index.html, app.js, styles/).
//
//go:embed index.html app.js styles
var Files embed.FS
