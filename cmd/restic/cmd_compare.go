package main

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
)

var cmdCompare = &cobra.Command{
	Use:   "compare [flags] [snapshot-ID ...]",
	Short: "compare usage between snapshots",
	Long: `
The "compare" command allows usage comparisons between files and directories in two different snapshots.

The special snapshot-ID "latest" can be used to list files and directories of the latest snapshot in the repository.
`,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCompare(compareOptions, globalOptions, args)
	},
}

// CompareOptions collects all options for the compare command.
type CompareOptions struct {
	ListLong bool
	Host     string
	Tags     restic.TagLists
	Paths    []string
}

var compareOptions CompareOptions

func init() {
	cmdRoot.AddCommand(cmdCompare)

	flags := cmdCompare.Flags()
	flags.BoolVarP(&compareOptions.ListLong, "long", "l", false, "use a long listing format showing size and mode")

	flags.StringVarP(&compareOptions.Host, "host", "H", "", "only consider snapshots for this `host`, when no snapshot ID is given")
	flags.Var(&compareOptions.Tags, "tag", "only consider snapshots which include this `taglist`, when no snapshot ID is given")
	flags.StringArrayVar(&compareOptions.Paths, "path", nil, "only consider snapshots which include this (absolute) `path`, when no snapshot ID is given")
}

// returns the size of the given tree and all its children
func printTreeCompare(repo *repository.Repository, id *restic.ID, prefix string) (size uint64, err error) {
	tree, err := repo.LoadTree(context.TODO(), *id)
	if err != nil {
		return 0, err
	}

	for _, entry := range tree.Nodes {
		//Printf("%s\n", formatNode(prefix, entry, compareOptions.ListLong))

		if entry.Type == "dir" && entry.Subtree != nil {
			subdirSize, err := printTreeCompare(repo, entry.Subtree, filepath.Join(prefix, entry.Name))
			if err != nil {
				return 0, err
			}

			size += subdirSize
			Verbosef("size of subdir %s: %s\n", entry.Name, formatBytes(subdirSize))
		} else if entry.Type == "file" {
			size += entry.Size
		}
	}

	return size, nil
}

func runCompare(opts CompareOptions, gopts GlobalOptions, args []string) error {
	if len(args) == 0 && opts.Host == "" && len(opts.Tags) == 0 && len(opts.Paths) == 0 {
		return errors.Fatal("Invalid arguments, either give one or more snapshot IDs or set filters.")
	}

	repo, err := OpenRepository(gopts)
	if err != nil {
		return err
	}

	if err = repo.LoadIndex(context.TODO()); err != nil {
		return err
	}

	var snapshots []*restic.Snapshot

	ctx, cancel := context.WithCancel(gopts.ctx)
	defer cancel()
	for sn := range FindFilteredSnapshots(ctx, repo, opts.Host, opts.Tags, opts.Paths, args) {
		Verbosef("snapshot %s of %v at %s):\n", sn.ID().Str(), sn.Paths, sn.Time)

		snapshots = append(snapshots, sn)
	}

	Verbosef("found %d snapshots\n", len(snapshots))

	if len(snapshots) != 2 {
		Verbosef("need to specify 2 snapshots\n")
		return nil
	}

	sn1 := snapshots[0]

	treeSize, err := printTreeCompare(repo, sn1.Tree, string(filepath.Separator))
	if err != nil {
		return err
	}

	Verbosef("total size: %s\n", formatBytes(treeSize))

	return nil
}
