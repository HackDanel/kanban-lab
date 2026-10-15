package service

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

func TestValidatorText(t *testing.T) {
	var v validator
	assert.Equal(t, "ok", v.text("a", "  ok ", true, 10, "required"))
	assert.Equal(t, "", v.text("b", "   ", true, 10, "required"))
	assert.Equal(t, "", v.text("c", "", false, 10, "required"))
	v.text("d", strings.Repeat("я", 11), false, 10, "")
	assert.Equal(t, strings.Repeat("я", 10), v.text("e", strings.Repeat("я", 10), true, 10, ""), "length is counted in characters")
	assert.Equal(t, []domain.FieldError{
		{Field: "b", Message: "required"},
		{Field: "d", Message: "Слишком длинное значение"},
	}, v.fields)
	assert.Equal(t, domain.KindValidation, domain.KindOf(v.err()))

	var empty validator
	assert.NoError(t, empty.err())
}

func TestValidEmail(t *testing.T) {
	for _, email := range []string{"a@b.co", "first.last@example.com", "x+tag@sub.example.org"} {
		assert.True(t, validEmail(email), email)
	}
	for _, email := range []string{"", "a", "a@b", "a@b.", "Name <a@b.co>", "a b@c.de", strings.Repeat("a", 250) + "@b.co"} {
		assert.False(t, validEmail(email), email)
	}
	assert.Equal(t, "a@b.co", normalizeEmail("  A@B.Co "))
}

func TestLikePattern(t *testing.T) {
	assert.Nil(t, likePattern("   "))
	assert.Equal(t, `%abc%`, *likePattern(" abc "))
	assert.Equal(t, `%100\%\_a\\b%`, *likePattern(`100%_a\b`))
}

func TestPageNormalized(t *testing.T) {
	cases := []struct {
		in            Page
		limit, offset int32
	}{
		{Page{}, 100, 0},
		{Page{Limit: 10, Offset: 5}, 10, 5},
		{Page{Limit: 10_000}, 500, 0},
		{Page{Limit: -1, Offset: -3}, 100, 0},
	}
	for _, tc := range cases {
		l, o := tc.in.normalized()
		assert.Equal(t, tc.limit, l)
		assert.Equal(t, tc.offset, o)
	}
}

func TestCardHelpers(t *testing.T) {
	var v validator
	validatePriority(&v, "urgent")
	validatePriority(&v, "low")
	validateDescription(&v, strings.Repeat("x", 10001))
	validateDescription(&v, "fine")
	assert.Len(t, v.fields, 2)

	a, b := uuid.New(), uuid.New()
	assert.Len(t, dedupe([]uuid.UUID{a, b, a, b}), 2)
	assert.Empty(t, dedupe(nil))

	d1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC)
	assert.True(t, sameDate(nil, nil))
	assert.False(t, sameDate(&d1, nil))
	assert.False(t, sameDate(nil, &d1))
	assert.True(t, sameDate(&d1, &d2), "only the calendar date matters")

	assert.Nil(t, uuidText(nullUUID(nil)))
	assert.Equal(t, a.String(), *uuidText(nullUUID(&a)))
	assert.Nil(t, dateText(nil))
	assert.Equal(t, "2026-01-02", *dateText(&d1))
	assert.Equal(t, a.String()+","+b.String(), *idsText([]uuid.UUID{a, b}))

	var c validator
	assert.Equal(t, "#aabbcc", validateColor(&c, "#AABBCC"))
	validateColor(&c, "blue")
	assert.Len(t, c.fields, 1)

	var w validator
	assert.Nil(t, validateWipLimit(&w, nil))
	n := 0
	assert.Nil(t, validateWipLimit(&w, &n))
	n = 5
	assert.Equal(t, int32(5), *validateWipLimit(&w, &n))
	assert.Len(t, w.fields, 1)
}

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"report.pdf":             "report.pdf",
		"../../etc/passwd":       "passwd",
		`C:\Users\me\file.txt`:   "file.txt",
		"a\x00b\nc\".txt":        "abc.txt",
		"":                       "file",
		"/":                      "file",
		".":                      "file",
		"  spaced  ":             "spaced",
		strings.Repeat("я", 200): strings.Repeat("я", 127),
	}
	for in, want := range cases {
		assert.Equal(t, want, sanitizeFilename(in), "%q", in)
	}
}

func TestNormalizeContentType(t *testing.T) {
	ct, ok := normalizeContentType("text/plain; charset=utf-8")
	assert.True(t, ok)
	assert.Equal(t, "text/plain", ct)
	_, ok = normalizeContentType("text/html")
	assert.False(t, ok)
	_, ok = normalizeContentType("")
	assert.False(t, ok)
}

func TestRequireActor(t *testing.T) {
	assert.Equal(t, domain.KindUnauthorized, domain.KindOf(requireActor(domain.Actor{})))
	assert.NoError(t, requireActor(domain.Actor{ID: uuid.New()}))
}
