package forgetest_test

import (
	"net/http"
	"testing"

	"github.com/mariomunozv/forge"
	"github.com/mariomunozv/forge/forgetest"
)

// --- sample controllers ---

type PostsController struct{}

func (c *PostsController) Index(ctx *forge.Context) error {
	return ctx.Success(forge.M{
		"posts": []forge.M{
			{"id": 1, "title": "Hello Forge"},
			{"id": 2, "title": "Go on Rails vibes"},
		},
	})
}

func (c *PostsController) Show(ctx *forge.Context) error {
	id := ctx.Param("id")
	if id == "999" {
		return ctx.Error(http.StatusNotFound, "post not found")
	}
	return ctx.Success(forge.M{"id": id, "title": "Hello Forge"})
}

func (c *PostsController) Create(ctx *forge.Context) error {
	var input struct {
		Title string `json:"title"`
	}
	if err := ctx.Bind(&input); err != nil {
		return ctx.Error(http.StatusBadRequest, "invalid body")
	}
	return ctx.Created(forge.M{"id": 3, "title": input.Title})
}

// --- tests ---

func newApp(t *testing.T) *forgetest.TestApp {
	app := forgetest.New(t)
	app.Register("posts", &PostsController{})
	app.Resources("posts")
	return app
}

func TestIndex(t *testing.T) {
	app := newApp(t)

	app.Request("GET", "/posts").
		AsJSON().
		Do().
		AssertOK().
		AssertJSONPath("data.posts.0.title", "Hello Forge").
		AssertJSONPath("data.posts.1.title", "Go on Rails vibes")
}

func TestShow(t *testing.T) {
	app := newApp(t)

	app.Request("GET", "/posts/1").
		AsJSON().
		Do().
		AssertOK().
		AssertJSONPath("data.id", "1").
		AssertJSONPath("data.title", "Hello Forge")
}

func TestShow_NotFound(t *testing.T) {
	app := newApp(t)

	app.Request("GET", "/posts/999").
		AsJSON().
		Do().
		AssertNotFound().
		AssertJSONPath("error.message", "post not found").
		AssertJSONPath("error.code", float64(404))
}

func TestCreate(t *testing.T) {
	app := newApp(t)

	app.Request("POST", "/posts").
		WithBody(forge.M{"title": "New Post"}).
		Do().
		AssertCreated().
		AssertJSONPath("data.title", "New Post").
		AssertJSONPathExists("data.id")
}

func TestQueryParam(t *testing.T) {
	app := forgetest.New(t)
	app.Register("posts", &PostsController{})
	app.GET("/posts", "posts#index")

	app.Request("GET", "/posts").
		WithParam("format", "json").
		Do().
		AssertOK().
		AssertJSONPathExists("data.posts")
}

func TestResponseHeader(t *testing.T) {
	app := newApp(t)

	app.Request("GET", "/posts").
		AsJSON().
		Do().
		AssertHeader("Content-Type", "application/json")
}

func TestChaining(t *testing.T) {
	// assertions chain, so you can read them like a sentence
	app := newApp(t)

	app.Request("GET", "/posts/1").
		AsJSON().
		Do().
		AssertOK().
		AssertHeader("Content-Type", "application/json").
		AssertJSONPathExists("data").
		AssertJSONPath("data.title", "Hello Forge")
}
