package service

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

var (
	projectKeyRe = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)
	colorRe      = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// validator collects field errors so that all of them are reported at once.
type validator struct {
	fields []domain.FieldError
}

func (v *validator) add(field, message string) {
	v.fields = append(v.fields, domain.FieldError{Field: field, Message: message})
}

// text trims value and checks that it is non-empty (when required) and at most max characters.
func (v *validator) text(field, value string, required bool, max int, emptyMessage string) string {
	value = strings.TrimSpace(value)
	switch {
	case required && value == "":
		v.add(field, emptyMessage)
	case utf8.RuneCountInString(value) > max:
		v.add(field, "Слишком длинное значение")
	}
	return value
}

func (v *validator) err() error {
	if len(v.fields) == 0 {
		return nil
	}
	return &domain.Error{Kind: domain.KindValidation, Message: "Некорректные данные", Fields: v.fields}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email && len(email) <= 254 && strings.Contains(email[strings.LastIndexByte(email, '@'):], ".")
}

// likePattern turns a search string into an ILIKE pattern matching it as a
// substring, with the wildcards of the input escaped.
func likePattern(query string) *string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
	pattern := "%" + escaped + "%"
	return &pattern
}

const (
	defaultPageLimit = 100
	maxPageLimit     = 500
)

// Page is a requested window of a list.
type Page struct {
	Limit  int
	Offset int
}

func (p Page) normalized() (limit, offset int32) {
	l, o := p.Limit, p.Offset
	if l <= 0 {
		l = defaultPageLimit
	}
	if l > maxPageLimit {
		l = maxPageLimit
	}
	if o < 0 {
		o = 0
	}
	return int32(l), int32(o)
}
