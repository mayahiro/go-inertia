package inertia

import "net/http"

// RedirectOption configures a redirect response.
type RedirectOption func(*redirectOptions)

type redirectOptions struct {
	flash            Flash
	errors           ValidationErrors
	errorBag         string
	status           int
	preserveFragment bool
}

// WithFlash stores flash data for the next request.
func WithFlash(data Flash) RedirectOption {
	return func(opts *redirectOptions) {
		opts.flash = data
	}
}

// WithValidationErrors stores validation errors for the next request.
func WithValidationErrors(errors ValidationErrors) RedirectOption {
	return func(opts *redirectOptions) {
		opts.errors = errors
	}
}

// WithErrorBag stores validation errors in the named error bag.
func WithErrorBag(name string) RedirectOption {
	return func(opts *redirectOptions) {
		opts.errorBag = name
	}
}

// WithStatus overrides the HTTP redirect status.
func WithStatus(code int) RedirectOption {
	return func(opts *redirectOptions) {
		opts.status = code
	}
}

// WithPreserveFragment requests an Inertia redirect that preserves the URL fragment.
func WithPreserveFragment() RedirectOption {
	return func(opts *redirectOptions) {
		opts.preserveFragment = true
	}
}

// Redirect sends a redirect response and stores flash data when configured.
func (r *Renderer) Redirect(w http.ResponseWriter, req *http.Request, url string, opts ...RedirectOption) error {
	options := redirectOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if len(options.flash) > 0 || len(options.errors) > 0 {
		if r.flashStore == nil {
			return ErrMissingFlashStore
		}
		data := FlashData{
			Flash:  options.flash,
			Errors: options.errors,
		}
		if options.errorBag != "" && len(options.errors) > 0 {
			data.Errors = nil
			data.Bags = map[string]ValidationErrors{options.errorBag: options.errors}
		}
		if err := r.flashStore.Put(w, req, data); err != nil {
			return err
		}
	}

	AppendVary(w.Header(), HeaderInertia)
	if options.preserveFragment && IsInertiaRequest(req) {
		w.Header().Set(HeaderInertiaRedirect, url)
		w.WriteHeader(http.StatusConflict)
		return nil
	}

	status := options.status
	if status == 0 {
		status = http.StatusFound
		if IsInertiaRequest(req) && req.Method != http.MethodGet {
			status = http.StatusSeeOther
		}
	}

	http.Redirect(w, req, url, status)
	return nil
}

// Back redirects to the Referer header or to "/" when no Referer is present.
func (r *Renderer) Back(w http.ResponseWriter, req *http.Request, opts ...RedirectOption) error {
	url := req.Header.Get("Referer")
	if url == "" {
		url = "/"
	}
	return r.Redirect(w, req, url, opts...)
}

// Location sends an Inertia location response or a normal redirect for non-Inertia requests.
func (r *Renderer) Location(w http.ResponseWriter, req *http.Request, url string) error {
	AppendVary(w.Header(), HeaderInertia)
	if IsInertiaRequest(req) {
		w.Header().Set(HeaderInertiaLocation, url)
		w.WriteHeader(http.StatusConflict)
		return nil
	}
	http.Redirect(w, req, url, http.StatusFound)
	return nil
}
