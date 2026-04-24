package cli

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

var generateAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Generate authentication scaffold (users table, User model, sessions controller)",
	Args:  cobra.NoArgs,
	RunE:  runGenerateAuth,
}

func init() {
	generateCmd.AddCommand(generateAuthCmd)
}

func runGenerateAuth(_ *cobra.Command, _ []string) error {
	version := time.Now().UTC().Format("20060102150405")
	modPath := readModulePath()

	data := struct{ ModulePath string }{ModulePath: modPath}

	files := []struct {
		path string
		tmpl string
	}{
		{
			path: fmt.Sprintf("db/migrations/%s_create_users.sql", version),
			tmpl: authMigrationTmpl,
		},
		{
			path: "app/models/user.go",
			tmpl: userModelTmpl,
		},
		{
			path: "app/controllers/sessions_controller.go",
			tmpl: sessionsControllerTmpl,
		},
		{
			path: "app/views/sessions/new.templ",
			tmpl: sessionsNewViewTmpl,
		},
	}

	for _, f := range files {
		if err := ensureFile(f.path, f.tmpl, data); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Println("=> Fetching golang.org/x/crypto...")
	c := exec.Command("go", "get", "golang.org/x/crypto")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		fmt.Println("   warning: could not fetch golang.org/x/crypto — run 'go get golang.org/x/crypto' manually")
	}

	fmt.Println()
	fmt.Println("Auth scaffold generated. Next steps:")
	fmt.Println("  1. forge db migrate")
	fmt.Println("  2. Wire up routes in config/app.go:")
	fmt.Println("       app.Register(\"sessions\", &controllers.SessionsController{})")
	fmt.Println("       app.GET(\"/login\",  \"sessions#new\")")
	fmt.Println("       app.POST(\"/login\",  \"sessions#create\")")
	fmt.Println("       app.DELETE(\"/logout\", \"sessions#destroy\")")
	fmt.Println("  3. Add middleware.Auth() to your middleware stack")
	return nil
}

var authMigrationTmpl = `-- migrate:up
CREATE TABLE users (
    id              BIGSERIAL    PRIMARY KEY,
    email           TEXT         NOT NULL UNIQUE,
    password_digest TEXT         NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- migrate:down
DROP TABLE users;
`

var userModelTmpl = `package models

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int64  ` + "`db:\"id\"`" + `
	Email          string ` + "`db:\"email\"`" + `
	PasswordDigest string ` + "`db:\"password_digest\"`" + `
}

func (u *User) Validate() []string {
	var errs []string
	if u.Email == "" {
		errs = append(errs, "email is required")
	}
	return errs
}

func (u *User) SetPassword(plain string) error {
	if plain == "" {
		return errors.New("password cannot be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordDigest = string(hash)
	return nil
}

func (u *User) CheckPassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordDigest), []byte(plain)) == nil
}
`

var sessionsControllerTmpl = `package controllers

import (
	"database/sql"
	"net/http"

	"{{.ModulePath}}/app/models"
	sessionviews "{{.ModulePath}}/app/views/sessions"
	"github.com/mariomunozv/forge"
	forgedb "github.com/mariomunozv/forge/db"
)

type SessionsController struct {
	DB *sql.DB
}

func (c *SessionsController) New(ctx *forge.Context) error {
	return ctx.Component(sessionviews.New(sessionviews.NewData{}))
}

func (c *SessionsController) Create(ctx *forge.Context) error {
	var params struct {
		Email    string ` + "`json:\"email\"`" + `
		Password string ` + "`json:\"password\"`" + `
	}
	if err := ctx.Bind(&params); err != nil {
		return ctx.Error(http.StatusBadRequest, "invalid request")
	}

	user, err := forgedb.QueryOne[models.User](c.DB,
		"SELECT * FROM users WHERE email = $1", params.Email)
	if err != nil {
		return ctx.Error(http.StatusUnauthorized, "invalid email or password")
	}

	if !user.CheckPassword(params.Password) {
		return ctx.Error(http.StatusUnauthorized, "invalid email or password")
	}

	ctx.SignIn(user.ID)
	return ctx.Redirect(http.StatusSeeOther, "/")
}

func (c *SessionsController) Destroy(ctx *forge.Context) error {
	ctx.SignOut()
	return ctx.Redirect(http.StatusSeeOther, "/login")
}
`

var sessionsNewViewTmpl = `package sessions

import "{{.ModulePath}}/app/views/layouts"

type NewData struct {
	Error string
}

templ New(data NewData) {
	@layouts.Application("Sign in") {
		<div class="auth-form">
			<h1>Sign in</h1>
			if data.Error != "" {
				<p class="error">{ data.Error }</p>
			}
			<form method="post" action="/login">
				<label>
					Email
					<input type="email" name="email" required/>
				</label>
				<label>
					Password
					<input type="password" name="password" required/>
				</label>
				<button type="submit">Sign in</button>
			</form>
		</div>
	}
}
`
