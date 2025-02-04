package handler

import (
	"net/http"

	"app/internal/view/page"
)

func HomePage(w http.ResponseWriter, r *http.Request) error {
	return Render(w, r, page.Home())
}

func LoginPage(w http.ResponseWriter, r *http.Request) error {
	return Render(w, r, page.Login())
}

func DashboardPage(w http.ResponseWriter, r *http.Request) error {
	return Render(w, r, page.Dashboard())
}
