package inertia

import (
	"encoding/json"
	"net/http"
	"reflect"
)

const defaultSessionFlashKey = "go-inertia.flash"

// FlashSession is the minimal serialized session API used by SessionFlashStore.
// Implementations are expected to persist mutations after the request finishes.
type FlashSession interface {
	// Get returns a copy of the serialized value stored at key.
	Get(key string) ([]byte, bool, error)
	// Set stores a serialized value at key.
	Set(key string, value []byte) error
	// Delete removes key from the session.
	Delete(key string) error
}

// FlashSessionResolver returns the application session attached to req.
type FlashSessionResolver interface {
	// FlashSession returns the session used for Inertia flash data.
	FlashSession(req *http.Request) (FlashSession, error)
}

// FlashSessionFunc adapts a function to FlashSessionResolver.
type FlashSessionFunc func(req *http.Request) (FlashSession, error)

// FlashSession calls f(req).
func (f FlashSessionFunc) FlashSession(req *http.Request) (FlashSession, error) {
	if f == nil {
		return nil, ErrMissingFlashSessionResolver
	}
	return f(req)
}

// SessionFlashStoreConfig configures a SessionFlashStore.
type SessionFlashStoreConfig struct {
	// Session resolves the application's durable request session.
	Session FlashSessionResolver
	// Key is the session key used for serialized FlashData.
	// The default is "go-inertia.flash".
	Key string
}

// SessionFlashStore bridges FlashStore to an application-managed durable session.
type SessionFlashStore struct {
	session FlashSessionResolver
	key     string
}

// NewSessionFlashStore creates a durable-session FlashStore bridge.
func NewSessionFlashStore(config SessionFlashStoreConfig) (*SessionFlashStore, error) {
	if isNilInterface(config.Session) {
		return nil, ErrMissingFlashSessionResolver
	}
	key := config.Key
	if key == "" {
		key = defaultSessionFlashKey
	}
	return &SessionFlashStore{session: config.Session, key: key}, nil
}

// Pull reads, decodes, and deletes flash data from the request session.
func (s *SessionFlashStore) Pull(req *http.Request) (FlashData, error) {
	session, err := s.resolveSession(req)
	if err != nil {
		return FlashData{}, err
	}
	encoded, found, err := session.Get(s.key)
	if err != nil || !found {
		return FlashData{}, err
	}
	var data FlashData
	if err := json.Unmarshal(encoded, &data); err != nil {
		return FlashData{}, err
	}
	if err := session.Delete(s.key); err != nil {
		return FlashData{}, err
	}
	return data, nil
}

// Put encodes flash data into the request session.
func (s *SessionFlashStore) Put(w http.ResponseWriter, req *http.Request, data FlashData) error {
	session, err := s.resolveSession(req)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return session.Set(s.key, encoded)
}

// Reflash preserves data because SessionFlashStore only deletes it during Pull.
func (s *SessionFlashStore) Reflash(w http.ResponseWriter, req *http.Request) error {
	return nil
}

func (s *SessionFlashStore) resolveSession(req *http.Request) (FlashSession, error) {
	if s == nil || isNilInterface(s.session) {
		return nil, ErrMissingFlashSessionResolver
	}
	session, err := s.session.FlashSession(req)
	if err != nil {
		return nil, err
	}
	if isNilInterface(session) {
		return nil, ErrMissingFlashSession
	}
	return session, nil
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
