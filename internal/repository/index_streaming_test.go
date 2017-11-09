package repository

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/restic/restic/internal/restic"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"
)

func GenerateBigIndex(t *testing.T) {
	idx := NewIndex()

	// create 50 packs with 20 blobs each
	for i := 0; i < 300000; i++ {
		packID := restic.NewRandomID()

		pos := uint(0)
		for j := 0; j < 20; j++ {
			id := restic.NewRandomID()
			length := uint(i*100 + j)
			idx.Store(restic.PackedBlob{
				Blob: restic.Blob{
					Type:   restic.DataBlob,
					ID:     id,
					Offset: pos,
					Length: length,
				},
				PackID: packID,
			})

			pos += length
		}
	}

	wr := bytes.NewBuffer(nil)
	err := idx.Encode(wr)

	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Create("big_index.json.gz")
	if err != nil {
		t.Fatal(err)
	}

	w := gzip.NewWriter(f)

	// Write bytes in compressed form to the file.
	w.Write(wr.Bytes())

	// Close the file.
	w.Close()

	fmt.Printf("updated big_index.json.gz\n")
}

func runMemoryLogger() {
	go func() {
		for {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			fmt.Printf("\nAlloc = %v\nTotalAlloc = %v\nSys = %v\nNumGC = %v\nHeapInUse = %v\nHeapSys = %v\n\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC, m.HeapInuse/1024, m.HeapSys/1024)
			time.Sleep(500 * time.Millisecond)
		}
	}()
}

func TestDecodeIndexStreaming(t *testing.T) {
	s := `
{
  "supersedes": [
    "0001020304050607080900010203040506070809000102030405060708090102",
    "0001020304050607080900010203040506070809000102030405060708090102"
  ],
  "packs": [
    {
      "id": "783445db3ed3da35ca83264ff9b264dd38cfd135c4779437ba0297f10752d64d",
      "blobs": [
        {
          "id": "e810053e2fa32f4666666b1c29cca8215d5590647c5cf667ebbff926796e27fd",
          "type": "data",
          "offset": 0,
          "length": 0
        },
        {
          "id": "069b6da585e8f1383504d47faa7dbf6a682757c26416c47ff19b064068991053",
          "type": "data",
          "offset": 0,
          "length": 1
        }
      ]
    },
    {
      "id": "d72f55d5266f2ff0197296488ba79cb30ca53a4cb9b8813c17a5820a106f28f0",
      "blobs": [
        {
          "id": "466b6b96b9c82e8b5f279116986bbd4c3b133cabcbe2d025142893c29850b7a7",
          "type": "data",
          "offset": 0,
          "length": 100
        },
        {
          "id": "c39a781b20677b98248fa580752436edce6e7f5132edc13b1f11035426af7576",
          "type": "data",
          "offset": 100,
          "length": 101
        }
      ]
    }
  ]
}
`

	rd := bytes.NewReader([]byte(s))

	index, err := DecodeIndexStreaming(rd)
	if err != nil {
		t.Fatalf("error decoding json from string: %v", err)
	}

	fmt.Printf("index has %d packs\n", index.Count(restic.DataBlob))
	fmt.Printf("index Supersedes: %+v\n", index.Supersedes())
}

// run like: GODEBUG=memprofilerate=1 go test -run=^$ -bench=^BenchmarkDecodeIndexStreamingBig$ -memprofile mem.prof && go tool pprof -top -cum mem.prof
func BenchmarkDecodeIndexStreamingBig(b *testing.B) {
	runMemoryLogger()

	fmt.Printf("running BenchmarkDecodeIndexStreamingBig\n")
	b.ResetTimer()

	f, err := os.Open("big_index.json.gz")
	if err != nil {
		b.Fatalf("error opening big_index.json: %v", err)
	}

	rd, err := gzip.NewReader(f)
	if err != nil {
		b.Fatalf("error unzipping stream: %v", err)
	}

	index, err := DecodeIndexStreaming(rd)
	if err != nil {
		b.Fatalf("error decoding big_index.json: %v", err)
	}

	fmt.Printf("index has %d packs\n", index.Count(restic.DataBlob))

	rd.Close()
	f.Close()
	//b.ReportAllocs()
}

func BenchmarkDecodeIndexBig(b *testing.B) {
	runMemoryLogger()

	fmt.Printf("running BenchmarkDecodeIndexBig\n")

	//b.ResetTimer()

	f, err := os.Open("big_index.json.gz")
	if err != nil {
		b.Fatalf("error opening big_index.json: %v", err)
	}

	rd, err := gzip.NewReader(f)
	if err != nil {
		b.Fatalf("error unzipping stream: %v", err)
	}

	bytes, err := ioutil.ReadAll(rd)
	if err != nil {
		b.Fatalf("error reading from unzipped stream: %v", err)
	}

	fmt.Printf("read a total of %d bytes\n", len(bytes))

	index, err := DecodeIndex(bytes)
	if err != nil {
		b.Fatalf("error decoding big_index.json: %v", err)
	}

	fmt.Printf("index has %d packs\n", index.Count(restic.DataBlob))

	rd.Close()
	f.Close()
	//b.ReportAllocs()
}
