package main

import (
	"github.com/restic/restic/internal/web"
	"github.com/spf13/cobra"
)

var cmdWeb = &cobra.Command{
	Use:   "web [flags]",
	Short: "start the restic web server",
	Long: `
The "web" command starts up a web server for running backups, restores, etc.
`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWeb(webOptions, globalOptions, args)
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

func runWeb(opts WebOptions, gopts GlobalOptions, args []string) error {
	repo, err := OpenRepository(gopts)
	if err != nil {
		return err
	}

	if !gopts.NoLock {
		lock, err := lockRepo(repo)
		defer unlockRepo(lock)
		if err != nil {
			return err
		}
	}

	//ctx, cancel := context.WithCancel(gopts.ctx)
	//defer cancel()

	web.RunWeb(opts.bindHost, opts.port, repo)

	return nil
}
