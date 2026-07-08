// Command gendocs regenerates the command reference in docs/reference from
// the cobra command tree — the same help text the binary shows, so the docs
// cannot drift from the code. Run via `make docs`.
package main

import (
	"log"
	"os"

	"github.com/cleura/cleura-cli/internal/cli"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	const dir = "docs/reference"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cli.NewRootCommand("dev")
	root.InitDefaultCompletionCmd() // normally added lazily at execute time
	disableAutoGenTag(root)         // no per-run date footers: keep regeneration diffs meaningful

	if err := doc.GenMarkdownTree(root, dir); err != nil {
		log.Fatal(err)
	}
	log.Printf("generated command reference in %s", dir)
}

func disableAutoGenTag(cmd *cobra.Command) {
	cmd.DisableAutoGenTag = true
	for _, sub := range cmd.Commands() {
		disableAutoGenTag(sub)
	}
}
