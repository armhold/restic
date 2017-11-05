package main

import (
	"context"
	"github.com/restic/restic/internal/crypto"
	"github.com/restic/restic/internal/restic"
	"sync"
)

type SynchronizedRepo struct {
	sync.Mutex

	repo restic.Repository
}

func (s *SynchronizedRepo) Backend() restic.Backend {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.Backend()
}

func (s *SynchronizedRepo) Key() *crypto.Key {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.Key()
}

func (s *SynchronizedRepo) SetIndex(idx restic.Index) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	s.repo.SetIndex(idx)
}

func (s *SynchronizedRepo) Index() restic.Index {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.Index()
}

func (s *SynchronizedRepo) SaveFullIndex(ctx context.Context) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.SaveFullIndex(ctx)
}

func (s *SynchronizedRepo) SaveIndex(ctx context.Context) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.SaveIndex(ctx)
}

func (s *SynchronizedRepo) LoadIndex(ctx context.Context) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.LoadIndex(ctx)
}

func (s *SynchronizedRepo) Config() restic.Config {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.Config()
}

func (s *SynchronizedRepo) LookupBlobSize(id restic.ID, blobType restic.BlobType) (uint, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.LookupBlobSize(id, blobType)
}

func (s *SynchronizedRepo) List(ctx context.Context, ft restic.FileType) <-chan restic.ID {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.List(ctx, ft)
}

func (s *SynchronizedRepo) ListPack(ctx context.Context, id restic.ID) ([]restic.Blob, int64, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.ListPack(ctx, id)
}

func (s *SynchronizedRepo) Flush() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.Flush()
}

func (s *SynchronizedRepo) SaveUnpacked(ctx context.Context, ft restic.FileType, b []byte) (restic.ID, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.SaveUnpacked(ctx, ft, b)
}

func (s *SynchronizedRepo) SaveJSONUnpacked(ctx context.Context, ft restic.FileType, item interface{}) (restic.ID, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.SaveJSONUnpacked(ctx, ft, item)
}

func (s *SynchronizedRepo) LoadJSONUnpacked(ctx context.Context, ft restic.FileType, id restic.ID, item interface{}) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.LoadJSONUnpacked(ctx, ft, id, item)
}

func (s *SynchronizedRepo) LoadAndDecrypt(ctx context.Context, ft restic.FileType, id restic.ID) ([]byte, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.LoadAndDecrypt(ctx, ft, id)
}

func (s *SynchronizedRepo) LoadBlob(ctx context.Context, blobType restic.BlobType, id restic.ID, b []byte) (int, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.LoadBlob(ctx, blobType, id, b)
}

func (s *SynchronizedRepo) SaveBlob(ctx context.Context, blobType restic.BlobType, b []byte, id restic.ID) (restic.ID, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.SaveBlob(ctx, blobType, b, id)
}

func (s *SynchronizedRepo) LoadTree(ctx context.Context, id restic.ID) (*restic.Tree, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.LoadTree(ctx, id)
}

func (s *SynchronizedRepo) SaveTree(ctx context.Context, tree *restic.Tree) (restic.ID, error) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.repo.SaveTree(ctx, tree)
}
