package inertia

import "net/http"

// Middleware returns an HTTP middleware that handles Inertia protocol concerns.
func (r *Renderer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r.devTools != nil && r.devTools.isEndpoint(req) {
			r.devTools.serveEndpoint(w, req)
			return
		}
		if r.devTools != nil && r.devTools.authorized(req) {
			wrappedWriter, wrappedRequest, state := r.devTools.begin(w, req)
			if state != nil {
				writer := wrappedWriter.(*devToolsResponseWriter)
				defer r.devTools.finish(state, wrappedRequest, writer)
				w = wrappedWriter
				req = wrappedRequest
			}
		}

		AppendVary(w.Header(), HeaderInertia)

		if IsInertiaRequest(req) && req.Method == http.MethodGet {
			current, err := r.versionProvider.Version(req.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			currentVersion := stringifyVersion(current)
			if currentVersion != "" && req.Header.Get(HeaderInertiaVersion) != currentVersion {
				if r.flashStore != nil {
					if err := r.flashStore.Reflash(w, req); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}
				w.Header().Set(HeaderInertiaLocation, r.urlResolver.URL(req))
				w.Header().Set(HeaderInertiaVersion, currentVersion)
				w.WriteHeader(http.StatusConflict)
				return
			}
		}

		next.ServeHTTP(w, req)
	})
}
