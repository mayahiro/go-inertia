package inertia

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionFlashStoreBridgesDurableSession(t *testing.T) {
	session := &testFlashSession{values: map[string][]byte{}}
	store, err := NewSessionFlashStore(SessionFlashStoreConfig{
		Session: FlashSessionFunc(func(req *http.Request) (FlashSession, error) {
			return session, nil
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	renderer := newTestRenderer(t, Config{FlashStore: store})
	postReq := httptest.NewRequest(http.MethodPost, "/users", nil)
	postReq.Header.Set(HeaderInertia, "true")
	postW := httptest.NewRecorder()
	if err := renderer.Redirect(postW, postReq, "/users", WithFlash(Flash{"success": "created"})); err != nil {
		t.Fatal(err)
	}
	if _, ok := session.values[defaultSessionFlashKey]; !ok {
		t.Fatalf("flash data was not stored: %#v", session.values)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/users", nil)
	getReq.Header.Set(HeaderInertia, "true")
	getW := httptest.NewRecorder()
	if err := renderer.Render(getW, getReq, "Users/Index", Props{}); err != nil {
		t.Fatal(err)
	}
	if decodePage(t, getW).Flash["success"] != "created" {
		t.Fatalf("session flash was not rendered: %#v", decodePage(t, getW).Flash)
	}
	if _, ok := session.values[defaultSessionFlashKey]; ok {
		t.Fatalf("pulled flash data should be deleted: %#v", session.values)
	}
}

func TestSessionFlashStoreRejectsMissingResolverAndSession(t *testing.T) {
	_, err := NewSessionFlashStore(SessionFlashStoreConfig{})
	if !errors.Is(err, ErrMissingFlashSessionResolver) {
		t.Fatalf("expected ErrMissingFlashSessionResolver, got %v", err)
	}
	store, err := NewSessionFlashStore(SessionFlashStoreConfig{
		Session: FlashSessionFunc(func(req *http.Request) (FlashSession, error) { return nil, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Pull(httptest.NewRequest(http.MethodGet, "/", nil))
	if !errors.Is(err, ErrMissingFlashSession) {
		t.Fatalf("expected ErrMissingFlashSession, got %v", err)
	}
}

func TestSessionFlashStoreUsesCustomKey(t *testing.T) {
	session := &testFlashSession{values: map[string][]byte{}}
	store, err := NewSessionFlashStore(SessionFlashStoreConfig{
		Session: FlashSessionFunc(func(req *http.Request) (FlashSession, error) { return session, nil }),
		Key:     "app.flash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil), FlashData{
		Flash: Flash{"notice": "saved"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := session.values["app.flash"]; !ok {
		t.Fatalf("custom key was not used: %#v", session.values)
	}
}

func TestSessionFlashStoreKeepsCorruptValue(t *testing.T) {
	session := &testFlashSession{values: map[string][]byte{
		defaultSessionFlashKey: []byte("{"),
	}}
	store, err := NewSessionFlashStore(SessionFlashStoreConfig{
		Session: FlashSessionFunc(func(req *http.Request) (FlashSession, error) { return session, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Pull(httptest.NewRequest(http.MethodGet, "/", nil)); err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if _, ok := session.values[defaultSessionFlashKey]; !ok {
		t.Fatal("corrupt data should not be deleted")
	}
}

func TestSessionFlashStoreReflashDoesNotResolveOrMutateSession(t *testing.T) {
	calls := 0
	store, err := NewSessionFlashStore(SessionFlashStoreConfig{
		Session: FlashSessionFunc(func(req *http.Request) (FlashSession, error) {
			calls++
			return nil, errors.New("unexpected session resolution")
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Reflash(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil)); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("reflash should leave unpulled session data unchanged: %d", calls)
	}
}

type testFlashSession struct {
	values map[string][]byte
}

func (s *testFlashSession) Get(key string) ([]byte, bool, error) {
	value, ok := s.values[key]
	return append([]byte(nil), value...), ok, nil
}

func (s *testFlashSession) Set(key string, value []byte) error {
	s.values[key] = append([]byte(nil), value...)
	return nil
}

func (s *testFlashSession) Delete(key string) error {
	delete(s.values, key)
	return nil
}
