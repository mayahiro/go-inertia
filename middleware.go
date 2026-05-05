package inertia

import "net/http"

// Middleware returns an HTTP middleware that handles Inertia protocol concerns.
func (r *Renderer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		AppendVary(w.Header(), HeaderInertia)

		if IsInertiaRequest(req) && req.Method == http.MethodGet {
			current, err := r.versionProvider.Version(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if current != nil && current != "" && req.Header.Get(HeaderInertiaVersion) != stringifyVersion(current) {
				if r.flashStore != nil {
					if err := r.flashStore.Reflash(w, req); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				w.Header().Set(HeaderInertiaLocation, r.urlResolver.URL(req))
				w.WriteHeader(http.StatusConflict)
				return
			}
		}

		next.ServeHTTP(w, req)
	})
}
