package httpserver

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

const (
	refreshCookie = "refresh_token"
	csrfCookie    = "csrf_token"
	// The refresh token is only ever needed by the /auth endpoints.
	refreshCookiePath = "/api/v1/auth"
)

type cookies struct {
	secure bool
}

func (c cookies) setSession(w http.ResponseWriter, s service.Session) {
	maxAge := int(s.RefreshTTL / time.Second)
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookie, Value: s.RefreshToken, Path: refreshCookiePath, MaxAge: maxAge,
		HttpOnly: true, Secure: c.secure, SameSite: http.SameSiteStrictMode,
	})
	// Readable by the page script, which echoes it in the X-CSRF-Token header.
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookie, Value: s.CSRFToken, Path: "/", MaxAge: maxAge,
		Secure: c.secure, SameSite: http.SameSiteStrictMode,
	})
}

func (c cookies) clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: refreshCookie, Path: refreshCookiePath, MaxAge: -1, HttpOnly: true, Secure: c.secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(w, &http.Cookie{Name: csrfCookie, Path: "/", MaxAge: -1, Secure: c.secure, SameSite: http.SameSiteStrictMode})
}

func cookieValue(r *http.Request, name string) string {
	if c, err := r.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}

var errCSRF = domain.E(domain.KindForbidden, "Неверный CSRF-токен")

// checkCSRF implements the double-submit check: a cross-site page can make the
// browser send the cookie but cannot read it to put the same value in the header.
func checkCSRF(r *http.Request, header string) error {
	cookie := cookieValue(r, csrfCookie)
	if cookie == "" || subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 {
		return errCSRF
	}
	return nil
}
