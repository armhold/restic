package web

import (
	"fmt"
	"testing"
)

func TestSessions(t *testing.T) {
	m := &Manager{}

	s1 := m.NewSession()
	s1.Set("foo", "bar")

	result, ok := s1.Get("foo")
	if !ok {
		t.Fail()
	}

	if result != "bar" {
		t.Errorf("expected \"bar\", got %s", result)
	}

	s2, isNew := m.GetSession(s1.sid)
	if s1.sid != s2.sid {
		t.Errorf("expected same sid, got %s != %s", s1.sid, s2.sid)
	}

	if isNew {
		t.Errorf("should not be new")
	}

	s1.Delete("foo")
	result, ok = s1.Get("foo")
	if ok {
		t.Errorf("\"ok\" should be false; failed to delete from s1")
	}

	if result != nil {
		t.Errorf("result not empty; failed to delete from s1: %s", result)
	}

	// now check s2; should be same results
	result, ok = s2.Get("foo")
	if ok {
		t.Errorf("\"ok\" should be false; failed to delete from s2")
	}

	if result != nil {
		t.Errorf("result not empty; failed to delete from s2: %s", result)
	}
}

func TestSessionsConcurrency(t *testing.T) {
	m := &Manager{}

	ch := make(chan bool)

	f := func(m *Manager) {
		s, _ := m.GetSession("abc123")

		for i := 0; i < 10000; i++ {
			k := fmt.Sprintf("key %d", i)
			v := fmt.Sprintf("value %d", i)
			s.Set(k, v)
		}

		ch <- true
	}

	go f(m)

	<-ch

	s, isNew := m.GetSession("abc123")
	if isNew {
		t.Fail()
	}

	for i := 0; i < 10000; i++ {
		k := fmt.Sprintf("key %d", i)
		expected := fmt.Sprintf("value %d", i)
		v, ok := s.Get(k)

		if !ok {
			t.Errorf("key not found for %d", i)
		}

		if v.(string) != expected {
			t.Errorf("expected %#v, got %#v", expected, v)
		}
	}
}
