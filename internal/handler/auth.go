package handler

import (
	"app/internal/view/component"
	"net/http"
)

func (h *Handler) handleLoginRequest(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	remember := r.Form.Get("remember") == "on"

	u, err := h.user.Authenticate(email, password)
	if err != nil {
		return Render(w, r, component.LoginForm(component.LoginFormValues{
			Email: email,
			Password: password,
			Remember: remember,
		}, err.Error()))
	}

	s, err := h.session.Create(r.Context(), u.Id, remember)
	if err != nil {
		return err
	}
	h.session.SetCookie(w, s.Id, remember)

	return HxRedirect(w, r, "/dashboard")
}

func (h *Handler) handleLogoutRequest(w http.ResponseWriter, r *http.Request) error {
	session, err := h.session.GetSession(r.Context())
	if err != nil {
		return HxRedirect(w, r, "/")
	}
	h.session.Destroy(session.Id)
	return HxRedirect(w, r, "/")
}
