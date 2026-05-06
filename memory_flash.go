package inertia

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
)

const defaultMemoryFlashCookieName = "go_inertia_flash"

type MemoryFlashStore struct {
	CookieName     string
	CookiePath     string
	CookieSecure   bool
	CookieSameSite http.SameSite

	mu   sync.Mutex
	data map[string]FlashData
}

func NewMemoryFlashStore() *MemoryFlashStore {
	return &MemoryFlashStore{data: map[string]FlashData{}}
}

func (s *MemoryFlashStore) Pull(req *http.Request) (FlashData, error) {
	id, ok := s.sessionID(req)
	if !ok {
		return FlashData{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.data[id]
	delete(s.data, id)
	return data, nil
}

func (s *MemoryFlashStore) Put(w http.ResponseWriter, req *http.Request, data FlashData) error {
	id, ok := s.sessionID(req)
	if !ok {
		var err error
		id, err = newMemoryFlashSessionID()
		if err != nil {
			return err
		}
		http.SetCookie(w, s.cookie(id))
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = map[string]FlashData{}
	}
	s.data[id] = data
	return nil
}

func (s *MemoryFlashStore) Reflash(w http.ResponseWriter, req *http.Request) error {
	return nil
}

func (s *MemoryFlashStore) sessionID(req *http.Request) (string, bool) {
	cookie, err := req.Cookie(s.cookieName())
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func (s *MemoryFlashStore) cookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     s.cookieName(),
		Value:    value,
		Path:     s.cookiePath(),
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: s.cookieSameSite(),
	}
}

func (s *MemoryFlashStore) cookieName() string {
	if s.CookieName == "" {
		return defaultMemoryFlashCookieName
	}
	return s.CookieName
}

func (s *MemoryFlashStore) cookiePath() string {
	if s.CookiePath == "" {
		return "/"
	}
	return s.CookiePath
}

func (s *MemoryFlashStore) cookieSameSite() http.SameSite {
	if s.CookieSameSite == 0 {
		return http.SameSiteLaxMode
	}
	return s.CookieSameSite
}

func newMemoryFlashSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("inertia: memory flash session id could not be generated")
	}
	return hex.EncodeToString(buf), nil
}
