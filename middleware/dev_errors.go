package middleware

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/mariomunozv/forge"
)

// DevErrors returns a middleware that renders a detailed HTML error page for
// panics, handler errors, and explicit 5xx responses (ctx.Error(5xx, ...)).
// Only use in development — replace with Recovery() in prod.
func DevErrors() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(ctx *forge.Context) (err error) {
			rec := &responseRecorder{original: ctx.Response, code: http.StatusOK}
			ctx.Response = rec

			defer func() {
				if r := recover(); r != nil {
					ctx.Response = rec.original
					renderDevError(ctx, fmt.Sprintf("%v", r), debug.Stack())
					err = nil
				}
			}()

			if handlerErr := next(ctx); handlerErr != nil {
				ctx.Response = rec.original
				renderDevError(ctx, handlerErr.Error(), debug.Stack())
				return nil
			}

			if rec.code >= 500 {
				ctx.Response = rec.original
				renderDevError(ctx, rec.errorMessage(), debug.Stack())
				return nil
			}

			rec.flush()
			return nil
		}
	}
}

// responseRecorder buffers the response so DevErrors can intercept 5xx writes.
type responseRecorder struct {
	original http.ResponseWriter
	code     int
	body     strings.Builder
	headers  http.Header
	written  bool
}

func (r *responseRecorder) Header() http.Header {
	if r.headers == nil {
		r.headers = make(http.Header)
	}
	return r.headers
}

func (r *responseRecorder) WriteHeader(code int) {
	if !r.written {
		r.code = code
	}
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.written = true
	return r.body.Write(b)
}

func (r *responseRecorder) errorMessage() string {
	body := r.body.String()
	// try to extract "message" from JSON envelope: {"error":{"message":"..."}}
	if i := strings.Index(body, `"message":"`); i >= 0 {
		rest := body[i+len(`"message":"`):]
		if end := strings.Index(rest, `"`); end >= 0 {
			return rest[:end]
		}
	}
	if body != "" {
		return body
	}
	return http.StatusText(r.code)
}

func (r *responseRecorder) flush() {
	for k, vals := range r.headers {
		for _, v := range vals {
			r.original.Header().Add(k, v)
		}
	}
	r.original.WriteHeader(r.code)
	fmt.Fprint(r.original, r.body.String())
}

// --- data types ---

type devErrorData struct {
	Error       string
	Method      string
	Path        string
	Frames      []stackFrame
	URLParams   []kv
	QueryParams []kv
	Headers     []kv
}

type stackFrame struct {
	Function string
	File     string
	Line     int
	Source   []sourceLine
	IsApp    bool
}

type sourceLine struct {
	Number  int
	Content string
	IsError bool
}

type kv struct {
	Key string
	Val string
}

// --- renderer ---

func renderDevError(ctx *forge.Context, msg string, stack []byte) {
	frames := parseStack(stack)

	data := devErrorData{
		Error:       msg,
		Method:      ctx.Request.Method,
		Path:        ctx.Request.URL.Path,
		Frames:      frames,
		URLParams:   toKVs(ctx.Params),
		QueryParams: queryKVs(ctx),
		Headers:     headerKVs(ctx),
	}

	var buf strings.Builder
	if err := devErrorTmpl.Execute(&buf, data); err != nil {
		http.Error(ctx.Response, msg, http.StatusInternalServerError)
		return
	}

	ctx.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.Response.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(ctx.Response, buf.String())
}

// --- stack parsing ---

func parseStack(stack []byte) []stackFrame {
	lines := strings.Split(string(stack), "\n")
	var frames []stackFrame

	for i := 0; i+1 < len(lines); i++ {
		fn := strings.TrimSpace(lines[i])
		next := lines[i+1]

		if !strings.HasPrefix(next, "\t") {
			continue
		}

		fileInfo := strings.TrimPrefix(next, "\t")
		if idx := strings.LastIndex(fileInfo, " +"); idx >= 0 {
			fileInfo = fileInfo[:idx]
		}

		lastColon := strings.LastIndex(fileInfo, ":")
		if lastColon < 0 {
			i++
			continue
		}

		file := fileInfo[:lastColon]
		lineNum, err := strconv.Atoi(strings.TrimSpace(fileInfo[lastColon+1:]))
		if err != nil {
			i++
			continue
		}

		frames = append(frames, stackFrame{
			Function: fn,
			File:     file,
			Line:     lineNum,
			Source:   readSourceContext(file, lineNum, 5),
			IsApp:    isAppFile(file),
		})
		i++ // skip file line
	}

	return frames
}

func isAppFile(file string) bool {
	goroot := runtime.GOROOT()
	return !strings.HasPrefix(file, goroot) &&
		!strings.Contains(file, "/go/pkg/mod/") &&
		!strings.Contains(file, "runtime/debug")
}

func readSourceContext(file string, line, ctx int) []sourceLine {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}

	all := strings.Split(string(data), "\n")
	start := max(0, line-ctx-1)
	end := min(len(all), line+ctx)

	out := make([]sourceLine, 0, end-start)
	for i := start; i < end; i++ {
		out = append(out, sourceLine{
			Number:  i + 1,
			Content: all[i],
			IsError: i+1 == line,
		})
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- request helpers ---

func toKVs(m map[string]string) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{k, v})
	}
	return out
}

func queryKVs(ctx *forge.Context) []kv {
	var out []kv
	for k, vals := range ctx.Request.URL.Query() {
		out = append(out, kv{k, strings.Join(vals, ", ")})
	}
	return out
}

var sensitiveHeaders = map[string]bool{
	"Authorization": true,
	"Cookie":        true,
	"Set-Cookie":    true,
}

func headerKVs(ctx *forge.Context) []kv {
	var out []kv
	for k, vals := range ctx.Request.Header {
		val := strings.Join(vals, ", ")
		if sensitiveHeaders[k] {
			val = "***"
		}
		out = append(out, kv{k, val})
	}
	return out
}

// --- HTML template ---

var devErrorTmpl = template.Must(template.New("dev_error").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>{{.Error}}</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f0f1a;color:#e2e8f0;min-height:100vh}
.header{background:linear-gradient(135deg,#c53030,#9b2c2c);padding:24px 32px;border-bottom:1px solid #742a2a}
.header .route{font-size:12px;color:rgba(255,255,255,.65);font-family:monospace;margin-bottom:6px}
.header .msg{font-size:20px;font-weight:700;color:#fff;line-height:1.4}
.layout{display:grid;grid-template-columns:1fr 280px;min-height:calc(100vh - 90px)}
.frames{border-right:1px solid #1e1e35;overflow:auto}
.frame{border-bottom:1px solid #1a1a30;cursor:pointer}
.fh{padding:10px 16px;display:flex;align-items:flex-start;gap:10px}
.fh:hover{background:#16162a}
.frame.app>.fh{background:#14142a}
.frame.app>.fh:hover{background:#1c1c38}
.badge{font-size:9px;font-weight:700;padding:2px 6px;border-radius:3px;flex-shrink:0;margin-top:2px}
.badge.app{background:#3b1fa3;color:#c4b5fd}
.badge.std{background:#1a2035;color:#4a5568}
.fn{font-size:12px;color:#a78bfa;font-family:monospace;word-break:break-all}
.fi{font-size:11px;color:#4a5568;margin-top:3px;font-family:monospace}
.frame.app .fi{color:#64748b}
.source{display:none;background:#080810}
.frame.open .source{display:block}
.source table{width:100%;border-collapse:collapse;font-family:'SF Mono',Consolas,monospace;font-size:12px}
.source td{padding:1px 0;white-space:pre}
.ln{color:#2d3748;padding:0 14px;text-align:right;min-width:48px;user-select:none;border-right:1px solid #12122a}
.cd{padding:0 16px;color:#718096}
.frame.app .cd{color:#a0aec0}
tr.el .ln{color:#f6ad55;background:#1a0f00;border-color:#2d1a00}
tr.el .cd{color:#fef3c7;background:#1a0f00}
.sidebar{padding:20px;overflow:auto;background:#0a0a18}
.sidebar h3{font-size:10px;font-weight:700;color:#374151;text-transform:uppercase;letter-spacing:.1em;margin:16px 0 8px}
.sidebar h3:first-child{margin-top:0}
.kv{display:flex;gap:8px;margin-bottom:5px;font-size:11px;font-family:monospace}
.kk{color:#7c3aed;flex-shrink:0;min-width:100px}
.vv{color:#6b7280;word-break:break-all}
</style>
</head>
<body>
<div class="header">
  <div class="route">{{.Method}} {{.Path}}</div>
  <div class="msg">{{.Error}}</div>
</div>
<div class="layout">
  <div class="frames">
    {{range $i,$f := .Frames}}
    <div class="frame{{if $f.IsApp}} app{{end}}{{if eq $i 0}} open{{end}}" onclick="this.classList.toggle('open')">
      <div class="fh">
        <span class="badge{{if $f.IsApp}} app{{else}} std{{end}}">{{if $f.IsApp}}app{{else}}std{{end}}</span>
        <div>
          <div class="fn">{{$f.Function}}</div>
          <div class="fi">{{$f.File}}:{{$f.Line}}</div>
        </div>
      </div>
      {{if $f.Source}}
      <div class="source"><table>
        {{range $f.Source}}<tr{{if .IsError}} class="el"{{end}}>
          <td class="ln">{{.Number}}</td>
          <td class="cd">{{.Content}}</td>
        </tr>{{end}}
      </table></div>
      {{end}}
    </div>
    {{end}}
  </div>
  <div class="sidebar">
    <h3>Request</h3>
    <div class="kv"><span class="kk">method</span><span class="vv">{{.Method}}</span></div>
    <div class="kv"><span class="kk">path</span><span class="vv">{{.Path}}</span></div>
    {{range .URLParams}}<div class="kv"><span class="kk">:{{.Key}}</span><span class="vv">{{.Val}}</span></div>{{end}}
    {{if .QueryParams}}<h3>Query</h3>
    {{range .QueryParams}}<div class="kv"><span class="kk">{{.Key}}</span><span class="vv">{{.Val}}</span></div>{{end}}{{end}}
    {{if .Headers}}<h3>Headers</h3>
    {{range .Headers}}<div class="kv"><span class="kk">{{.Key}}</span><span class="vv">{{.Val}}</span></div>{{end}}{{end}}
  </div>
</div>
</body>
</html>`))
