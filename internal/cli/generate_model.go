package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var generateModelCmd = &cobra.Command{
	Use:     "model [name] [fields...]",
	Aliases: []string{"m"},
	Short:   "Generate a model",
	Example: "  forge g model User name:string email:string age:int active:bool",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runGenerateModel,
}

func init() {
	generateCmd.AddCommand(generateModelCmd)
}

type modelData struct {
	Name            string  // "User"
	SnakeName       string  // "user"
	Fields          []Field // parsed fields
	HasTime         bool    // true if any field uses time.Time
	HasValidation   bool    // true if any field has email/url validation
}

func runGenerateModel(cmd *cobra.Command, args []string) error {
	name := singular(args[0])
	rawFields := args[1:]

	fields := make([]Field, 0, len(rawFields))
	hasTime := false
	hasValidation := false

	for _, raw := range rawFields {
		f, err := parseField(raw)
		if err != nil {
			return err
		}
		if f.GoType == "time.Time" {
			hasTime = true
		}
		if f.ValidationType != "" {
			hasValidation = true
		}
		fields = append(fields, f)
	}

	data := modelData{
		Name:          pascal(name),
		SnakeName:     snake(name),
		Fields:        fields,
		HasTime:       hasTime,
		HasValidation: hasValidation,
	}

	// Ensure validate.go exists in app/models/ when validation helpers are needed.
	if hasValidation {
		if err := ensureFile("app/models/validate.go", modelsValidateTmpl, nil); err != nil {
			return err
		}
	}

	path := fmt.Sprintf("app/models/%s.go", data.SnakeName)
	if err := writeGeneratedFile(path, modelTmpl, data); err != nil {
		return err
	}

	fmt.Printf("\nDone! Your model is at %s\n", path)
	return nil
}

var modelTmpl = `package models
{{if .HasTime}}
import "time"
{{end}}
type {{.Name}} struct {
	ID int ` + "`" + `json:"id" db:"id"` + "`" + `
{{- range .Fields}}
	{{.Name}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}" db:"{{.DBName}}"` + "`" + `
{{- end}}
}

// Table returns the database table name for {{.Name}}.
func ({{.Name}}) Table() string {
	return "{{.SnakeName}}s"
}

var {{.Name}}Fields = []string{
	{{- range .Fields}}
	"{{.JSONName}}",
	{{- end}}
}

// Validate returns a list of validation errors, or nil if the model is valid.
func (m *{{.Name}}) Validate() []string {
	{{- if .Fields}}
	var errs []string
	{{- range .Fields}}
	{{- if eq .ValidationType "email"}}
	if !isEmail(m.{{.Name}}) {
		errs = append(errs, "{{.JSONName}} must be a valid email address")
	}
	{{- else if eq .ValidationType "url"}}
	if !isURL(m.{{.Name}}) {
		errs = append(errs, "{{.JSONName}} must be a valid URL")
	}
	{{- else if eq .GoType "string"}}
	if m.{{.Name}} == "" {
		errs = append(errs, "{{.JSONName}} is required")
	}
	{{- end}}
	{{- end}}
	return errs
	{{- else}}
	return nil
	{{- end}}
}
` + func() string {
	_ = strings.ToLower
	return ""
}()

var modelsValidateTmpl = `package models

import (
	"net/url"
	"regexp"
)

var emailRe = regexp.MustCompile(` + "`" + `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$` + "`" + `)

func isEmail(s string) bool {
	return s != "" && emailRe.MatchString(s)
}

func isURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}
`
