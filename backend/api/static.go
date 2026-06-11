package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// StaticHandler serves the SvelteKit static build with SPA routing fallback.
//
// Behavior:
//   - exact file under dir/ exists  → serve it (with proper Content-Type)
//   - path doesn't match a file     → serve dir/index.html (so client-side
//                                     SvelteKit routing handles /admin etc.)
//   - any path starting with /api/  → return 404 (caller should have routed it
//                                     before falling through to this handler)
//
// In production we mount a single container that serves both API and the SPA.
// In local dev we run vite separately and skip this entirely.
func StaticHandler(dir string) http.HandlerFunc {
	indexPath := filepath.Join(dir, "index.html")
	fs := http.FileServer(http.Dir(dir))

	return func(w http.ResponseWriter, r *http.Request) {
		// /api/* must never be intercepted by the static handler.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		// Try the requested asset.
		clean := filepath.Clean(r.URL.Path)
		// Disallow directory traversal escaping `dir`.
		if strings.HasPrefix(clean, "..") {
			http.NotFound(w, r)
			return
		}
		full := filepath.Join(dir, clean)
		info, err := os.Stat(full)
		if err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		// SPA fallback — let SvelteKit's client router handle the path.
		http.ServeFile(w, r, indexPath)
	}
}
