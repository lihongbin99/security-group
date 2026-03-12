package auth

import (
	"crypto/subtle"
	"sync"
	"time"

	"security-group/internal/config"
)

type failRecord struct {
	count     int
	firstFail time.Time
}

type Auth struct {
	password      string
	maxFailures   int
	failWindow    time.Duration
	blockDuration time.Duration

	mu       sync.Mutex
	failures map[string]*failRecord
	blocked  map[string]time.Time

	userMu    sync.Mutex
	userLocks map[string]*sync.Mutex
}

func New(cfg *config.Config) *Auth {
	a := &Auth{
		password:      cfg.Password,
		maxFailures:   cfg.Security.MaxFailures,
		failWindow:    cfg.Security.FailWindow,
		blockDuration: cfg.Security.BlockDuration,
		failures:      make(map[string]*failRecord),
		blocked:       make(map[string]time.Time),
		userLocks:     make(map[string]*sync.Mutex),
	}
	go a.cleanupLoop()
	return a
}

func (a *Auth) IsBlocked(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	t, ok := a.blocked[ip]
	if !ok {
		return false
	}
	if time.Now().After(t) {
		delete(a.blocked, ip)
		return false
	}
	return true
}

func (a *Auth) Authenticate(ip, password string) bool {
	if constantTimeEqual(a.password, password) {
		a.mu.Lock()
		delete(a.failures, ip)
		a.mu.Unlock()
		return true
	}
	a.recordFailure(ip)
	return false
}

func (a *Auth) LockUser(username string) *sync.Mutex {
	a.userMu.Lock()
	defer a.userMu.Unlock()
	m, ok := a.userLocks[username]
	if !ok {
		m = &sync.Mutex{}
		a.userLocks[username] = m
	}
	return m
}

func (a *Auth) recordFailure(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	rec, ok := a.failures[ip]
	if !ok || now.Sub(rec.firstFail) > a.failWindow {
		a.failures[ip] = &failRecord{count: 1, firstFail: now}
		return
	}
	rec.count++
	if rec.count >= a.maxFailures {
		a.blocked[ip] = now.Add(a.blockDuration)
		delete(a.failures, ip)
	}
}

func (a *Auth) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		a.mu.Lock()
		now := time.Now()
		for ip, t := range a.blocked {
			if now.After(t) {
				delete(a.blocked, ip)
			}
		}
		for ip, rec := range a.failures {
			if now.Sub(rec.firstFail) > a.failWindow {
				delete(a.failures, ip)
			}
		}
		a.mu.Unlock()
	}
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
