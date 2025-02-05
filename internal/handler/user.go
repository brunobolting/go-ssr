package handler

import (
	"app/internal/user"
	"app/internal/view/component"
	component_user "app/internal/view/component/user"
	"net/http"
)

func (h *Handler) handleCreateUserRequest(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	name := r.Form.Get("name")
	email := r.Form.Get("email")
	password := r.Form.Get("password")
	passwordCheck := r.Form.Get("password_check")

	req := &user.CreateUserRequest{
		Name: name,
		Email: email,
		Password: password,
		PasswordCheck: passwordCheck,
	}

	user, errors, err := h.user.StoreUser(req)
	if err != nil && errors == nil {
		return Render(w, r, component.Error(err.Error()))
	}
	if user == nil {
		return Render(w, r, component_user.CreateUserForm(
			component_user.CreateUserFormValues{
				Email: email,
				Name: name,
				Password: password,
				PasswordCheck: passwordCheck,
			},
			component_user.CreateUserFormErrors{
				Email: errors["email"],
				Name: errors["name"],
				Password: errors["password"],
				PasswordCheck: errors["password_check"],
			},
		))
	}
	return HxRedirect(w, r, "/login")
}
