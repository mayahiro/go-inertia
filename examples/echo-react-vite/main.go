package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"sync"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
	inertiaecho "github.com/mayahiro/go-inertia/adapters/echo"
)

type user struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type memoryFlashStore struct {
	mu   sync.Mutex
	data map[string]inertia.FlashData
}

func main() {
	rootView, err := inertia.NewTemplateRootViewFromFile("views/app.html", "app.html")
	if err != nil {
		panic(err)
	}

	vite, err := inertia.NewVite(inertia.ViteConfig{
		ManifestPath: "public/build/.vite/manifest.json",
		PublicPath:   "/build",
		Entry:        "resources/js/app.tsx",
		DevServerURL: os.Getenv("VITE_DEV_SERVER"),
		ReactRefresh: true,
	})
	if err != nil {
		panic(err)
	}

	renderer, err := inertia.New(inertia.Config{
		RootView:        rootView,
		VersionProvider: vite.VersionProvider(),
		FlashStore:      &memoryFlashStore{data: map[string]inertia.FlashData{}},
		SharedProps: inertia.SharedPropsFunc(func(req *http.Request) (inertia.Props, error) {
			return inertia.Props{
				"app": map[string]any{"name": "Go Inertia Admin"},
			}, nil
		}),
	})
	if err != nil {
		panic(err)
	}

	app := inertiaecho.New(renderer)
	e := echo.New()
	e.Static("/build", "public/build")
	e.Use(app.Middleware)

	users := []user{
		{ID: 1, Name: "Ada Lovelace", Email: "ada@example.com"},
		{ID: 2, Name: "Grace Hopper", Email: "grace@example.com"},
	}

	render := func(c *echo.Context, component string, props inertia.Props) error {
		tags, err := vite.Tags()
		if err != nil {
			return err
		}
		return app.Render(c, component, props, inertia.WithViteTags(tags))
	}

	e.GET("/", func(c *echo.Context) error {
		return render(c, "Dashboard", inertia.Props{
			"stats": map[string]any{
				"users":   len(users),
				"version": "v0.1.0",
			},
		})
	})

	e.GET("/users", func(c *echo.Context) error {
		return render(c, "Users/Index", inertia.Props{
			"users": users,
		})
	})

	e.POST("/users", func(c *echo.Context) error {
		name := c.FormValue("name")
		email := c.FormValue("email")
		if name == "" || email == "" {
			return app.Back(c, inertia.WithValidationErrors(requiredErrors(name, email)))
		}
		users = append(users, user{ID: len(users) + 1, Name: name, Email: email})
		return app.Redirect(c, "/users", inertia.WithFlash(inertia.Flash{
			"success": "ユーザーを作成しました",
		}))
	})

	if err := e.Start(serverAddress()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Error("server error", "error", err)
	}
}

func serverAddress() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return ":" + port
}

func requiredErrors(name string, email string) inertia.ValidationErrors {
	errors := inertia.ValidationErrors{}
	if name == "" {
		errors["name"] = "入力してください"
	}
	if email == "" {
		errors["email"] = "入力してください"
	}
	return errors
}

func (s *memoryFlashStore) Pull(req *http.Request) (inertia.FlashData, error) {
	id, ok := sessionID(req)
	if !ok {
		return inertia.FlashData{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data := s.data[id]
	delete(s.data, id)
	return data, nil
}

func (s *memoryFlashStore) Put(w http.ResponseWriter, req *http.Request, data inertia.FlashData) error {
	id, ok := sessionID(req)
	if !ok {
		var err error
		id, err = newSessionID()
		if err != nil {
			return err
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "go_inertia_example",
			Value:    id,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = data
	return nil
}

func (s *memoryFlashStore) Reflash(w http.ResponseWriter, req *http.Request) error {
	return nil
}

func sessionID(req *http.Request) (string, bool) {
	cookie, err := req.Cookie("go_inertia_example")
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func newSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("session id を生成できませんでした")
	}
	return hex.EncodeToString(buf), nil
}
