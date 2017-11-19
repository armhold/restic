package repository

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	"reflect"
	"testing"
)

func TestReservedReadCloser(t *testing.T) {
	prefixLen := 1024
	suffix := []byte("reserve0123")
	suffixLen := len(suffix)

	buf := make([]byte, prefixLen+len(suffix))
	_, err := io.ReadFull(rand.Reader, buf[:prefixLen])
	if err != nil {
		t.FailNow()
	}

	copy(buf[len(buf)-suffixLen:], suffix)

	bufferLengths := []int{1, 2, 3, 15, 16, 32, 64, 64, 65, 1024, 1025}

	for _, bl := range bufferLengths {
		var accum []byte

		// buffer for reads
		p := make([]byte, bl)

		rc := NewReservedReadCloser(ioutil.NopCloser(bytes.NewReader(buf)), len(suffix))

		for {
			n, err := rc.Read(p)
			if n > 0 {
				accum = append(accum, p[:n]...)
			}

			if err == io.EOF {
				break
			}
		}

		if len(accum) != prefixLen {
			t.Fatalf("len(accum) != prefixLen (%d != %d)", len(accum), prefixLen)
		}

		if !reflect.DeepEqual(accum, buf[:prefixLen]) {
			t.Fatalf("expected: %+v, got: %+v", buf[:prefixLen], accum)
		}

		if !reflect.DeepEqual(rc.Reserved(), suffix) {
			t.Fatalf("bl: %d, expected: %+v, got: %+v", bl, suffix, rc.Reserved())
		}
	}
}
