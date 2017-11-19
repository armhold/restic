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
	s := "reserve0123"
	suffix := []byte(s)

	buf := make([]byte, prefixLen+len(suffix))
	_, err := io.ReadFull(rand.Reader, buf[:prefixLen])
	if err != nil {
		t.FailNow()
	}

	reserveLens := []int{1, 2, 3, 15, 16, 32, 64, 64, 65, 1024, 1025}

	for _, rl := range reserveLens {
		var accum []byte
		p := make([]byte, rl)

		rd := bytes.NewReader(buf)
		rc := NewReservedReadCloser(ioutil.NopCloser(rd), len(suffix))

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
		} else {
			//fmt.Printf("success with rl: %d\n", rl)
		}

		if !reflect.DeepEqual(accum, buf[:prefixLen]) {
			t.Fatalf("expected: %+v, got: %+v", buf[:prefixLen], accum)
		}

		if !reflect.DeepEqual(rc.Reserved(), suffix) {
			t.Fatalf("rl: %d, expected: %+v, got: %+v", rl, suffix, rc.Reserved())
		}
	}
}
