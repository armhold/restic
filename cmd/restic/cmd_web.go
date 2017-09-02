package main

import (
	"github.com/spf13/cobra"
	"github.com/restic/restic/internal/web"
)

var cmdWeb = &cobra.Command{
	Use:   "web [flags]",
	Short: "start the restic web server",
	Long: `
The "web" command starts up a web server for running backups, restores, etc.
`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return web.RunWeb(webOptions.bindHost, webOptions.port)
	},
}

// WebOptions collects all options for the web command.
type WebOptions struct {
	port     int
	bindHost string
}

var webOptions WebOptions

func init() {
	cmdRoot.AddCommand(cmdWeb)

	flags := cmdWeb.Flags()
	flags.StringVarP(&webOptions.bindHost, "host", "H", "localhost", "hostname to bind to")
	flags.IntVar(&webOptions.port, "port", 8080, "port to bind to")
}
