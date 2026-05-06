package inertia

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"
)

// ViteConfig configures Vite asset tag generation.
type ViteConfig struct {
	// ManifestPath is the path to Vite's manifest file.
	ManifestPath string
	// ManifestFS is the optional filesystem used to read ManifestPath.
	ManifestFS fs.FS
	// PublicPath is the public URL prefix for built assets.
	PublicPath string
	// Entry is the Vite entrypoint key in the manifest.
	Entry string
	// DevServerURL enables development mode when set.
	DevServerURL string
	// ReactRefresh enables the React refresh preamble in development mode.
	ReactRefresh bool
}

// Vite generates script and stylesheet tags for Vite assets.
type Vite struct {
	config ViteConfig
}

type viteManifestEntry struct {
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	Imports []string `json:"imports"`
}

// NewVite creates a Vite helper from config.
func NewVite(config ViteConfig) (*Vite, error) {
	if config.DevServerURL == "" && config.ManifestPath == "" {
		return nil, errors.New("inertia: vite manifest path is required")
	}
	if config.Entry == "" {
		return nil, errors.New("inertia: vite entry is required")
	}
	if config.PublicPath == "" {
		config.PublicPath = "/"
	}
	return &Vite{config: config}, nil
}

// Tags returns HTML tags for the configured Vite entry.
func (v *Vite) Tags() (template.HTML, error) {
	if v.config.DevServerURL != "" {
		return v.devTags()
	}

	manifest, err := v.readManifest()
	if err != nil {
		return "", err
	}
	entry, ok := manifest[v.config.Entry]
	if !ok {
		return "", errors.New("inertia: vite entry not found in manifest")
	}
	assets, err := collectViteAssets(manifest, entry)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, css := range assets.CSS {
		b.WriteString(`<link rel="stylesheet" href="`)
		b.WriteString(html.EscapeString(v.assetURL(css)))
		b.WriteString(`">`)
	}
	for _, preload := range assets.ModulePreloads {
		b.WriteString(`<link rel="modulepreload" href="`)
		b.WriteString(html.EscapeString(v.assetURL(preload)))
		b.WriteString(`">`)
	}
	b.WriteString(`<script type="module" src="`)
	b.WriteString(html.EscapeString(v.assetURL(entry.File)))
	b.WriteString(`"></script>`)
	return template.HTML(b.String()), nil
}

// VersionProvider returns a VersionProvider based on the Vite configuration.
func (v *Vite) VersionProvider() VersionProvider {
	if v.config.DevServerURL != "" {
		return StaticVersion(v.config.DevServerURL)
	}
	if v.config.ManifestFS != nil {
		return VersionFromFSFileHash(v.config.ManifestFS, v.config.ManifestPath)
	}
	return VersionFromFileHash(v.config.ManifestPath)
}

func (v *Vite) readManifest() (map[string]viteManifestEntry, error) {
	var body []byte
	var err error
	if v.config.ManifestFS != nil {
		body, err = fs.ReadFile(v.config.ManifestFS, v.config.ManifestPath)
	} else {
		body, err = os.ReadFile(v.config.ManifestPath)
	}
	if err != nil {
		return nil, err
	}

	manifest := map[string]viteManifestEntry{}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

type viteAssets struct {
	CSS            []string
	ModulePreloads []string
}

func collectViteAssets(manifest map[string]viteManifestEntry, entry viteManifestEntry) (viteAssets, error) {
	assets := viteAssets{}
	seenCSS := map[string]bool{}
	seenPreload := map[string]bool{}
	seenImport := map[string]bool{}

	addCSS := func(files []string) {
		for _, file := range files {
			if file != "" && !seenCSS[file] {
				seenCSS[file] = true
				assets.CSS = append(assets.CSS, file)
			}
		}
	}
	addPreload := func(file string) {
		if file != "" && !seenPreload[file] {
			seenPreload[file] = true
			assets.ModulePreloads = append(assets.ModulePreloads, file)
		}
	}

	var walk func(key string) error
	walk = func(key string) error {
		if seenImport[key] {
			return nil
		}
		seenImport[key] = true

		imported, ok := manifest[key]
		if !ok {
			return errors.New("inertia: vite import not found in manifest")
		}
		addCSS(imported.CSS)
		addPreload(imported.File)
		for _, child := range imported.Imports {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}

	addCSS(entry.CSS)
	for _, key := range entry.Imports {
		if err := walk(key); err != nil {
			return viteAssets{}, err
		}
	}
	return assets, nil
}

func (v *Vite) devTags() (template.HTML, error) {
	base, err := url.Parse(v.config.DevServerURL)
	if err != nil {
		return "", err
	}
	clientURL := v.devURL(base, "@vite/client")
	entryURL := v.devURL(base, v.config.Entry)

	var b strings.Builder
	if v.config.ReactRefresh {
		b.WriteString(`<script type="module">`)
		b.WriteString(`import RefreshRuntime from "`)
		b.WriteString(html.EscapeString(v.devURL(base, "@react-refresh")))
		b.WriteString(`";RefreshRuntime.injectIntoGlobalHook(window);window.$RefreshReg$=()=>{};window.$RefreshSig$=()=>type=>type;window.__vite_plugin_react_preamble_installed__=true;`)
		b.WriteString(`</script>`)
	}
	b.WriteString(`<script type="module" src="`)
	b.WriteString(html.EscapeString(clientURL))
	b.WriteString(`"></script>`)
	b.WriteString(`<script type="module" src="`)
	b.WriteString(html.EscapeString(entryURL))
	b.WriteString(`"></script>`)
	return template.HTML(b.String()), nil
}

func (v *Vite) assetURL(file string) string {
	publicPath := v.config.PublicPath
	if publicPath == "" {
		publicPath = "/"
	}
	return strings.TrimRight(publicPath, "/") + "/" + strings.TrimLeft(file, "/")
}

func (v *Vite) devURL(base *url.URL, file string) string {
	u := *base
	u.Path = path.Join(strings.TrimRight(base.Path, "/"), file)
	return u.String()
}

// Version returns the current Vite asset version.
func (v *Vite) Version(ctx context.Context) (any, error) {
	return v.VersionProvider().Version(ctx)
}
