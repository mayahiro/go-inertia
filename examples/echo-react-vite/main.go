package main

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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
		"stats":      p.Stats,
		"serverTime": inertia.Defer(loadServerTime).Rescue().Once(),
	}
}

func loadServerTime(req *http.Request) (any, error) {
	if req.URL.Query().Get("failServerTime") == "1" {
		return nil, errors.New("demonstration deferred prop failure")
	}
	return time.Now().Format(time.RFC3339), nil
}

type usersIndexPageProps struct {
	Users usersPage `json:"users"`
}

func (p usersIndexPageProps) Props() inertia.Props {
	return inertia.Props{
		"users": inertia.Scroll(p.Users, p.Users.metadata()).MatchOn("data.id"),
	}
}

type usersPage struct {
	Data        []user `json:"data"`
	previous    any
	next        any
	currentPage int
}

type createUserInput struct {
	Name  string `json:"name" form:"name"`
	Email string `json:"email" form:"email"`
}

func (p usersPage) metadata() inertia.ScrollMetadata {
	return inertia.ScrollMetadata{
		PageName:     "page",
		PreviousPage: p.previous,
		NextPage:     p.next,
		CurrentPage:  p.currentPage,
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

	dashboardHead, err := pageHead("Dashboard", "Dashboard overview")
	if err != nil {
		panic(err)
	}
	usersHead, err := pageHead("Users", "Manage users")
	if err != nil {
		panic(err)
	}

	renderer, err := inertia.New(inertia.Config{
		RootView:        rootView,
		RootElementID:   "inertia-app",
		VersionProvider: vite.VersionProvider(),
		FlashStore:      inertia.NewMemoryFlashStore(),
		DevTools: inertia.DevToolsConfig{
			Enabled: devToolsEnabled(os.Getenv("INERTIA_DEVTOOLS_ENABLED")),
			ComponentPathResolver: func(component string) string {
				return "resources/js/Pages/" + component + ".tsx"
			},
		},
		SharedProps: inertia.SharedPropsFunc(func(req *http.Request) (inertia.Props, error) {
			return inertia.Props{
				"app": inertia.Always(appProps{Name: "Go Inertia Admin"}),
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
	e.HTTPErrorHandler = app.ErrorHandler(inertiaecho.ErrorHandlerConfig{
		Pages: map[int]string{
			http.StatusNotFound: "Errors/NotFound",
		},
		DefaultComponent: "Errors/Error",
		Props: func(c *echo.Context, err error, status int) (inertia.Props, error) {
			return inertia.Props{"status": status}, nil
		},
		RenderOptions: func(c *echo.Context, err error, status int) ([]inertia.RenderOption, error) {
			title := strconv.Itoa(status) + " " + http.StatusText(status)
			head, headErr := pageHead(title, "The request could not be completed")
			return []inertia.RenderOption{inertia.WithServerHead(head)}, headErr
		},
	})
	e.Static("/build", "public/build")
	e.Use(app.Middleware)

	users := seedUsers()
	var usersMu sync.RWMutex

	e.GET("/", func(c *echo.Context) error {
		usersMu.RLock()
		userCount := len(users)
		usersMu.RUnlock()
		return app.Render(c, "Dashboard", dashboardPageProps{
			Stats: dashboardStats{
				Users:   userCount,
				Version: "local",
			},
		}.Props(), inertia.WithServerHead(dashboardHead))
	})

	e.GET("/users", func(c *echo.Context) error {
		usersMu.RLock()
		page := paginateUsers(c.Request(), users)
		usersMu.RUnlock()
		return app.Render(c, "Users/Index", usersIndexPageProps{
			Users: page,
		}.Props(), inertia.WithServerHead(usersHead))
	})

	e.POST("/users", func(c *echo.Context) error {
		input, err := bindCreateUser(c)
		if err != nil {
			return err
		}
		if input.Name == "" || input.Email == "" {
			return app.Back(c, inertia.WithValidationErrors(requiredErrors(input.Name, input.Email)))
		}
		usersMu.Lock()
		users = prependUser(users, input)
		usersMu.Unlock()
		return app.Redirect(c, "/users", inertia.WithFlash(inertia.Flash{
			"success": "User created",
		}))
	})

	e.GET("/demo/error", func(c *echo.Context) error {
		return errors.New("demonstration error")
	})

	if err := e.Start(serverAddress()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Error("server error", "error", err)
	}
}

func pageHead(title string, description string) (inertia.ServerHead, error) {
	return inertia.NewServerHead(
		inertia.HeadTitle(title+" | Go Inertia Admin"),
		inertia.HeadMeta("description", description),
	)
}

func seedUsers() []user {
	names := []struct {
		name  string
		email string
	}{
		{"Ada Lovelace", "ada@example.com"},
		{"Grace Hopper", "grace@example.com"},
		{"Katherine Johnson", "katherine@example.com"},
		{"Margaret Hamilton", "margaret@example.com"},
		{"Dorothy Vaughan", "dorothy@example.com"},
		{"Mary Jackson", "mary@example.com"},
		{"Annie Easley", "annie@example.com"},
		{"Radia Perlman", "radia@example.com"},
		{"Barbara Liskov", "barbara@example.com"},
		{"Frances Allen", "frances@example.com"},
		{"Jean Bartik", "jean@example.com"},
		{"Evelyn Boyd Granville", "evelyn@example.com"},
		{"Joan Clarke", "joan@example.com"},
		{"Hedy Lamarr", "hedy@example.com"},
		{"Sister Mary Kenneth Keller", "mary.keller@example.com"},
		{"Karen Sparck Jones", "karen@example.com"},
		{"Adele Goldberg", "adele@example.com"},
		{"Carol Shaw", "carol@example.com"},
	}

	users := make([]user, 0, len(names))
	for i, item := range names {
		users = append(users, user{
			ID:    i + 1,
			Name:  item.name,
			Email: item.email,
		})
	}
	return users
}

func prependUser(users []user, input createUserInput) []user {
	created := user{
		ID:    len(users) + 1,
		Name:  input.Name,
		Email: input.Email,
	}
	return append([]user{created}, users...)
}

func paginateUsers(req *http.Request, users []user) usersPage {
	const pageSize = 6

	totalPages := (len(users) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	page := queryPage(req, "page")
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := min(start+pageSize, len(users))

	var previous any
	if page > 1 {
		previous = page - 1
	}

	var next any
	if page < totalPages {
		next = page + 1
	}

	return usersPage{
		Data:        append([]user(nil), users[start:end]...),
		previous:    previous,
		next:        next,
		currentPage: page,
	}
}

func queryPage(req *http.Request, name string) int {
	page, err := strconv.Atoi(req.URL.Query().Get(name))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func bindCreateUser(c *echo.Context) (createUserInput, error) {
	var input createUserInput
	if err := c.Bind(&input); err != nil {
		return createUserInput{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	return input, nil
}

func serverAddress() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return ":" + port
}

func devToolsEnabled(value string) bool {
	enabled, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && enabled
}

func requiredErrors(name string, email string) inertia.ValidationErrors {
	errors := inertia.ValidationErrors{}
	if name == "" {
		errors["name"] = "Name is required"
	}
	if email == "" {
		errors["email"] = "Email is required"
	}
	return errors
}
