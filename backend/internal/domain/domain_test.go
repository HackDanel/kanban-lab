package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKindCodes(t *testing.T) {
	cases := map[Kind]string{
		KindInternal:         "INTERNAL",
		KindValidation:       "VALIDATION_ERROR",
		KindUnauthorized:     "UNAUTHORIZED",
		KindForbidden:        "FORBIDDEN",
		KindNotFound:         "NOT_FOUND",
		KindConflict:         "CONFLICT",
		KindWipLimit:         "WIP_LIMIT_EXCEEDED",
		KindColumnNotEmpty:   "COLUMN_NOT_EMPTY",
		KindRateLimited:      "RATE_LIMITED",
		KindFileTooLarge:     "FILE_TOO_LARGE",
		KindUnsupportedMedia: "UNSUPPORTED_MEDIA_TYPE",
		KindPayloadTooLarge:  "PAYLOAD_TOO_LARGE",
		Kind(999):            "INTERNAL",
	}
	for kind, code := range cases {
		assert.Equal(t, code, kind.Code())
	}
}

func TestKindOf(t *testing.T) {
	assert.Equal(t, KindForbidden, KindOf(ErrForbidden))
	assert.Equal(t, KindNotFound, KindOf(fmt.Errorf("load card: %w", NotFound("Карточка"))))
	assert.Equal(t, KindInternal, KindOf(errors.New("connection reset")))
	assert.Equal(t, KindInternal, KindOf(nil))
}

func TestInvalid(t *testing.T) {
	err := Invalid("title", "Название обязательно")
	assert.Equal(t, KindValidation, err.Kind)
	assert.Equal(t, []FieldError{{Field: "title", Message: "Название обязательно"}}, err.Fields)
	assert.Equal(t, "VALIDATION_ERROR: Некорректные данные", err.Error())
}

func TestRoles(t *testing.T) {
	assert.True(t, RoleAdmin.Valid())
	assert.True(t, RoleManager.Valid())
	assert.True(t, RoleDeveloper.Valid())
	assert.False(t, Role("owner").Valid())
	assert.False(t, Role("").Valid())

	assert.True(t, Actor{Role: RoleAdmin}.IsAdmin())
	assert.False(t, Actor{Role: RoleManager}.IsAdmin())

	assert.True(t, Actor{Role: RoleAdmin}.CanManage())
	assert.True(t, Actor{Role: RoleManager}.CanManage())
	assert.False(t, Actor{Role: RoleDeveloper}.CanManage())
}
