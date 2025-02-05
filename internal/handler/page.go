package handler

import (
	"net/http"

	"app/internal/view/component"
	component_user "app/internal/view/component/user"
	"app/internal/view/page"
)

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) error {
	return Render(w, r, page.Home())
}

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) error {
	if h.session.IsAuthenticated(r.Context()) {
		return HxRedirect(w, r, "/dashboard")
	}
	return Render(w, r, page.Login(component.LoginFormValues{}, ""))
}

func (h *Handler) DashboardPage(w http.ResponseWriter, r *http.Request) error {
	return Render(w, r, page.Dashboard())
}

func (h *Handler) CreateUserPage(w http.ResponseWriter, r *http.Request) error {
	if h.session.IsAuthenticated(r.Context()) {
		return HxRedirect(w, r, "/dashboard")
	}
	return Render(w, r, page.CreateUser(
		component_user.CreateUserFormValues{},
		component_user.CreateUserFormErrors{},
	))
}
