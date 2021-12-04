package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/groob/plist"
	"gopkg.in/yaml.v2"
)

type Defaults struct {
	Catalogs  []string `yaml:"catalogs"`
	Manifests []string `yaml:"manifests"`
}

type Device struct {
	Name             string   `yaml:"name"`
	ClientIdentifier string   `yaml:"client_identifier"`
	Catalogs         []string `yaml:"catalogs"`
	Manifests        []string `yaml:"manifests"`
}

// Assignments is a parsed assignment configuration structure
type Assignments struct {
	*Defaults `yaml:"default"`
	Devices   []*Device `yaml:"devices"`
}

func NewAssignments(path string) (*Assignments, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	config := new(Assignments)
	d := yaml.NewDecoder(f)
	if err = d.Decode(config); err != nil {
		return nil, fmt.Errorf("could not decode config: %w", err)
	}

	return config, nil
}

// Manifest returns a generated munki manifest for the given ClientIdentifier
func (a *Assignments) Manifest(id string) ([]byte, error) {
	type data struct {
		Catalogs          []string `plist:"catalogs"`
		IncludedManifests []string `plist:"included_manifests"`
	}

	var catalogs []string
	catalogSet := make(map[string]struct{})
	var manifests []string
	manifestSet := make(map[string]struct{})

	for _, d := range a.Devices {
		if d.ClientIdentifier == id {
			for _, c := range d.Catalogs {
				if _, ok := catalogSet[c]; !ok {
					catalogs = append(catalogs, c)
					catalogSet[c] = struct{}{}
				}
			}
			for _, m := range d.Manifests {
				if _, ok := manifestSet[m]; !ok {
					manifests = append(manifests, m)
					manifestSet[m] = struct{}{}
				}
			}
			break
		}
	}

	for _, c := range a.Defaults.Catalogs {
		if _, ok := catalogSet[c]; !ok {
			catalogs = append(catalogs, c)
			catalogSet[c] = struct{}{}
		}
	}
	for _, m := range a.Defaults.Manifests {
		if _, ok := manifestSet[m]; !ok {
			manifests = append(manifests, m)
			manifestSet[m] = struct{}{}
		}
	}

	d := &data{Catalogs: catalogs, IncludedManifests: manifests}
	return plist.MarshalIndent(d, "\t")
}

func ManifestHandler(rootPath, assignmentPath string) http.Handler {
	fs := NoIndexFileSystem{http.Dir(rootPath)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := r.Context().Value(ContextKeyLog).(*Log)

		// if manifest exists, serve it
		if _, err := fs.Open(r.URL.Path); err == nil {
			http.FileServer(fs).ServeHTTP(w, r)
			return
		} else if err != nil && !os.IsNotExist(err) {
			l.Error = &Error{fmt.Errorf("could not open file: %w", err)}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 404 on manifests directory
		if r.URL.Path == "" {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}

		l.ClientIdentifier = r.URL.Path

		// otherwise generate manifest from assignments
		assign, err := NewAssignments(assignmentPath)
		if err != nil {
			l.Error = &Error{fmt.Errorf("could not open assignments: %w", err)}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		buf, err := assign.Manifest(r.URL.Path)
		if err != nil {
			l.Error = &Error{fmt.Errorf("could not generate plist: %w", err)}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/xml")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(buf); err != nil {
			l.Error = &Error{fmt.Errorf("could not write response: %w", err)}
		}
	})
}
