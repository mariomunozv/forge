package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"g"},
	Short:   "Generate controllers, models, views, and resources",
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

// --- naming helpers ---

// pascal converts "blog_posts" or "blogPosts" to "BlogPosts"
func pascal(s string) string {
	parts := splitWords(s)
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteRune(unicode.ToUpper(rune(p[0])))
		b.WriteString(p[1:])
	}
	return b.String()
}

// camel converts "blog_posts" to "blogPosts"
func camel(s string) string {
	p := pascal(s)
	if len(p) == 0 {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// snake converts "BlogPosts" or "blogPosts" to "blog_posts"
func snake(s string) string {
	parts := splitWords(s)
	return strings.Join(parts, "_")
}

func splitWords(s string) []string {
	// split on underscores, hyphens, spaces, and camelCase boundaries
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	var parts []string
	var current strings.Builder
	for i, r := range s {
		if r == '_' {
			if current.Len() > 0 {
				parts = append(parts, strings.ToLower(current.String()))
				current.Reset()
			}
			continue
		}
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(rune(s[i-1])) {
			if current.Len() > 0 {
				parts = append(parts, strings.ToLower(current.String()))
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, strings.ToLower(current.String()))
	}
	return parts
}

// singular returns a naive singular form (users→user, posts→post)
func singular(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

// --- field parsing ---

// Field represents a model field parsed from "name:type"
type Field struct {
	Name           string // pascal case: "UserName"
	JSONName       string // snake case: "user_name"
	DBName         string // snake case: "user_name"
	GoType         string // "string", "int", "bool", etc.
	ValidationType string // "email", "url", or "" for default required check
}

func parseField(raw string) (Field, error) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return Field{}, fmt.Errorf("invalid field format %q — use name:type (e.g. title:string)", raw)
	}
	name, typ := parts[0], parts[1]
	goType, err := mapType(typ)
	if err != nil {
		return Field{}, err
	}
	snakeName := snake(name)
	return Field{
		Name:           pascal(name),
		JSONName:       snakeName,
		DBName:         snakeName,
		GoType:         goType,
		ValidationType: validationType(typ),
	}, nil
}

func mapType(t string) (string, error) {
	switch strings.ToLower(t) {
	case "string", "str", "text":
		return "string", nil
	case "email":
		return "string", nil
	case "url", "uri":
		return "string", nil
	case "int", "integer":
		return "int", nil
	case "int64":
		return "int64", nil
	case "float", "float64", "decimal":
		return "float64", nil
	case "bool", "boolean":
		return "bool", nil
	case "time", "datetime", "timestamp":
		return "time.Time", nil
	default:
		return "", fmt.Errorf("unknown field type %q — supported: string, int, int64, float, bool, time, email, url", t)
	}
}

func validationType(t string) string {
	switch strings.ToLower(t) {
	case "email":
		return "email"
	case "url", "uri":
		return "url"
	default:
		return ""
	}
}

// readModulePath reads the module path from go.mod in the current directory.
func readModulePath() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// --- file writer ---

// ensureFile creates path only if it doesn't already exist.
func ensureFile(path string, tmpl string, data any) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return writeGeneratedFile(path, tmpl, data)
}

func writeGeneratedFile(path string, tmpl string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	t, err := template.New("").Funcs(template.FuncMap{
		"pascal": pascal,
		"camel":  camel,
		"snake":  snake,
	}).Parse(tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Printf("  create  %s\n", path)
	return t.Execute(f, data)
}
