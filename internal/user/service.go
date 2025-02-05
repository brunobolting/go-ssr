package user

import (
	"app/internal/core"
	"time"
)

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo}
}

func (s *UserService) Find(id string) (*User, error) {
	return s.repo.Find(id)
}

func (s *UserService) FindByEmail(email string) (*User, error) {
	return s.repo.FindByEmail(email)
}

func (s *UserService) StoreUser(req *CreateUserRequest) (*User, map[string]string, error) {
	user, _ := s.repo.FindByEmail(req.Email)
	if user != nil {
		return nil, nil, ErrUserAlreadyExists
	}
	errs := req.Validate()
	if len(errs) > 0 {
		return nil, errs, ErrInvalidRequest
	}
	user, err := NewUser(
		req.Name,
		req.Email,
		req.Password,
		req.Avatar,
	)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.Store(user); err != nil {
		return nil, nil, err
	}
	return user, nil, nil
}

func (s *UserService) UpdateUser(user *User, req *CreateUserRequest) (map[string]string, error) {
	errs := req.Validate()
	if len(errs) > 0 {
		return errs, ErrInvalidRequest
	}
	var roles []Role
	for _, role := range user.Roles {
		roles = append(roles, role)
	}
	user.Name = req.Name
	user.Avatar = req.Avatar
	user.Roles = append(req.Roles, roles...)
	return nil, s.Update(user)
}

func (s *UserService) Update(user *User) error {
	user.UpdatedAt = time.Now()
	return s.repo.Update(user)
}

func (s *UserService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *UserService) ListUsers(req ListRequest) (*ListUserResponse, error) {
	return s.repo.ListUsers(req)
}

func (s *UserService) ChangeStatus(user *User, status UserStatus) error {
	user.Status = status
	return s.repo.Update(user)
}

func (s *UserService) ListRoles(req ListRequest) (*ListRoleResponse, error) {
	return s.repo.ListRoles(req)
}

func (s *UserService) FindRole(id string) (*Role, error) {
	return s.repo.FindRole(id)
}

func (s *UserService) FindRoles(ids []string) ([]Role, error) {
	return s.repo.FindRoles(ids)
}

func (s *UserService) Authenticate(email, password string) (*User, error) {
	time.Sleep(core.GetRandomSleep())
	user, err := s.repo.FindByEmail(email)
	if err != nil || user == nil {
		return nil, ErrInvalidEmailOrPassword
	}

	time.Sleep(core.GetRandomSleep())
	if !user.ComparePassword(password) {
		return nil, ErrInvalidEmailOrPassword
	}
	return user, nil
}
