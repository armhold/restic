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


// comparison between two snapshots
type comparison struct {
	snap1, snap2 *restic.Snapshot

	// map of paths to their size in bytes
	map1, map2 map[string]uint64
}

func init() {
	cmdRoot.AddCommand(cmdCompare)

	flags := cmdCompare.Flags()
	flags.BoolVarP(&compareOptions.ListLong, "long", "l", false, "use a long listing format showing size and mode")

	flags.StringVarP(&compareOptions.Host, "host", "H", "", "only consider snapshots for this `host`, when no snapshot ID is given")
	flags.Var(&compareOptions.Tags, "tag", "only consider snapshots which include this `taglist`, when no snapshot ID is given")
	flags.StringArrayVar(&compareOptions.Paths, "path", nil, "only consider snapshots which include this (absolute) `path`, when no snapshot ID is given")
}

// for every subdir under id, add an entry to the pathMap with its aggregate size
func sumAllSubdirs(repo *repository.Repository, id *restic.ID, prefix string, pathMap map[string]uint64) (uint64, error) {
	var size uint64

	tree, err := repo.LoadTree(context.TODO(), *id)
	if err != nil {
		return 0, err
	}

	for _, entry := range tree.Nodes {
		//Printf("%s\n", formatNode(prefix, entry, compareOptions.ListLong))

		if entry.Type == "dir" && entry.Subtree != nil {
			fullPath := filepath.Join(prefix, entry.Name)

			subdirSize, err := sumAllSubdirs(repo, entry.Subtree, fullPath, pathMap)
			if err != nil {
				return 0, err
			}

			pathMap[fullPath] = subdirSize
			size += subdirSize

			//Verbosef("size of subdir %s: %s\n", fullPath, formatBytes(subdirSize))
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
		Verbosef("snapshot %s of %v at %s):\r\n", sn.ID().Str(), sn.Paths, sn.Time)
		snapshots = append(snapshots, sn)
	}

	Verbosef("found %d snapshots\r\n", len(snapshots))

	if len(snapshots) != 2 {
		Verbosef("need to specify 2 snapshots\r\n")
		return nil
	}

	c := &comparison{snap1: snapshots[0], snap2: snapshots[1]}

	if err = c.doCompare(repo); err != nil {
		return err
	}

	c.compareTrees()

	return nil
}

func (c *comparison) doCompare(repo *repository.Repository) (error) {
	c.map1 = make(map[string]uint64)
	c.map2 = make(map[string]uint64)

	Verbosef("building map for %s\r\n", c.snap1.ID())
	treeSize, err := sumAllSubdirs(repo, c.snap1.Tree, string(filepath.Separator), c.map1)
	if err != nil {
		return err
	}
	Verbosef("total size: of %s: %s\r\n", c.snap1.ID(), formatBytes(treeSize))

	Verbosef("building map for %s\r\n", c.snap2.ID())
	treeSize, err = sumAllSubdirs(repo, c.snap2.Tree, string(filepath.Separator), c.map2)
	if err != nil {
		return err
	}
	Verbosef("total size: of %s: %s\r\n", c.snap2.ID(), formatBytes(treeSize))

	return nil
}

func (c *comparison) compareTrees() {
	// map2 should be the more recent snapshot

	// iterate over more recent snapshot
	for path, _ := range c.map2 {
		change, sign, present := c.comparePath(path)
		if present {
			if change != 0 {
				Verbosef("%s: change: %s%s (%d)\r\n", path, sign, formatBytes(change), change)
			}
		} else {
			Verbosef("%s: [missing]\r\n", path)
		}
	}
}

func (c* comparison) comparePath(path string) (change uint64, sign string, present bool) {
	size1, ok1 := c.map1[path]
	size2, ok2 := c.map2[path]

	if !ok1 || !ok2 {
		present = false
		return
	}

	present = true

	if size1 < size2 {
		change = size2 - size1
		sign = "+"
	} else if size2 > size1 {
		change = size1 - size2
		sign = "-"
	} else {
		change = 0
	}

	return
}

