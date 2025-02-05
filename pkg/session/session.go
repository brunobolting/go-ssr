package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	SESSION_NAME  = "session"
)

type Session struct {
	Id        string
	UserId    string
	Data      map[string]any
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Manager struct {
	sessions   map[string]*Session
	mu         sync.RWMutex
	lifetime   time.Duration
	lifetimeExtended time.Duration
	cookie     *CookieConfig
	repository SessionRepository
	gcInterval time.Duration
	secretKey  []byte
}

type CookieConfig struct {
	Name     string
	Path     string
	Domain   string
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

type SessionRepository interface {
	Get(id string) (*Session, error)
	Set(session *Session) error
	Delete(id string) error
	GC() error
	GetExpired() ([]Session, error)
}

type Options struct {
	Lifetime   time.Duration
	Cookie     *CookieConfig
	Repository SessionRepository
	GCInterval time.Duration
	SecretKey  []byte
}

func New(opts *Options) *Manager {
	if opts.Cookie == nil {
		opts.Cookie = &CookieConfig{
			Name:     "session_id",
			Path:     "/",
			MaxAge:   int(opts.Lifetime.Seconds()),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
	}
	m := &Manager{
		sessions:   make(map[string]*Session),
		lifetime:   opts.Lifetime,
		cookie:     opts.Cookie,
		repository: opts.Repository,
		gcInterval: opts.GCInterval,
		secretKey:  opts.SecretKey,
	}
	m.RunGC()
	return m
}

func (m *Manager) generateSessionId() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (m *Manager) signSessionId(sessionId string) string {
	h := hmac.New(sha256.New, m.secretKey)
	h.Write([]byte(sessionId))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", sessionId, signature)
}

func (m *Manager) verifySessionId(signedId string) (string, bool) {
	parts := strings.Split(signedId, ".")
	if len(parts) != 2 {
		return "", false
	}
	sessionId := parts[0]
	expectedSignature := m.signSessionId(sessionId)
	return sessionId, hmac.Equal([]byte(expectedSignature), []byte(signedId))
}

func (m *Manager) Create(ctx context.Context, userId string, extended bool) (*Session, error) {
	id, err := m.generateSessionId()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(m.lifetime)
	if extended {
		expiresAt = now.Add(m.lifetimeExtended)
	}
	session := &Session{
		Id:        id,
		UserId:    userId,
		Data:      make(map[string]any),
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	if m.repository != nil {
		if err := m.repository.Set(session); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (m *Manager) Get(id string) (*Session, error) {
	id, valid := m.verifySessionId(id)
	if !valid {
		return nil, ErrInvalidSession
	}
	m.mu.RLock()
	session, exists := m.sessions[id]
	m.mu.RUnlock()

	if !exists && m.repository != nil {
		var err error
		session, err = m.repository.Get(id)
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.sessions[id] = session
		m.mu.Unlock()
	}

	if session == nil {
		return nil, ErrSessionNotFound
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		m.Destroy(id)
		return nil, ErrSessionExpired
	}

	return session, nil
}

func (m *Manager) Destroy(id string) error {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()

	if m.repository != nil {
		return m.repository.Delete(id)
	}

	return nil
}

func (m *Manager) SetSessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(m.cookie.Name)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		session, err := m.Get(cookie.Value)
		if err != nil {
			m.clearCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), SESSION_NAME, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// func (m *Manager) RequireAuthenticationMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		_, err := m.GetSession(r.Context())
// 		if err != nil {
// 			http.Error(w, ErrUserUnauthorized.Error(), http.StatusUnauthorized)
// 			return
// 		}
// 		next.ServeHTTP(w, r)
// 	})
// }

func (m *Manager) RequireAuthenticationMiddleware(w http.ResponseWriter, r *http.Request) error {
	_, err := m.GetSession(r.Context())
	if err != nil {
		log.Println(err)
		return ErrUserUnauthorized
	}
	return nil
}

func (m *Manager) SetCookie(w http.ResponseWriter, sessionId string, extended bool) {
	signedId := m.signSessionId(sessionId)
	maxAge := m.cookie.MaxAge
	if extended {
		maxAge = int(m.lifetimeExtended.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookie.Name,
		Value:    signedId,
		Path:     m.cookie.Path,
		Domain:   m.cookie.Domain,
		MaxAge:   maxAge,
		Secure:   m.cookie.Secure,
		HttpOnly: m.cookie.HttpOnly,
		SameSite: m.cookie.SameSite,
	})
}

func (m *Manager) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookie.Name,
		Value:    "",
		Path:     m.cookie.Path,
		Domain:   m.cookie.Domain,
		MaxAge:   -1,
		Secure:   m.cookie.Secure,
		HttpOnly: m.cookie.HttpOnly,
		SameSite: m.cookie.SameSite,
	})
}

func (m *Manager) GC() {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessions, err := m.GetExpiredSessions()
	if err != nil {
		log.Println(err)
		return
	}
	for _, session := range sessions {
		delete(m.sessions, session.Id)
		if (m.repository != nil) {
			err = m.repository.Delete(session.Id)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (m *Manager) RunGC() {
	go func() {
		i := m.gcInterval
		if i == 0 {
			i = 1 * time.Hour
		}
		ticker := time.NewTicker(i)
		for range ticker.C {
			log.Println("running session GC")
			m.GC()
		}
	}()
}

func (m *Manager) GetSession(ctx context.Context) (*Session, error) {
	session, ok := ctx.Value(SESSION_NAME).(*Session)
	if !ok || session == nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (m *Manager) GetExpiredSessions() ([]Session, error) {
	if m.repository != nil {
		return m.repository.GetExpired()
	}
	var expired []Session
	now := time.Now().UTC()
	for _, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			expired = append(expired, *session)
		}
	}
	return expired, nil
}

func (m *Manager) IsAuthenticated(ctx context.Context) bool {
	session, err := m.GetSession(ctx)
	if err != nil {
		return false
	}
	return session.UserId != ""
}
