// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Command passmcp-graph builds a local, open graph of which agents reach which
// MCP servers and tools, under which identities, from passmcp's evidence.
package main

import (
	"os"
	"time"

	"satellion.com/passmcp-graph/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{Stdout: os.Stdout, Stderr: os.Stderr, Now: time.Now}))
}
