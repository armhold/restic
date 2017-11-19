package repository

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"github.com/restic/restic/internal/crypto"
	"github.com/restic/restic/internal/restic"
	"hash"
	"io"
)

type HashingReadCloser interface {
	io.ReadCloser
	restic.HashChecker
}

type sha256HashingReadCloser struct {
	orig     io.ReadCloser
	expected restic.ID
	hash     hash.Hash
	closed   bool
}

type CipherReader struct {
	r io.Reader
	c io.Closer
}

// ReadCloser that holds back the last N bytes from the stream
type ReservedReadCloser struct {
	r       *bufio.Reader
	c       io.Closer
	n       int
	reserve []byte
}

func NewReservedReadCloser(rc io.ReadCloser, n int) *ReservedReadCloser {
	buf := bufio.NewReaderSize(rc, 32)

	return &ReservedReadCloser{r: buf, c: rc, n: n, reserve: make([]byte, n)}
}

func (r *ReservedReadCloser) Read(p []byte) (int, error) {
	b, peekErr := r.r.Peek(len(p) + r.n)

	cutoff := len(b) - r.n
	if cutoff > 0 {
		copy(p, b[:cutoff])
		copy(r.reserve, b[cutoff:])

		if peekErr == io.EOF {
			r.r.Discard(len(b))
			return cutoff, io.EOF
		} else {
			r.r.Discard(cutoff)
			return cutoff, nil
		}
	}

	return 0, io.EOF
}

func (r *ReservedReadCloser) Close() error {
	return r.c.Close()
}

func (r *ReservedReadCloser) Reserved() []byte {
	return r.reserve
}

func NewCipherReader(nonce []byte, key crypto.EncryptionKey, rc io.ReadCloser) *CipherReader {
	c, err := aes.NewCipher(key[:])
	if err != nil {
		panic(fmt.Sprintf("unable to create cipher: %v", err))
	}
	stream := cipher.NewCTR(c, nonce)
	sr := &cipher.StreamReader{S: stream, R: rc}

	return &CipherReader{r: sr, c: rc}
}

func (c *CipherReader) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *CipherReader) Close() error {
	return c.c.Close()
}

func NewSha256HashingReadCloser(orig io.ReadCloser, id restic.ID) *sha256HashingReadCloser {
	return &sha256HashingReadCloser{orig: orig, expected: id, hash: sha256.New()}
}

func (h *sha256HashingReadCloser) Read(p []byte) (int, error) {
	n, err := h.orig.Read(p)
	if err != nil {
		h.hash.Write(p)
	}

	return n, err
}

func (h *sha256HashingReadCloser) Close() error {
	err := h.orig.Close()
	h.closed = true

	return err
}

func (h *sha256HashingReadCloser) HashWasValid() bool {
	if !h.closed {
		panic("Hash() called before reader was closed")
	}

	var id restic.ID
	h.hash.Sum(id[:])

	return id == h.expected
}

// always valid- performs no hashing (use for config files)
type noOpHashingReadCloser struct {
	io.ReadCloser
}

func (h *noOpHashingReadCloser) HashWasValid() bool {
	return true
}
