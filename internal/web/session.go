package web

import (
	"io"
	"crypto/rand"
	"encoding/base64"
	"sync"
	"fmt"
	"net/url"
	"net/http"
	"time"
)

const (
	cookieName = "RESTIC_SESSION_ID"
	maxLifeTime = time.Hour / time.Second // time in seconds before client-side expiration. TODO: server-side GC of stale cookies
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

func (s *Session) Delete(k interface{}) {
	s.values.Delete(k)
}

func (m *Manager) GetOrCreateSession(w http.ResponseWriter, r *http.Request) (result *Session, isNew bool) {
	cookie, err := r.Cookie(cookieName)

	if err != nil || cookie.Value == "" {
		result = m.NewSession()
		isNew = true
		cookie := http.Cookie{Name: cookieName, Value: url.QueryEscape(result.sid), Path: "/", HttpOnly: true, MaxAge: int(maxLifeTime)}
		http.SetCookie(w, &cookie)
		fmt.Printf("created new cookie: %#v\n", cookie)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		result, isNew = m.GetSession(sid)
		fmt.Printf("found existing cookie: %#v\n", cookie)
	}

	return
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
