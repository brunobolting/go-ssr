package user

import (
	"app/internal/core"
	"time"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusDeleted  UserStatus = "deleted"
)

type Role struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type User struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	Avatar    string     `json:"avatar"`
	Email     string     `json:"email"`
	Password  string     `json:"-"`
	Roles     []Role     `json:"roles"`
	Status    UserStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ListRequest struct {
	Page     int
	PageSize int
	Search   string
}

type ListUserResponse struct {
	Users    []User `json:"users"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	LastPage int    `json:"last_page"`
}

type ListRoleResponse struct {
	Roles    []Role `json:"roles"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	LastPage int    `json:"last_page"`
}

type UserRepository interface {
	Find(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	Store(user *User) error
	Update(user *User) error
	Delete(id string) error
	ListUsers(req ListRequest) (*ListUserResponse, error)
	ListRoles(req ListRequest) (*ListRoleResponse, error)
	FindRole(id string) (*Role, error)
	FindRoles(ids []string) ([]Role, error)
	StoreRole(role *Role) error
	UpdateRole(role *Role) error
	DeleteRole(id string) error
	GetUserRoles(userId string) ([]Role, error)
}

type CreateUserRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Avatar		  string `json:"avatar"`
	Password      string `json:"password"`
	PasswordCheck string `json:"password_check"`
	Roles         []Role `json:"roles"`
}

func (r *CreateUserRequest) Validate() map[string]string {
	errs := make(map[string]string)
	if r.Name == "" {
		errs["name"] = "Name is required"
	}
	if r.Email == "" {
		errs["email"] = "Email is required"
	}
	if r.Password == "" {
		errs["password"] = "Password is required"
	}
	if r.PasswordCheck == "" {
		errs["password_check"] = "Password confirmation is required"
	}
	if r.Password != r.PasswordCheck {
		errs["password_check"] = "Password and password confirmation must be the same"
	}
	// if len(r.Roles) == 0 {
	// 	return ErrRolesIsRequired
	// }
	return errs
}

func NewUser(name, email, password, avatar string) (*User, error) {
	hash, err := core.HashPassword(password)
	if err != nil {
		return nil, err
	}
	return &User{
		Id:        core.NewID(),
		Name:      name,
		Email:     email,
		Avatar:    avatar,
		Password:  hash,
		Status:    UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (u *User) ComparePassword(password string) bool {
	return core.ComparePassword(u.Password, password)
}
