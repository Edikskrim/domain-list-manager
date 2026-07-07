package auth

import (
	"context"
	"net/http"
	"time"
)

const sessionCookieName = "session_token"

// Middleware returns a Chi middleware that checks for valid sessions.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			redirectToLogin(w, r)
			return
		}

		session, err := s.GetSession(cookie.Value)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    "",
				Path:     "/",
				Expires:  time.Now().Add(-1 * time.Hour),
				HttpOnly: true,
			})
			redirectToLogin(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, "username", session.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// redirectToLogin redirects the request to the login page.
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	query := ""
	if r.URL.RawQuery != "" {
		query = "&" + r.URL.RawQuery
	} else {
		query = "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, "/login"+query, http.StatusFound)
}

// GetCurrentUsername returns the username from the request context.
func GetCurrentUsername(r *http.Request) string {
	if user, ok := r.Context().Value("username").(string); ok {
		return user
	}
	return ""
}
