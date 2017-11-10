package index

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/restic/restic/internal/checker"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
	"github.com/restic/restic/internal/test"
	"io"
	"runtime"
)

var (
	snapshotTime = time.Unix(1470492820, 207401672)
	depth        = 3
)

func createFilledRepo(t testing.TB, snapshots int, dup float32) (restic.Repository, func()) {
	repo, cleanup := repository.TestRepository(t)

	for i := 0; i < 3; i++ {
		restic.TestCreateSnapshot(t, repo, snapshotTime.Add(time.Duration(i)*time.Second), depth, dup)
	}

	return repo, cleanup
}

func validateIndex(t testing.TB, repo restic.Repository, idx *Index) {
	for id := range repo.List(context.TODO(), restic.DataFile) {
		p, ok := idx.Packs[id]
		if !ok {
			t.Errorf("pack %v missing from index", id.Str())
		}

		if !p.ID.Equal(id) {
			t.Errorf("pack %v has invalid ID: want %v, got %v", id.Str(), id, p.ID)
		}
	}
}

func TestIndexNew(t *testing.T) {
	repo, cleanup := createFilledRepo(t, 3, 0)
	defer cleanup()

	idx, _, err := New(context.TODO(), repo, restic.NewIDSet(), nil)
	if err != nil {
		t.Fatalf("New() returned error %v", err)
	}

	if idx == nil {
		t.Fatalf("New() returned nil index")
	}

	validateIndex(t, repo, idx)
}

func TestIndexLoad(t *testing.T) {
	repo, cleanup := createFilledRepo(t, 3, 0)
	defer cleanup()

	loadIdx, err := Load(context.TODO(), repo, nil)
	if err != nil {
		t.Fatalf("Load() returned error %v", err)
	}

	if loadIdx == nil {
		t.Fatalf("Load() returned nil index")
	}

	validateIndex(t, repo, loadIdx)

	newIdx, _, err := New(context.TODO(), repo, restic.NewIDSet(), nil)
	if err != nil {
		t.Fatalf("New() returned error %v", err)
	}

	if len(loadIdx.Packs) != len(newIdx.Packs) {
		t.Errorf("number of packs does not match: want %v, got %v",
			len(loadIdx.Packs), len(newIdx.Packs))
	}

	validateIndex(t, repo, newIdx)

	for packID, packNew := range newIdx.Packs {
		packLoad, ok := loadIdx.Packs[packID]

		if !ok {
			t.Errorf("loaded index does not list pack %v", packID.Str())
			continue
		}

		if len(packNew.Entries) != len(packLoad.Entries) {
			t.Errorf("  number of entries in pack %v does not match: %d != %d\n  %v\n  %v",
				packID.Str(), len(packNew.Entries), len(packLoad.Entries),
				packNew.Entries, packLoad.Entries)
			continue
		}

		for _, entryNew := range packNew.Entries {
			found := false
			for _, entryLoad := range packLoad.Entries {
				if !entryLoad.ID.Equal(entryNew.ID) {
					continue
				}

				if entryLoad.Type != entryNew.Type {
					continue
				}

				if entryLoad.Offset != entryNew.Offset {
					continue
				}

				if entryLoad.Length != entryNew.Length {
					continue
				}

				found = true
				break
			}

			if !found {
				t.Errorf("blob not found in loaded index: %v", entryNew)
			}
		}
	}
}

func BenchmarkIndexNew(b *testing.B) {
	repo, cleanup := createFilledRepo(b, 3, 0)
	defer cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx, _, err := New(context.TODO(), repo, restic.NewIDSet(), nil)

		if err != nil {
			b.Fatalf("New() returned error %v", err)
		}

		if idx == nil {
			b.Fatalf("New() returned nil index")
		}
		b.Logf("idx %v packs", len(idx.Packs))
	}
}

func BenchmarkIndexSave(b *testing.B) {
	repo, cleanup := repository.TestRepository(b)
	defer cleanup()

	idx, _, err := New(context.TODO(), repo, restic.NewIDSet(), nil)
	test.OK(b, err)

	for i := 0; i < 8000; i++ {
		entries := make([]restic.Blob, 0, 200)
		for j := 0; j < cap(entries); j++ {
			entries = append(entries, restic.Blob{
				ID:     restic.NewRandomID(),
				Length: 1000,
				Offset: 5,
				Type:   restic.DataBlob,
			})
		}

		idx.AddPack(restic.NewRandomID(), 10000, entries)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		id, err := idx.Save(context.TODO(), repo, nil)
		if err != nil {
			b.Fatalf("New() returned error %v", err)
		}

		b.Logf("saved as %v", id.Str())
	}
}

func TestIndexDuplicateBlobs(t *testing.T) {
	repo, cleanup := createFilledRepo(t, 3, 0.01)
	defer cleanup()

	idx, _, err := New(context.TODO(), repo, restic.NewIDSet(), nil)
	if err != nil {
		t.Fatal(err)
	}

	dups := idx.DuplicateBlobs()
	if len(dups) == 0 {
		t.Errorf("no duplicate blobs found")
	}
	t.Logf("%d packs, %d duplicate blobs", len(idx.Packs), len(dups))

	packs := idx.PacksForBlobs(dups)
	if len(packs) == 0 {
		t.Errorf("no packs with duplicate blobs found")
	}
	t.Logf("%d packs with duplicate blobs", len(packs))
}

func loadIndex(t testing.TB, repo restic.Repository) *Index {
	idx, err := Load(context.TODO(), repo, nil)
	if err != nil {
		t.Fatalf("Load() returned error %v", err)
	}

	return idx
}

func TestSave(t *testing.T) {
	repo, cleanup := createFilledRepo(t, 3, 0)
	defer cleanup()

	idx := loadIndex(t, repo)

	packs := make(map[restic.ID][]restic.Blob)
	for id := range idx.Packs {
		if rand.Float32() < 0.5 {
			packs[id] = idx.Packs[id].Entries
		}
	}

	t.Logf("save %d/%d packs in a new index\n", len(packs), len(idx.Packs))

	id, err := Save(context.TODO(), repo, packs, idx.IndexIDs.List())
	if err != nil {
		t.Fatalf("unable to save new index: %v", err)
	}

	t.Logf("new index saved as %v", id.Str())

	for id := range idx.IndexIDs {
		t.Logf("remove index %v", id.Str())
		h := restic.Handle{Type: restic.IndexFile, Name: id.String()}
		err = repo.Backend().Remove(context.TODO(), h)
		if err != nil {
			t.Errorf("error removing index %v: %v", id, err)
		}
	}

	idx2 := loadIndex(t, repo)
	t.Logf("load new index with %d packs", len(idx2.Packs))

	if len(idx2.Packs) != len(packs) {
		t.Errorf("wrong number of packs in new index, want %d, got %d", len(packs), len(idx2.Packs))
	}

	for id := range packs {
		if _, ok := idx2.Packs[id]; !ok {
			t.Errorf("pack %v is not contained in new index", id.Str())
		}
	}

	for id := range idx2.Packs {
		if _, ok := packs[id]; !ok {
			t.Errorf("pack %v is not contained in new index", id.Str())
		}
	}
}

func TestIndexSave(t *testing.T) {
	repo, cleanup := createFilledRepo(t, 3, 0)
	defer cleanup()

	idx := loadIndex(t, repo)

	id, err := idx.Save(context.TODO(), repo, idx.IndexIDs.List())
	if err != nil {
		t.Fatalf("unable to save new index: %v", err)
	}

	t.Logf("new index saved as %v", id.Str())

	for id := range idx.IndexIDs {
		t.Logf("remove index %v", id.Str())
		h := restic.Handle{Type: restic.IndexFile, Name: id.String()}
		err = repo.Backend().Remove(context.TODO(), h)
		if err != nil {
			t.Errorf("error removing index %v: %v", id, err)
		}
	}

	idx2 := loadIndex(t, repo)
	t.Logf("load new index with %d packs", len(idx2.Packs))

	checker := checker.New(repo)
	hints, errs := checker.LoadIndex(context.TODO())
	for _, h := range hints {
		t.Logf("hint: %v\n", h)
	}

	for _, err := range errs {
		t.Errorf("checker found error: %v", err)
	}
}

func TestIndexAddRemovePack(t *testing.T) {
	repo, cleanup := createFilledRepo(t, 3, 0)
	defer cleanup()

	idx, err := Load(context.TODO(), repo, nil)
	if err != nil {
		t.Fatalf("Load() returned error %v", err)
	}

	packID := <-repo.List(context.TODO(), restic.DataFile)

	t.Logf("selected pack %v", packID.Str())

	blobs := idx.Packs[packID].Entries

	idx.RemovePack(packID)

	if _, ok := idx.Packs[packID]; ok {
		t.Errorf("removed pack %v found in index.Packs", packID.Str())
	}

	for _, blob := range blobs {
		h := restic.BlobHandle{ID: blob.ID, Type: blob.Type}
		_, err := idx.FindBlob(h)
		if err == nil {
			t.Errorf("removed blob %v found in index", h)
		}
	}
}

// example index serialization from doc/Design.rst
var docExample = []byte(`
{
  "supersedes": [
	"ed54ae36197f4745ebc4b54d10e0f623eaaaedd03013eb7ae90df881b7781452"
  ],
  "packs": [
	{
	  "id": "73d04e6125cf3c28a299cc2f3cca3b78ceac396e4fcf9575e34536b26782413c",
	  "blobs": [
		{
		  "id": "3ec79977ef0cf5de7b08cd12b874cd0f62bbaf7f07f3497a5b1bbcc8cb39b1ce",
		  "type": "data",
		  "offset": 0,
		  "length": 25
		},{
		  "id": "9ccb846e60d90d4eb915848add7aa7ea1e4bbabfc60e573db9f7bfb2789afbae",
		  "type": "tree",
		  "offset": 38,
		  "length": 100
		},
		{
		  "id": "d3dc577b4ffd38cc4b32122cabf8655a0223ed22edfd93b353dc0c3f2b0fdf66",
		  "type": "data",
		  "offset": 150,
		  "length": 123
		}
	  ]
	}
  ]
}
`)

func TestIndexLoadDocReference(t *testing.T) {
	repo, cleanup := repository.TestRepository(t)
	defer cleanup()

	id, err := repo.SaveUnpacked(context.TODO(), restic.IndexFile, docExample)
	if err != nil {
		t.Fatalf("SaveUnpacked() returned error %v", err)
	}

	t.Logf("index saved as %v", id.Str())

	idx := loadIndex(t, repo)

	blobID := restic.TestParseID("d3dc577b4ffd38cc4b32122cabf8655a0223ed22edfd93b353dc0c3f2b0fdf66")
	locs, err := idx.FindBlob(restic.BlobHandle{ID: blobID, Type: restic.DataBlob})
	if err != nil {
		t.Errorf("FindBlob() returned error %v", err)
	}

	if len(locs) != 1 {
		t.Errorf("blob found %d times, expected just one", len(locs))
	}

	l := locs[0]
	if !l.ID.Equal(blobID) {
		t.Errorf("blob IDs are not equal: %v != %v", l.ID, blobID)
	}

	if l.Type != restic.DataBlob {
		t.Errorf("want type %v, got %v", restic.DataBlob, l.Type)
	}

	if l.Offset != 150 {
		t.Errorf("wrong offset, want %d, got %v", 150, l.Offset)
	}

	if l.Length != 123 {
		t.Errorf("wrong length, want %d, got %v", 123, l.Length)
	}
}

// an io.Reader implementation that produces a streamable Index json
//
type jsonIndexReader struct {
	buf          bytes.Buffer
	packsCount   int
	packsWritten int

	inited     bool
	headerDone bool
	packsDone  bool
	sep        string
}

func NewJsonIndexReader(packsCount int) *jsonIndexReader {
	return &jsonIndexReader{packsCount: packsCount}
}

func (j *jsonIndexReader) Read(p []byte) (int, error) {
	if !j.inited {
		j.buf.WriteString(HEAD)
		j.inited = true
	}

	n, err := j.buf.Read(p)
	if err == io.EOF {
		if !j.headerDone {
			j.headerDone = true
			j.buf.Reset()

			// re-use same pack+blobs... technically not a valid index
			j.buf.WriteString(j.sep)
			j.buf.WriteString(PACK)
			j.sep = "    ,"
			j.packsWritten++
		} else if !j.packsDone {
			if j.packsWritten == j.packsCount {
				j.packsDone = true
				j.buf.Reset()
				j.buf.WriteString(TAIL)
			} else {
				j.buf.Reset()
				j.buf.WriteString(j.sep)
				j.buf.WriteString(PACK)
				j.packsWritten++
			}
		}

		if n < len(p) {
			c, err2 := j.buf.Read(p[n:])
			n += c
			err = err2
		}
	}

	return n, err
}

const HEAD = `
{
  "supersedes": [
	"ed54ae36197f4745ebc4b54d10e0f623eaaaedd03013eb7ae90df881b7781452"
  ],
  "packs": [
`

const PACK = `
	{
	  "id": "73d04e6125cf3c28a299cc2f3cca3b78ceac396e4fcf9575e34536b26782413c",
	  "blobs": [
		{
		  "id": "3ec79977ef0cf5de7b08cd12b874cd0f62bbaf7f07f3497a5b1bbcc8cb39b1ce",
		  "type": "data",
		  "offset": 0,
		  "length": 25
		},
	    {
		  "id": "9ccb846e60d90d4eb915848add7aa7ea1e4bbabfc60e573db9f7bfb2789afbae",
		  "type": "tree",
		  "offset": 38,
		  "length": 100
		},
		{
		  "id": "d3dc577b4ffd38cc4b32122cabf8655a0223ed22edfd93b353dc0c3f2b0fdf66",
		  "type": "data",
		  "offset": 150,
		  "length": 123
		}
	  ]
   }
`

const TAIL = `
  ]
}
`

func checkIndexJson(indexJSON *indexJSON, expectedPackCount int, t *testing.T) {

	supersedesId, err := restic.ParseID("ed54ae36197f4745ebc4b54d10e0f623eaaaedd03013eb7ae90df881b7781452")
	if err != nil {
		t.Fatal(err)
	}

	if len(indexJSON.Supersedes) != 1 {
		t.Fatalf("expected 1 element in Supercedes, got: %d", len(indexJSON.Supersedes))
	}

	if indexJSON.Supersedes[0] != supersedesId {
		t.Fatalf("expected: %v, got: %v", supersedesId, indexJSON.Supersedes[0])
	}

	if len(indexJSON.Packs) != expectedPackCount {
		t.Fatalf("expected %d elements in Packs, got: %d", expectedPackCount, len(indexJSON.Packs))
	}

	packId, err := restic.ParseID("73d04e6125cf3c28a299cc2f3cca3b78ceac396e4fcf9575e34536b26782413c")
	if err != nil {
		t.Fatal(err)
	}

	pack := indexJSON.Packs[0]
	if pack.ID != packId {
		t.Fatalf("expected: %v, got: %v", packId, pack.ID)
	}
}

func TestLoadIndexJSONStreaming(t *testing.T) {
	rd := NewJsonIndexReader(10)

	indexJSON, err := loadIndexJSONStreaming(rd)
	if err != nil {
		t.Fatal(err)
	}

	checkIndexJson(indexJSON, 10, t)
}

const giantPackCount = 2000000

// test performance of current index code which users json.Unmarshal
func TestLoadGiantIndexUnmarshal(t *testing.T) {
	runMemoryLogger()

	rd := NewJsonIndexReader(giantPackCount)
	buf := new(bytes.Buffer)
	buf.ReadFrom(rd)
	jsonString := buf.String()

	var indexJSON indexJSON
	json.Unmarshal([]byte(jsonString), &indexJSON)

	if len(indexJSON.Packs) != giantPackCount {
		t.Errorf("expected %d packs, got: %d", giantPackCount, len(indexJSON.Packs))
	}
}

// test performance using new streaming json parser
func TestLoadGiantIndexStreaming(t *testing.T) {
	runMemoryLogger()

	rd := NewJsonIndexReader(giantPackCount)

	indexJSON, err := loadIndexJSONStreaming(rd)
	if err != nil {
		t.Fatalf("error streaming bigJSON: %v", err)
	}

	if len(indexJSON.Packs) != giantPackCount {
		t.Errorf("expected %d packs, got: %d", giantPackCount, len(indexJSON.Packs))
	}
}

func runMemoryLogger() {
	go func() {
		var mem runtime.MemStats
		for {
			MB := float64(1 << 20)

			runtime.ReadMemStats(&mem)
			fmt.Printf("HeapSys    %.3f MiB\n", float64(mem.HeapSys)/MB)
			fmt.Printf("HeapInuse  %.3f MiB\n", float64(mem.HeapInuse)/MB)
			fmt.Println()
			time.Sleep(1000 * time.Millisecond)
		}
	}()
}
