package main

import (
	"context"
	"github.com/restic/restic/internal/crypto"
	"github.com/restic/restic/internal/restic"
	"github.com/restic/restic/internal/web"
	"github.com/spf13/cobra"
	"sync"
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

type SynchronizedRepo struct {
	sync.Mutex

	repo restic.Repository
}

func (s *SynchronizedRepo) Backend() restic.Backend {
	return s.repo.Backend()
}

func (s *SynchronizedRepo) Key() *crypto.Key {
	return s.repo.Key()
}

func (s *SynchronizedRepo) SetIndex(idx restic.Index) {
	s.repo.SetIndex(idx)
}

func (s *SynchronizedRepo) Index() restic.Index {
	return s.repo.Index()
}

func (s *SynchronizedRepo) SaveFullIndex(ctx context.Context) error {
	return s.repo.SaveFullIndex(ctx)
}

func (s *SynchronizedRepo) SaveIndex(ctx context.Context) error {
	return s.repo.SaveIndex(ctx)
}

func (s *SynchronizedRepo) LoadIndex(ctx context.Context) error {
	return s.repo.LoadIndex(ctx)
}

func (s *SynchronizedRepo) Config() restic.Config {
	return s.repo.Config()
}

func (s *SynchronizedRepo) LookupBlobSize(id restic.ID, blobType restic.BlobType) (uint, error) {
	return s.repo.LookupBlobSize(id, blobType)
}

func (s *SynchronizedRepo) List(ctx context.Context, ft restic.FileType) <-chan restic.ID {
	return s.repo.List(ctx, ft)
}

func (s *SynchronizedRepo) ListPack(ctx context.Context, id restic.ID) ([]restic.Blob, int64, error) {
	return s.repo.ListPack(ctx, id)
}

func (s *SynchronizedRepo) Flush() error {
	return s.repo.Flush()
}

func (s *SynchronizedRepo) SaveUnpacked(ctx context.Context, ft restic.FileType, b []byte) (restic.ID, error) {
	return s.repo.SaveUnpacked(ctx, ft, b)
}

func (s *SynchronizedRepo) SaveJSONUnpacked(ctx context.Context, ft restic.FileType, item interface{}) (restic.ID, error) {
	return s.repo.SaveJSONUnpacked(ctx, ft, item)
}

func (s *SynchronizedRepo) LoadJSONUnpacked(ctx context.Context, ft restic.FileType, id restic.ID, item interface{}) error {
	return s.repo.LoadJSONUnpacked(ctx, ft, id, item)
}

func (s *SynchronizedRepo) LoadAndDecrypt(ctx context.Context, ft restic.FileType, id restic.ID) ([]byte, error) {
	return s.repo.LoadAndDecrypt(ctx, ft, id)
}

func (s *SynchronizedRepo) LoadBlob(ctx context.Context, blobType restic.BlobType, id restic.ID, b []byte) (int, error) {
	return s.repo.LoadBlob(ctx, blobType, id, b)
}

func (s *SynchronizedRepo) SaveBlob(ctx context.Context, blobType restic.BlobType, b []byte, id restic.ID) (restic.ID, error) {
	return s.repo.SaveBlob(ctx, blobType, b, id)
}

func (s *SynchronizedRepo) LoadTree(ctx context.Context, id restic.ID) (*restic.Tree, error) {
	return s.LoadTree(ctx, id)
}

func (s *SynchronizedRepo) SaveTree(ctx context.Context, tree *restic.Tree) (restic.ID, error) {
	return s.repo.SaveTree(ctx, tree)
}
