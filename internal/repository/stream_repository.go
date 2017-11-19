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

type sha256ReadCloser struct {
	rc       io.ReadCloser
	expected restic.ID
	hash     hash.Hash
	closed   bool
}

func NewSha256ReadCloser(rc io.ReadCloser, id restic.ID) *sha256ReadCloser {
	return &sha256ReadCloser{rc: rc, expected: id, hash: sha256.New()}
}

func (h *sha256ReadCloser) Read(p []byte) (int, error) {
	n, err := h.rc.Read(p)

	if n > 0 {
		h.hash.Write(p[:n])
	}

	return n, err
}

func (h *sha256ReadCloser) Close() error {
	h.closed = true
	return h.rc.Close()
}

func (h *sha256ReadCloser) HashWasValid() bool {
	if !h.closed {
		panic("Hash() called before reader was closed")
	}

	var id restic.ID
	sum := h.hash.Sum(nil)
	copy(id[:], sum)

	return id.Equal(h.expected)
}

// always valid- performs no hashing (use for config files)
type nopHashingReadCloser struct {
	io.ReadCloser
}

func NewNopHashingReadCloser(rc io.ReadCloser) *nopHashingReadCloser {
	return &nopHashingReadCloser{ReadCloser: rc}
}

func (h *nopHashingReadCloser) HashWasValid() bool {
	return true
}

// streams plaintext from the underlying aes-encrypted stream
type cipherReadCloser struct {
	io.Reader
	io.Closer
}

func NewCipherReader(nonce []byte, key crypto.EncryptionKey, rc io.ReadCloser) *cipherReadCloser {
	c, err := aes.NewCipher(key[:])
	if err != nil {
		panic(fmt.Sprintf("unable to create cipher: %v", err))
	}
	stream := cipher.NewCTR(c, nonce)
	sr := &cipher.StreamReader{S: stream, R: rc}

	return &cipherReadCloser{Reader: sr, Closer: rc}
}

// ReadCloser that holds back the last n bytes from the stream
type ReservedReadCloser struct {
	r *bufio.Reader
	io.Closer
	n       int
	reserve []byte
}

func NewReservedReadCloser(rc io.ReadCloser, n int) *ReservedReadCloser {
	buf := bufio.NewReaderSize(rc, 32)

	return &ReservedReadCloser{r: buf, Closer: rc, n: n, reserve: make([]byte, n)}
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
	} else {
		r.r.Discard(len(b))
	}

	return 0, io.EOF
}

func (r *ReservedReadCloser) Reserved() []byte {
	return r.reserve
}

type HashingReadCloser interface {
	io.ReadCloser
	restic.HashChecker
}
