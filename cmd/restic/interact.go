package main

import (
	"github.com/restic/restic/internal/restic"
	"golang.org/x/crypto/ssh/terminal"
	"strings"
	"bytes"
	"path"
	"text/tabwriter"
	"github.com/restic/restic/internal/repository"
	"path/filepath"
	"os"
	"fmt"
	"context"
	"github.com/restic/restic/internal/errors"
	"io"

)

type Interact struct {
	ctx context.Context
	opts InteractOptions
	repo *repository.Repository
	snapshotID restic.ID

	args []string
	term *terminal.Terminal
	dirStack []directory
}


// since a restic.Tree does not know its own name, we must remember it here as we descend the tree
type directory struct {
	Name string
	Id   *restic.ID
	Tree *restic.Tree
}

// path user has currently navigated down


func NewInteract(ctx context.Context, opts InteractOptions, repo *repository.Repository, snapshotID restic.ID) (*Interact) {
	i := &Interact{ctx: ctx, opts: opts, repo: repo, snapshotID: snapshotID}

	return i
}

// load the root of the snapshot
func (i *Interact) loadRootDir(id *restic.ID) error {
	tree, err := i.repo.LoadTree(context.TODO(), *id)
	if err != nil {
		return err
	}

	// assumes treeStack is initially empty
	i.dirStack = append(i.dirStack, directory{Name: "/", Tree: tree, Id: id})

	return nil
}

func (i *Interact) readCommands() error {
	sn, err := restic.LoadSnapshot(context.TODO(), i.repo, i.snapshotID)
	if err != nil {
		return fmt.Errorf("could not load snapshot %q: %v\n", i.snapshotID, err)
	}

	if err := i.loadRootDir(sn.Tree); err != nil {
		return err
	}

	printDirectory(i.currDir())

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

	i.term = terminal.NewTerminal(screen, prompt)
	i.term.AutoCompleteCallback = i.autoCompleteCallback

	for {
		line, err := i.term.ReadLine()
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

		case "cd":
			if args == ".." {
				i.popDir()
			} else {
				if err := i.pushDir(i.repo, args); err != nil {
					fmt.Println(err)
				}
			}
			i.pwd()

		case "pwd":
			i.pwd()

		case "ls":
			// TODO: check if args != nil, and ls the given dir
			printDirectory(i.currDir())

		case "extract":
			screenPrintf("extract files")
			extractFiles(i.ctx, i.opts, i.repo, i.snapshotID)

		case "done":
			screenPrintf("done, returning")
			return nil

		case "":
		// nothing

		default:
			screenPrintf("unrecognized command: %s", cmd)
		}

		i.term.SetPrompt(prompt)
	}

	if err != nil {
		screenPrintf("got error: %s", err)
	}

	return err
}

func (i *Interact) popDir() {
	if len(i.dirStack) > 1 {
		i.dirStack = i.dirStack[:len(i.dirStack)-1]
	}
}

func (i *Interact) currDir() directory {
	return i.dirStack[len(i.dirStack)-1]
}

func (i *Interact) CurrDir() directory {
	return i.dirStack[len(i.dirStack)-1]
}

func (i *Interact) CurrPath() string {
	var paths []string

	for _, dir := range i.dirStack {
		paths = append(paths, dir.Name)
	}

	return path.Join(paths...)
}

func (i *Interact) pwd() {
	screenPrintf(i.CurrPath())
}

func (i *Interact) pushDir(repo *repository.Repository, dir string) error {
	id, tree, err := findSubdirByName(repo, i.currDir().Tree, dir)
	if err != nil {
		return err
	}

	i.dirStack = append(i.dirStack, directory{Name: dir, Tree: tree, Id: id})

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

func parseCmd(line string) (cmd, args string, err error) {
	cmdsWithArgs := []string{"add", "cd", "del", "ls"}
	cmdsWithoutArgs := []string{"", "done", "exit", "extract", "ls", "pwd"} // NB: "ls" appears in both

	for _, c := range cmdsWithArgs {
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


func (i *Interact) autoCompleteCallback(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	if key != '\t' {
		return
	}

	// for now, just autocomplete the arg (file list).
	// TODO: autocomplete commands as well
	_, args, err := parseCmd(line)
	if err != nil {
		return
	}

	dir := i.currDir()
	var matches []string
	for _, entry := range dir.Tree.Nodes {
		if strings.HasPrefix(entry.Name, args) {
			matches = append(matches, entry.Name)
		}
	}

	if len(matches) > 1 {
		matchesFormatted := renderInColumns(matches)
		i.term.Write([]byte("\r\n"))
		i.term.Write([]byte(matchesFormatted))
		i.term.Write([]byte("\r\n"))

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
