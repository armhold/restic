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
func printTreeCompare(repo *repository.Repository, id *restic.ID, prefix string, pathMap map[string]uint64) (uint64, error) {
	var size uint64

	tree, err := repo.LoadTree(context.TODO(), *id)
	if err != nil {
		return 0, err
	}

	for _, entry := range tree.Nodes {
		//Printf("%s\n", formatNode(prefix, entry, compareOptions.ListLong))

		if entry.Type == "dir" && entry.Subtree != nil {
			subdirSize, err := printTreeCompare(repo, entry.Subtree, filepath.Join(prefix, entry.Name), pathMap)
			if err != nil {
				return 0, err
			}

			fullPath := filepath.Join(prefix, entry.Name)
			size += subdirSize
			pathMap[fullPath] = size

			Verbosef("size of subdir %s: %s\n", fullPath, formatBytes(subdirSize))
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

	sn1, sn2 := snapshots[0], snapshots[1]

	var map1, map2 = make(map[string]uint64), make(map[string]uint64)

	treeSize, err := printTreeCompare(repo, sn1.Tree, string(filepath.Separator), map1)
	if err != nil {
		return err
	}
	Verbosef("total size: of %s: %s\n", sn1.ID(), formatBytes(treeSize))

	treeSize, err = printTreeCompare(repo, sn2.Tree, string(filepath.Separator), map2)
	if err != nil {
		return err
	}
	Verbosef("total size: of %s: %s\n", sn2.ID(), formatBytes(treeSize))

	compareTrees(map1, map2)

	return nil
}

func compareTrees(map1, map2 map[string]uint64) {
	// map2 should be the more recent snapshot

	// iterate over more recent snapshot
	for path, size2 := range map2 {
		size1, ok := map1[path]
		if ok {
			var change uint64
			var sign string
			if size1 < size2 {
				change = size2 - size1
				sign = "+"
			} else if size2 > size1 {
				change = size1 - size2
				sign = "-"
			} else {
				change = 0
			}

			if change != 0 {
				Verbosef("%s: size1: %d, size2: %d, change: %s%s (%d)\n", path, size1, size2, sign, formatBytes(change), change)
			}
		} else {
			Verbosef("%s: [missing]\n", path)
		}
	}
}
