package inertia

import "net/http"

// FlashStore stores one-time flash data and validation errors across redirects.
type FlashStore interface {
	// Pull reads and clears flash data for req.
	Pull(req *http.Request) (FlashData, error)
	// Put stores flash data for the next request.
	Put(w http.ResponseWriter, req *http.Request, data FlashData) error
	// Reflash preserves existing flash data for another request.
	Reflash(w http.ResponseWriter, req *http.Request) error
}

// FlashData is the data stored by a FlashStore.
type FlashData struct {
	// Flash contains one-time flash props.
	Flash Flash
	// Errors contains default validation errors.
	Errors ValidationErrors
	// Bags contains named validation error bags.
	Bags map[string]ValidationErrors
}

// NoopFlashStore is a FlashStore implementation that stores no data.
type NoopFlashStore struct{}

// Pull returns no flash data.
func (NoopFlashStore) Pull(req *http.Request) (FlashData, error) {
	return FlashData{}, nil
}

// Put ignores data.
func (NoopFlashStore) Put(w http.ResponseWriter, req *http.Request, data FlashData) error {
	return nil
}

// Reflash does nothing.
func (NoopFlashStore) Reflash(w http.ResponseWriter, req *http.Request) error {
	return nil
}
