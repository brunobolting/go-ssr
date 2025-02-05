package user

import (
	"app/internal/core"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type UserRepositorySqlite struct {
	db *sql.DB
}

func NewUserRepositorySqlite(db *sql.DB) *UserRepositorySqlite {
	return &UserRepositorySqlite{db}
}

func (r *UserRepositorySqlite) scanUserRow(row core.Rowscan) (*User, error) {
	var u User
	var nullableAvatar sql.NullString
	err := row.Scan(
		&u.Id,
		&u.Email,
		&u.Name,
		&u.Password,
		&nullableAvatar,
		&u.Status,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	u.Avatar = nullableAvatar.String
	roles, err := r.GetUserRoles(u.Id)
	if err != nil {
		return nil, err
	}
	u.Roles = roles
	return &u, nil
}


func (r *UserRepositorySqlite) scanRoleRow(row core.Rowscan) (*Role, error) {
	var role Role
	var nullablePermissions sql.NullString
	err := row.Scan(
		&role.Id,
		&role.Name,
		&role.Description,
		&nullablePermissions,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	role.Permissions = []string{}
	if nullablePermissions.Valid {
		var permissions []string
		err := json.Unmarshal([]byte(nullablePermissions.String), &permissions); if err != nil {
			return nil, err
		}
		role.Permissions = permissions
	}
	return &role, nil
}


func (r *UserRepositorySqlite) Find(id string) (*User, error) {
	query := "SELECT id, email, name, password, avatar, status, created_at, updated_at FROM users WHERE id = ?"
	return r.scanUserRow(r.db.QueryRow(query, id))
}

func (r *UserRepositorySqlite) FindByEmail(email string) (*User, error) {
	query := "SELECT id, email, name, password, avatar, status, created_at, updated_at FROM users WHERE email = ?"
	return r.scanUserRow(r.db.QueryRow(query, email))
}

func (r *UserRepositorySqlite) Store(user *User) error {
	tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

	query := `INSERT INTO users (
		id, email, name, password, avatar, status, created_at, updated_at
	) VALUES (
		?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = tx.Exec(
		query,
		user.Id,
		user.Email,
		user.Name,
		user.Password,
		user.Avatar,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if len(user.Roles) > 0 {
		for _, role := range user.Roles {
			query = "INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON DUPLICATE KEY UPDATE role_id = VALUES(role_id)"
			_, err = tx.Exec(
				query,
				user.Id,
				role.Id,
			)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *UserRepositorySqlite) Update(user *User) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE users SET name = ?, avatar = ?, status = ?, updated_at = ? WHERE id = ?",
		user.Name,
		user.Avatar,
		user.Status,
		user.UpdatedAt,
		user.Id,
	)
	if err != nil {
		return err
	}

	err = r.deleteRolesFromUser(tx, user.Id)
	if err != nil {
		return err
	}

	for _, role := range user.Roles {
		_, err = tx.Exec(
			"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON DUPLICATE KEY UPDATE role_id = VALUES(role_id)",
			user.Id,
			role.Id,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *UserRepositorySqlite) Delete(id string) error {
	query := "UPDATE users SET status = ? WHERE id = ?"
	_, err := r.db.Exec(
		query,
		UserStatusDeleted,
		id,
	)
	return err
}

func (r *UserRepositorySqlite) ListUsers(req ListRequest) (*ListUserResponse, error) {
	where := ""
	args := []interface{}{}
	if req.Search != "" {
		where = "WHERE email LIKE ? OR name LIKE ? OR username LIKE ?"
		search := "%" + req.Search + "%"
		args = append(args, search, search, search)
	}

	countQuery := "SELECT COUNT(*) FROM users " + where
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	offset := (req.Page - 1) * req.PageSize
	lastPage := int(math.Ceil(float64(total) / float64(req.PageSize)))

	query := fmt.Sprintf(`
        SELECT id, email, name, password, avatar, status, created_at, updated_at
        FROM users %s
        ORDER BY id
        LIMIT ? OFFSET ?`, where)

	args = append(args, req.PageSize, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := r.scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return &ListUserResponse{
		Users:    users,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		LastPage: lastPage,
	}, nil
}


func (r *UserRepositorySqlite) deleteRolesFromUser(tx *sql.Tx, userId string) error {
	_, err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", userId)
	return err
}

func (r *UserRepositorySqlite) ListRoles(req ListRequest) (*ListRoleResponse, error) {
	where := ""
	args := []interface{}{}
	if req.Search != "" {
		where = "WHERE name LIKE ? OR description LIKE ?"
		search := "%" + req.Search + "%"
		args = append(args, search, search)
	}

	countQuery := "SELECT COUNT(*) FROM roles " + where
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	offset := (req.Page - 1) * req.PageSize
	lastPage := int(math.Ceil(float64(total) / float64(req.PageSize)))

	query := fmt.Sprintf(`
        SELECT id, name, description,
		permissions, created_at, updated_at
		FROM roles %s
        ORDER BY id
        LIMIT ? OFFSET ?`, where)

	args = append(args, req.PageSize, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		rl, err := r.scanRoleRow(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, *rl)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return &ListRoleResponse{
		Roles:    roles,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		LastPage: lastPage,
	}, nil
}

func (r *UserRepositorySqlite) FindRole(id string) (*Role, error) {
	query := "SELECT id, name, description, permissions, created_at, updated_at FROM roles WHERE id = ?"
	role, err := r.scanRoleRow(r.db.QueryRow(query, id))
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (r *UserRepositorySqlite) FindRoles(ids []string) ([]Role, error) {
	if len(ids) == 0 {
		return []Role{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT
			id,
			name,
			description,
			permissions,
			created_at,
			updated_at
		FROM roles
		WHERE id IN (%s) ORDER BY id`,
		strings.Join(placeholders, ","),
	)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		r, err := r.scanRoleRow(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, *r)
	}
	if err = rows.Err(); err != nil {
        return nil, err
    }
	return roles, nil
}


func (r *UserRepositorySqlite) StoreRole(role *Role) error {
	query := `INSERT INTO roles (
		id, name, description, permissions, created_at, updated_at
	) VALUES (
		?, ?, ?, ?, ?, ?
	)`
	p, err := json.Marshal(role.Permissions)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		query,
		role.Id,
		role.Name,
		role.Description,
		p,
		role.CreatedAt,
		role.UpdatedAt,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepositorySqlite) UpdateRole(role *Role) error {
	query := `UPDATE roles SET name = ?, description = ?, permissions = ?, updated_at = ? WHERE id = ?`
	p, err := json.Marshal(role.Permissions)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		query,
		role.Name,
		role.Description,
		p,
		role.UpdatedAt,
		role.Id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepositorySqlite) DeleteRole(id string) error {
	query := "DELETE FROM roles WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepositorySqlite) GetUserRoles(userId string) ([]Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.permissions, r.created_at, r.updated_at
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = ? ORDER BY id
	`
	rows, err := r.db.Query(query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []Role
	for rows.Next() {
		r, err := r.scanRoleRow(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, *r)
	}
	if err = rows.Err(); err != nil {
        return nil, err
    }
	return roles, nil
}
