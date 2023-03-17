package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/net/webdav"
)

// NoIndexFileSystem is a wrapper for http.FileSystem that prevents automatic indexes.
// Adapted from https://www.alexedwards.net/blog/disable-http-fileserver-directory-listings
type NoIndexFileSystem struct {
	http.FileSystem
}

// Open wraps http.FileSystem.Open to prevent automatic indexes
func (fs NoIndexFileSystem) Open(filepath string) (http.File, error) {
	f, err := fs.FileSystem.Open(filepath)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if s.IsDir() {
		index := path.Join(filepath, "index.html")
		if _, err := fs.FileSystem.Open(index); err != nil {
			if err := f.Close(); err != nil {
				return nil, err
			}
			return nil, err
		}
	}

	return f, nil
}

func logger(r *http.Request, err error) {
	if err != nil {
		l := r.Context().Value(ContextKeyLog).(*Log)
		l.Error = &Error{err}
	}
}

// RunServer starts the server
func RunServer() error {
	config := new(Config)
	err := envconfig.Process("", config)
	if err != nil {
		return fmt.Errorf("could not process configuration from environment: %w", err)
	}

	mux := http.NewServeMux()

	root, err := filepath.Abs(config.WebRoot)
	if err != nil {
		return fmt.Errorf("could not get absolute path to web root: %w", err)
	}

	// main repo file server
	mux.Handle("/", http.FileServer(NoIndexFileSystem{http.Dir(root)}))

	assignments, err := filepath.Abs(config.AssignmentsPath)
	if err != nil {
		return fmt.Errorf("could not get absolute path to assignments: %w", err)
	}

	// hide assignments if it's inside root
	if strings.HasPrefix(assignments, root+"/") {
		mux.Handle(strings.TrimPrefix(assignments, root), http.NotFoundHandler())
	}

	// dynamic manifests
	mux.Handle(config.ManifestRoot,
		http.StripPrefix(config.ManifestRoot,
			ManifestHandler(path.Join(root, config.ManifestRoot), assignments)))

	// webdav file server
	mux.Handle(config.WebDAVPrefix,
		BasicAuthHandler(config.Username, config.Password,
			&webdav.Handler{
				Prefix:     config.WebDAVPrefix,
				FileSystem: webdav.Dir(root),
				LockSystem: webdav.NewMemLS(),
				Logger:     logger,
			}))

	var handler = LogHandler(NewLogger(os.Stdout), mux)

	// rewrite for x-forwarded-for, etc headers
	if config.ProxyHeaders {
		handler = handlers.ProxyHeaders(handler)
	}

	log.Println("Listening on:", config.ListenAddr)

	return http.ListenAndServe(config.ListenAddr, handler)
}

func main() {
	if err := RunServer(); err != nil {
		log.Println("could not start server:", err)
	}
}
