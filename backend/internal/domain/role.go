package domain

import "github.com/google/uuid"

// Role is a global user role.
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleManager   Role = "manager"
	RoleDeveloper Role = "developer"
)

// Valid reports whether r is one of the known roles.
func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleManager || r == RoleDeveloper
}

// Actor is the authenticated user on whose behalf an operation runs.
type Actor struct {
	ID   uuid.UUID
	Role Role
}

// IsAdmin reports whether the actor is an administrator.
func (a Actor) IsAdmin() bool { return a.Role == RoleAdmin }

// CanManage reports whether the actor's role allows managing projects,
// boards, columns and labels. Project membership is checked separately.
func (a Actor) CanManage() bool { return a.Role == RoleAdmin || a.Role == RoleManager }
