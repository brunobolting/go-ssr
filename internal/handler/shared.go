package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type HttpHandler func(w http.ResponseWriter, r *http.Request) error

func NewHttpHandler(r *chi.Mux) http.Handler {
	r.Use(middleware.Logger)
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"SomeOrigin"},
    	AllowedMethods:   []string{"GET", "PUT", "POST", "DELETE", "HEAD", "OPTION"},
    	AllowedHeaders:   []string{"User-Agent", "Content-Type", "Accept", "Accept-Encoding", "Accept-Language", "Cache-Control", "Connection", "DNT", "Host", "Origin", "Pragma", "Referer"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)

	r.Get("/", Make(HomePage))
	r.Get("/login", Make(LoginPage))
	r.Get("/dashboard", Make(DashboardPage))

	return r
}

func Make(h HttpHandler) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			slog.Error("API", "err", err.Error(), "path", fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		}
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

