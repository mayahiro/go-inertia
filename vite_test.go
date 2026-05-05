package inertia

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"testing/fstest"
)

func TestViteManifestTags(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.json": {
			Data: []byte(`{"resources/js/app.jsx":{"file":"assets/app-def.js","css":["assets/app-abc.css"]}}`),
		},
	}
	vite, err := NewVite(ViteConfig{
		ManifestFS:   fsys,
		ManifestPath: "manifest.json",
		PublicPath:   "/build",
		Entry:        "resources/js/app.jsx",
	})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := vite.Tags()
	if err != nil {
		t.Fatal(err)
	}
	got := string(tags)
	want := `<link rel="stylesheet" href="/build/assets/app-abc.css"><script type="module" src="/build/assets/app-def.js"></script>`
	if got != want {
		t.Fatalf("unexpected tags:\nwant: %s\n got: %s", want, got)
	}
}

func TestViteDevTags(t *testing.T) {
	vite, err := NewVite(ViteConfig{
		DevServerURL: "http://localhost:5173",
		Entry:        "resources/js/app.jsx",
		ReactRefresh: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := vite.Tags()
	if err != nil {
		t.Fatal(err)
	}
	got := string(tags)
	for _, expected := range []string{
		`http://localhost:5173/@vite/client`,
		`http://localhost:5173/resources/js/app.jsx`,
		`__vite_plugin_react_preamble_installed__`,
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %s in %s", expected, got)
		}
	}
}

func TestViteVersionProviderUsesManifestHash(t *testing.T) {
	manifest := []byte(`{"resources/js/app.jsx":{"file":"assets/app-def.js"}}`)
	fsys := fstest.MapFS{
		"manifest.json": {
			Data: manifest,
		},
	}
	vite, err := NewVite(ViteConfig{
		ManifestFS:   fsys,
		ManifestPath: "manifest.json",
		PublicPath:   "/build",
		Entry:        "resources/js/app.jsx",
	})
	if err != nil {
		t.Fatal(err)
	}

	version, err := vite.VersionProvider().Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	hash := sha256.Sum256(manifest)
	if version != hex.EncodeToString(hash[:]) {
		t.Fatalf("unexpected version: %v", version)
	}
}

func TestViteMissingEntryReturnsError(t *testing.T) {
	fsys := fstest.MapFS{
		"manifest.json": {
			Data: []byte(`{"resources/js/other.jsx":{"file":"assets/other.js"}}`),
		},
	}
	vite, err := NewVite(ViteConfig{
		ManifestFS:   fsys,
		ManifestPath: "manifest.json",
		PublicPath:   "/build",
		Entry:        "resources/js/app.jsx",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := vite.Tags(); err == nil {
		t.Fatal("expected missing entry error")
	}
}
