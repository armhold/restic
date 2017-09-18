package web

import (
	"io"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"fmt"
)

const (
	cookieName = "RESTIC_SESSION_ID"
)

type Manager struct {
	// zero-value is valid. map must not be copied.
	sessions sync.Map
}

type Session struct {
	// zero-value is valid. map must not be copied.
	values sync.Map
	sid    string
}

func (s *Session) Get(k interface{}) (interface{}, bool) {
	return s.values.Load(k)
}

func (s *Session) Set(k, v interface{}) {
	s.values.Store(k, v)
}

func (m *Manager) NewSession() *Session {
	s := &Session{sid: sessionId()}
	m.sessions.Store(s.sid, s)

	fmt.Printf("created new session: %s\n", s.sid)

	return s
}

func (m *Manager) GetSession(sid string) (result *Session, isNew bool) {
	i, ok := m.sessions.Load(sid)

	if ok {
		result = i.(*Session)
		isNew = false
	} else {
		result = m.NewSession()
		m.sessions.Store(sid, result)
		isNew = true
	}

	return
}

func sessionId() string {
	b := make([]byte, 32)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(b)
}
