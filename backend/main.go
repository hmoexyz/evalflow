package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func main() {
	addr := getenv("ADDR", ":8080")
	store, err := openStore(getenv("DB_PATH", "data.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.db.Close()

	srv := &Server{
		store: store,
		auth:  newAuthStore(),
	}

	router := srv.routes()

	if dist := getenv("FRONTEND_DIST", filepath.Join("..", "frontend", "dist")); dirExists(dist) {
		router.Handle("/", spaHandler(dist))
		log.Printf("serving frontend from %s", dist)
	}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(router)))
}

// spaHandler serves static files from dist and falls back to index.html for SPA routes.
func spaHandler(dist string) http.Handler {
	fs := http.FileServer(http.Dir(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "..") {
			http.NotFound(w, r)
			return
		}
		p := filepath.Join(dist, path.Clean("/"+r.URL.Path))
		if fi, err := os.Stat(p); err != nil || fi.IsDir() {
			http.ServeFile(w, r, filepath.Join(dist, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}
