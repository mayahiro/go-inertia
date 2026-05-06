package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	echo "github.com/labstack/echo/v5"
)

func TestBindCreateUserAcceptsJSON(t *testing.T) {
	input, err := bindCreateUserContext(`{"name":"Ada Lovelace","email":"ada@example.com"}`, echo.MIMEApplicationJSON)
	if err != nil {
		t.Fatal(err)
	}

	if input.Name != "Ada Lovelace" || input.Email != "ada@example.com" {
		t.Fatalf("unexpected input: %#v", input)
	}
}

func TestBindCreateUserAcceptsForm(t *testing.T) {
	input, err := bindCreateUserContext("name=Grace+Hopper&email=grace%40example.com", echo.MIMEApplicationForm)
	if err != nil {
		t.Fatal(err)
	}

	if input.Name != "Grace Hopper" || input.Email != "grace@example.com" {
		t.Fatalf("unexpected input: %#v", input)
	}
}

func TestCreatedUserAppearsOnFirstPage(t *testing.T) {
	users := prependUser(seedUsers(), createUserInput{
		Name:  "New User",
		Email: "new@example.com",
	})

	page := paginateUsers(httptest.NewRequest(http.MethodGet, "/users", nil), users)

	if len(page.Data) == 0 {
		t.Fatal("expected first page users")
	}
	if page.Data[0].Name != "New User" || page.Data[0].Email != "new@example.com" {
		t.Fatalf("created user should be first: %#v", page.Data[0])
	}
}

func bindCreateUserContext(body string, contentType string) (createUserInput, error) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, contentType)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return bindCreateUser(c)
}
