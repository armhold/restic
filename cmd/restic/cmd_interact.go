package main

import (
	"context"

	"github.com/restic/restic/internal/debug"
	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/filter"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
	"github.com/spf13/cobra"
	"path/filepath"
)

var cmdInteract = &cobra.Command{
	Use:   "interact [flags] snapshotID",
	Short: "interactively restore files from a snapshot",
	Long: `
The "interact" command allows you to interactively restore files from snapshots.
`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteract(interactOptions, globalOptions, args)
	},
}

// InteractOptions bundles all options for the 'interact' command.
type InteractOptions struct {
	Target string
	Host   string
	Paths  []string
	Tags   restic.TagLists
}

// files/paths user has selected for restore
var addedFiles map[string]bool = make(map[string]bool)

var interactOptions InteractOptions

var interact *Interact

func init() {
	cmdRoot.AddCommand(cmdInteract)

	flags := cmdInteract.Flags()
	flags.StringVarP(&interactOptions.Target, "target", "t", "", "directory to extract data to")

	flags.StringVarP(&interactOptions.Host, "host", "H", "", `only consider snapshots for this host when the snapshot ID is "latest"`)
	flags.Var(&interactOptions.Tags, "tag", "only consider snapshots which include this `taglist` for snapshot ID \"latest\"")
	flags.StringArrayVar(&interactOptions.Paths, "path", nil, "only consider snapshots which include this (absolute) `path` for snapshot ID \"latest\"")
}

func runInteract(opts InteractOptions, gopts GlobalOptions, args []string) error {
	ctx := gopts.ctx

	if len(args) != 1 {
		return errors.Fatal("no snapshot ID specified")
	}

	if opts.Target == "" {
		return errors.Fatal("please specify a directory to restore to (--target)")
	}

	snapshotIDString := args[0]

	debug.Log("interactively restore %v to %v", snapshotIDString, opts.Target)

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

	if err = repo.LoadIndex(context.TODO()); err != nil {
		return err
	}

	var snapshotID restic.ID

	if snapshotIDString == "latest" {
		snapshotID, err = restic.FindLatestSnapshot(ctx, repo, opts.Paths, opts.Tags, opts.Host)
		if err != nil {
			Exitf(1, "latest snapshot for criteria not found: %v Paths:%v Host:%v", err, opts.Paths, opts.Host)
		}
	} else {
		snapshotID, err = restic.FindSnapshot(repo, snapshotIDString)
		if err != nil {
			Exitf(1, "invalid id %q: %v", snapshotIDString, err)
		}
	}

	interact = NewInteract(ctx, opts, repo, snapshotID)
	err = interact.readCommands()

	// TODO: unsure when to return err vs when to Exitf() in this func

	return err
}


func addFiles(pattern string) {
	// TODO: validate files/paths exist in index

	// TODO: should we expand wildcards into discrete filepaths during "add", or expand later in the extractFiles() selector?

	if pattern[0] != filepath.Separator {
		// resolve pattern relative to currentPath
		pattern = interact.CurrPath() + string(filepath.Separator) + pattern
	}

	addedFiles[pattern] = true

	screenPrintf("added: %s", pattern)
}

func delFiles(pattern string) {
	// TODO: validate files exist in index, from added files

	if pattern[0] != filepath.Separator {
		// resolve pattern relative to currentPath
		pattern = interact.CurrPath() + string(filepath.Separator) + pattern
	}

	_, ok := addedFiles[pattern]
	if ok {
		delete(addedFiles, pattern)
		screenPrintf("deleted: %s", pattern)
	} else {
		// warn user about potential typo
		Warnf("no such files added: \"%s\"\n", pattern)
	}
}

func extractFiles(ctx context.Context, opts InteractOptions, repo *repository.Repository, snapshotID restic.ID) error {
	// TODO: return error rather than exit, and let caller handle error?

	res, err := restic.NewRestorer(repo, snapshotID)
	if err != nil {
		Exitf(2, "creating restorer failed: %v\n", err)
	}

	totalErrors := 0
	res.Error = func(dir string, node *restic.Node, err error) error {
		Warnf("ignoring error for %s: %s\n", dir, err)
		totalErrors++
		return nil
	}

	var selectedPaths []string
	for k := range addedFiles {
		selectedPaths = append(selectedPaths, k)
	}

	res.SelectFilter = func(item string, dstpath string, node *restic.Node) bool {
		matched, err := filter.List(selectedPaths, item)
		if err != nil {
			Warnf("error for path: %v", err)
		}

		return matched
	}

	screenPrintf("restoring %s to %s\n", res.Snapshot(), opts.Target)

	err = res.RestoreTo(ctx, opts.Target)
	if totalErrors > 0 {
		Printf("There were %d errors\n", totalErrors)
	}
	return err
}
