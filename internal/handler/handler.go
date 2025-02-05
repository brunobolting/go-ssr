package handler

import (
	"app/internal/user"
	"app/internal/view/component"
	"app/pkg/session"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type HttpHandler func(w http.ResponseWriter, r *http.Request) error

type Middleware func(w http.ResponseWriter, r *http.Request) error

type Options struct {
	AllowedOrigins []string
}

type Handler struct {
	r *chi.Mux
	mu *sync.Mutex
	user *user.UserService
	session *session.Manager
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r.ServeHTTP(w, r)
}

func NewHttpHandler(r *chi.Mux, userService *user.UserService, session *session.Manager, opts Options) http.Handler {
	h := &Handler{
		mu: &sync.Mutex{},
		user: userService,
		session: session,
	}
	r.Use(middleware.Logger)
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(h.session.SetSessionMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   opts.AllowedOrigins,
    	AllowedMethods:   []string{"GET", "PUT", "POST", "DELETE", "HEAD", "OPTION"},
    	AllowedHeaders:   []string{"User-Agent", "Content-Type", "Accept", "Accept-Encoding", "Accept-Language", "Cache-Control", "Connection", "DNT", "Host", "Origin", "Pragma", "Referer"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)

	r.Get("/", MakeHandler(h.HomePage))
	r.Group(func (r chi.Router) {
		r.Get("/login", MakeHandler(h.LoginPage))
		r.Post("/login", MakeHandler(h.handleLoginRequest))
		r.Get("/signup", MakeHandler(h.CreateUserPage))
		r.Post("/user/create", MakeHandler(h.handleCreateUserRequest))
		r.Get("/logout", MakeHandler(h.handleLogoutRequest))
	})
	r.Group(func (r chi.Router) {
		r.Use(MakeMiddleware(h.session.RequireAuthenticationMiddleware))
		r.Get("/dashboard", MakeHandler(h.DashboardPage))
	})

	h.r = r
	return h
}

func MakeHandler(h HttpHandler) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			slog.Error("API", "err", err.Error(), "path", fmt.Sprintf("%s %s", r.Method, r.URL.Path))
			Render(w, r, component.Error(err.Error()))
		}
	}
}

func MakeMiddleware(h Middleware) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := h(w, r); err != nil {
				Render(w, r, component.Error(err.Error()))
				slog.Error("API", "err", err.Error(), "path", fmt.Sprintf("%s %s", r.Method, r.URL.Path))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Render(w http.ResponseWriter, r *http.Request, c templ.Component) error {
	return c.Render(r.Context(), w)
}

func HxRedirect(w http.ResponseWriter, r *http.Request, url string) error {
	if len(r.Header.Get("HX-Request")) > 0 {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
	return nil
}
