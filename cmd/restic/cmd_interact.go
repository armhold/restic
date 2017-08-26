package main

import (
	"context"

	"github.com/restic/restic/internal/debug"
	"github.com/restic/restic/internal/errors"

	"bytes"
	"fmt"
	"github.com/restic/restic/internal/filter"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
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

// since a restic.Tree does not know its own name, we must remember it here as we descend the tree
type directory struct {
	Name string
	Id   *restic.ID
	Tree *restic.Tree
}


// path user has currently navigated down
var dirStack []directory

// files/paths user has selected for restore
var addedFiles map[string]bool = make(map[string]bool)

var interactOptions InteractOptions

var term *terminal.Terminal

var currSnapshot *restic.Snapshot

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

	currSnapshot, err = restic.LoadSnapshot(context.TODO(), repo, snapshotID)
	if err != nil {
		return fmt.Errorf("could not load snapshot %q: %v\n", snapshotID, err)
	}

	if err = loadRootDir(repo, currSnapshot.Tree); err != nil {
		return err
	}

	printDirectory(currDir())

	err = readCommands(ctx, opts, repo, snapshotID)

	// TODO: unsure when to return err vs when to Exitf() in this func

	return err
}

func readCommands(ctx context.Context, opts InteractOptions, repo *repository.Repository, snapshotID restic.ID) error {
	prompt := "restore>"

	// put stdin into raw mode so we can can unbuffered chars for autocompletion
	stdinFd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(stdinFd)
	if err != nil {
		return err
	}
	defer terminal.Restore(stdinFd, oldState)

	var screen = struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}

	term = terminal.NewTerminal(screen, prompt)
	term.AutoCompleteCallback = autoCompleteCallback

	for {
		line, err := term.ReadLine()
		if err != nil {
			break
		}

		cmd, args, err := parseCmd(line)
		if err != nil {
			screenPrintf("got error: %v", err)
			continue
		}

		switch cmd {
		case "add":
			addFiles(args)

		case "del":
			delFiles(args)

		case "compare":
			compareToSnapshotId := args
			err := compareToSnapshot(repo, compareToSnapshotId)
			if err != nil {
				screenPrintf("%s\r\n", err.Error())
			}

		case "cd":
			if args == ".." {
				popDir()
			} else {
				if err := pushDir(repo, args); err != nil {
					fmt.Println(err)
				}
			}
			pwd()

		case "pwd":
			pwd()

		case "ls":
			// TODO: check if args != nil, and ls the given dir
			printDirectory(currDir())

		case "extract":
			screenPrintf("extract files")
			extractFiles(ctx, opts, repo, snapshotID)

		case "done":
			screenPrintf("done, returning")
			return nil

		case "":
			// nothing

		case "snapshots":
			snapshots()

		case "load":
			loadSnapshot(args)

		default:
			screenPrintf("unrecognized command: %s", cmd)
		}

		term.SetPrompt(prompt)
	}

	if err != nil {
		screenPrintf("got error: %s", err)
	}

	return err
}

func snapshots() {
	screenPrintf("list snapshots")
}

func loadSnapshot(args string) {
	snapId := args
	screenPrintf("load snapshot id: %s", snapId)
}

func compareToSnapshot(repo *repository.Repository, compareToSnapshotId string) (error) {
	screenPrintf("compare current snapshot to %s", compareToSnapshotId)

	snapID, err := restic.FindSnapshot(repo, compareToSnapshotId)
	if err != nil {
		return errors.Errorf("invalid id %q: %v", compareToSnapshotId, err)
	}

	Verbosef("found snapshot: %v\r\n", snapID)


	compareToSnapshot, err := restic.LoadSnapshot(context.TODO(), repo, snapID)
	if err != nil {
		return fmt.Errorf("could not load snapshot %q: %v\n", snapID, err)
	}

	c := &comparison{snap1: compareToSnapshot, snap2: currSnapshot}

	if err = c.doCompare(repo); err != nil {
		return err
	}

	c.compareTrees()

	return nil
}

func pwd() {
	screenPrintf(currPath())
}

func parseCmd(line string) (cmd, args string, err error) {
	cmdsWithArgs := []string{"add", "cd", "compare", "del", "load"}
	cmdsWithoutArgs := []string{"", "done", "exit", "extract", "ls", "pwd", "snapshots"}

	// TODO: ls should handle optional arg

	for _, c := range cmdsWithArgs {
		if line == c {
			err = errors.Errorf("%s: arguent missing", line)
			return
		}

		if strings.HasPrefix(line, c+" ") {
			cmd = c
			args = strings.TrimSpace(line[len(c)+1:])
			return
		}
	}

	for _, c := range cmdsWithoutArgs {
		if strings.TrimSpace(line) == c {
			cmd = c
			return
		}
	}

	err = errors.Errorf("no such command: %s", strings.TrimSpace(line))
	return
}

func autoCompleteCallback(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	if key != '\t' {
		return
	}

	// for now, just autocomplete the arg (file list).
	// TODO: autocomplete commands as well
	_, args, err := parseCmd(line)
	if err != nil {
		return
	}

	dir := currDir()
	var matches []string
	for _, entry := range dir.Tree.Nodes {
		if strings.HasPrefix(entry.Name, args) {
			matches = append(matches, entry.Name)
		}
	}

	if len(matches) > 1 {
		matchesFormatted := renderInColumns(matches)
		term.Write([]byte("\r\n"))
		term.Write([]byte(matchesFormatted))
		term.Write([]byte("\r\n"))

		// render the largest common prefix on the line
		newLine = line[0:len(line)-len(args)] + longestCommonPrefix(matches)

	} else if len(matches) == 1 {
		// just render the entire (single) match on the line
		newLine = line[0:len(line)-len(args)] + matches[0]
	} else {
		// beep ?
	}

	newPos = len(newLine)
	ok = true

	return
}

func renderInColumns(matches []string) string {
	var buf bytes.Buffer

	padding := 3
	//w := tabwriter.NewWriter(&buf, 0, 0, padding, '-', tabwriter.Debug)
	w := tabwriter.NewWriter(&buf, 0, 0, padding, ' ', 0)

	col := 0
	s := ""
	maxCols := 4

	for _, match := range matches {
		s += match + "\t"
		col++
		if col == maxCols {
			fmt.Fprintln(w, s)
			s = ""
			col = 0
		}
	}

	if s != "" {
		fmt.Fprintln(w, s)
	}

	w.Flush()

	return buf.String()
}

func longestCommonPrefix(words []string) string {
	if len(words) == 0 {
		return ""
	}

	maxOffsetToCheck := len(words[0])

	wordsInRunes := make([][]rune, len(words))
	for i, name := range words {
		wordsInRunes[i] = []rune(name)
		if len(wordsInRunes[i]) < maxOffsetToCheck {
			maxOffsetToCheck = len(wordsInRunes[i])
		}
	}

	max := 0
	for ; max < maxOffsetToCheck; max++ {
		if !allSameAtOffset(wordsInRunes, max) {
			break
		}
	}

	return string(wordsInRunes[0][:max])
}

func allSameAtOffset(words [][]rune, offset int) bool {
	r := words[0][offset]

	for _, runes := range words {
		if runes[offset] != r {
			return false
		}
	}

	return true
}

// load the root of the snapshot
func loadRootDir(repo *repository.Repository, id *restic.ID) error {
	tree, err := repo.LoadTree(context.TODO(), *id)
	if err != nil {
		return err
	}

	// assumes treeStack is initially empty
	dirStack = append(dirStack, directory{Name: "/", Tree: tree, Id: id})

	return nil
}

func pushDir(repo *repository.Repository, dir string) error {
	id, tree, err := findSubdirByName(repo, currDir().Tree, dir)
	if err != nil {
		return err
	}

	dirStack = append(dirStack, directory{Name: dir, Tree: tree, Id: id})

	return nil
}

// in raw mode, need to add carriage return
func screenPrintf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\r\n", args...)
}

func printDirectory(dir directory) {
	prefix := ""

	for _, entry := range dir.Tree.Nodes {
		screenPrintf("%s", formatNodeInteractive(prefix, entry, lsOptions.ListLong))
	}
}

func formatNodeInteractive(prefix string, n *restic.Node, long bool) string {
	if !long {
		result := filepath.Join(prefix, n.Name)

		if n.Type == "dir" {
			result += string(filepath.Separator)
		}
		return result
	}

	switch n.Type {
	case "file":
		return fmt.Sprintf("%s %5d %5d %6d %s %s",
			n.Mode, n.UID, n.GID, n.Size, n.ModTime.Format(TimeFormat), filepath.Join(prefix, n.Name))
	case "dir":
		return fmt.Sprintf("%s %5d %5d %6d %s %s",
			n.Mode|os.ModeDir, n.UID, n.GID, n.Size, n.ModTime.Format(TimeFormat), filepath.Join(prefix, n.Name))
	case "symlink":
		return fmt.Sprintf("%s %5d %5d %6d %s %s -> %s",
			n.Mode|os.ModeSymlink, n.UID, n.GID, n.Size, n.ModTime.Format(TimeFormat), filepath.Join(prefix, n.Name), n.LinkTarget)
	default:
		return fmt.Sprintf("<Node(%s) %s>", n.Type, n.Name)
	}
}

func popDir() {
	if len(dirStack) > 1 {
		dirStack = dirStack[:len(dirStack)-1]
	}
}

func currDir() directory {
	return dirStack[len(dirStack)-1]
}

func currPath() string {
	var paths []string

	for _, dir := range dirStack {
		paths = append(paths, dir.Name)
	}

	return path.Join(paths...)
}

func addFiles(pattern string) {
	// TODO: validate files/paths exist in index

	// TODO: should we expand wildcards into discrete filepaths during "add", or expand later in the extractFiles() selector?

	if pattern[0] != filepath.Separator {
		// resolve pattern relative to currentPath
		pattern = currPath() + string(filepath.Separator) + pattern
	}

	addedFiles[pattern] = true

	screenPrintf("added: %s", pattern)
}

func delFiles(pattern string) {
	// TODO: validate files exist in index, from added files

	if pattern[0] != filepath.Separator {
		// resolve pattern relative to currentPath
		pattern = currPath() + string(filepath.Separator) + pattern
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

func findSubdirByName(repo *repository.Repository, currTree *restic.Tree, subdirName string) (id *restic.ID, tree *restic.Tree, err error) {

	for _, entry := range currTree.Nodes {
		if entry.Type == "dir" && entry.Subtree != nil && entry.Name == subdirName {
			id = entry.Subtree
			tree, err = repo.LoadTree(context.TODO(), *id)
			return
		}
	}

	err = errors.Errorf("no such subdir: \"%s\"", subdirName)
	return
}
