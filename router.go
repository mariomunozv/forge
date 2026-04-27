package forge

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// route represents a single registered route.
type route struct {
	method  string
	parts   []string // path split by "/"
	handler string   // "controller#action"
}

// RouteInfo is a public description of a registered route.
type RouteInfo struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	Controller string `json:"controller"`
	Action     string `json:"action"`
}

// Routes returns all registered routes as RouteInfo slice.
func (r *Router) Routes() []RouteInfo {
	infos := make([]RouteInfo, 0, len(r.routes))
	for _, rt := range r.routes {
		parts := strings.SplitN(rt.handler, "#", 2)
		controller, action := parts[0], ""
		if len(parts) == 2 {
			action = parts[1]
		}
		infos = append(infos, RouteInfo{
			Method:     rt.method,
			Path:       "/" + strings.Join(rt.parts, "/"),
			Controller: controller,
			Action:     action,
		})
	}
	return infos
}

// Router holds all registered routes and the controller registry.
type Router struct {
	routes   []*route
	registry *registry
}

func newRouter() *Router {
	return &Router{registry: newRegistry()}
}

func (r *Router) add(method, path, handler string) {
	r.routes = append(r.routes, &route{
		method:  method,
		parts:   splitPath(path),
		handler: handler,
	})
}

func (r *Router) serve(w http.ResponseWriter, req *http.Request, middleware []MiddlewareFunc) {
	ctx := newContext(w, req, nil)

	// Route matching is the innermost handler so middleware runs first,
	// allowing any middleware (e.g. MethodOverride) to mutate ctx.Request
	// before the method and path are compared against registered routes.
	routeMatcher := func(ctx *Context) error {
		params := make(map[string]string)
		reqParts := splitPath(ctx.Request.URL.Path)

		for _, route := range r.routes {
			if route.method != ctx.Request.Method {
				continue
			}
			if match(route.parts, reqParts, params) {
				for k, v := range params {
					ctx.Params[k] = v
				}
				handler, err := r.resolve(route.handler)
				if err != nil {
					return err
				}
				return handler(ctx)
			}
			for k := range params {
				delete(params, k)
			}
		}

		http.NotFound(ctx.Response, ctx.Request)
		return nil
	}

	final := applyMiddleware(routeMatcher, middleware)
	if err := final(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// resolve turns "home#index" into a HandlerFunc by looking up the controller
// in the registry and calling the action method via reflection.
func (r *Router) resolve(handler string) (HandlerFunc, error) {
	parts := strings.SplitN(handler, "#", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("forge: invalid handler format '%s', expected 'controller#action'", handler)
	}

	controllerName := parts[0]
	actionName := strings.Title(parts[1]) //nolint:staticcheck

	controller, err := r.registry.get(controllerName)
	if err != nil {
		return nil, err
	}

	method := reflect.ValueOf(controller).MethodByName(actionName)
	if !method.IsValid() {
		return nil, fmt.Errorf("forge: action '%s' not found on controller '%s'", actionName, controllerName)
	}

	return func(ctx *Context) error {
		results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
		if len(results) == 1 && !results[0].IsNil() {
			return results[0].Interface().(error)
		}
		return nil
	}, nil
}

// match checks if route parts match request parts, filling params.
func match(routeParts, reqParts []string, params map[string]string) bool {
	if len(routeParts) != len(reqParts) {
		return false
	}
	for i, part := range routeParts {
		if strings.HasPrefix(part, ":") {
			params[part[1:]] = reqParts[i]
		} else if part != reqParts[i] {
			return false
		}
	}
	return true
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return []string{}
	}
	return parts
}

func applyMiddleware(h HandlerFunc, middleware []MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
