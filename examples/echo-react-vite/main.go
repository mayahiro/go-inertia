package main

import (
	"errors"
	"net/http"
	"os"

	echo "github.com/labstack/echo/v5"
	inertia "github.com/mayahiro/go-inertia"
	inertiaecho "github.com/mayahiro/go-inertia/adapters/echo"
)

type user struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type appProps struct {
	Name string `json:"name"`
}

type dashboardStats struct {
	Users   int    `json:"users"`
	Version string `json:"version"`
}

type dashboardPageProps struct {
	Stats dashboardStats `json:"stats"`
}

func (p dashboardPageProps) Props() inertia.Props {
	return inertia.Props{
		"stats": p.Stats,
	}
}

type usersIndexPageProps struct {
	Users []user `json:"users"`
}

func (p usersIndexPageProps) Props() inertia.Props {
	return inertia.Props{
		"users": p.Users,
	}
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

	viteTags, err := vite.Tags()
	if err != nil {
		panic(err)
	}

	renderer, err := inertia.New(inertia.Config{
		RootView:        rootView,
		VersionProvider: vite.VersionProvider(),
		FlashStore:      inertia.NewMemoryFlashStore(),
		SharedProps: inertia.SharedPropsFunc(func(req *http.Request) (inertia.Props, error) {
			return inertia.Props{
				"app": appProps{Name: "Go Inertia Admin"},
			}, nil
		}),
		DefaultRenderOptions: []inertia.RenderOption{
			inertia.WithViteTags(viteTags),
		},
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

	e.GET("/", func(c *echo.Context) error {
		return app.Render(c, "Dashboard", dashboardPageProps{
			Stats: dashboardStats{
				Users:   len(users),
				Version: "local",
			},
		}.Props())
	})

	e.GET("/users", func(c *echo.Context) error {
		return app.Render(c, "Users/Index", usersIndexPageProps{Users: users}.Props())
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
