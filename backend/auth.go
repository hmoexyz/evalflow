package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

const tokenTTL = 7 * 24 * time.Hour

type tokenInfo struct {
	userID int64
	exp    time.Time
}

type authStore struct {
	mu     sync.Mutex
	tokens map[string]tokenInfo
}

func newAuthStore() *authStore {
	return &authStore{tokens: make(map[string]tokenInfo)}
}

// issue creates a new session token bound to the given user.
func (a *authStore) issue(userID int64) string {
	b := make([]byte, 32)
	rand.Read(b)
	tok := hex.EncodeToString(b)
	a.mu.Lock()
	a.tokens[tok] = tokenInfo{userID: userID, exp: time.Now().Add(tokenTTL)}
	a.mu.Unlock()
	return tok
}

// lookup returns the user id for a valid token.
func (a *authStore) lookup(tok string) (int64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	info, ok := a.tokens[tok]
	if !ok {
		return 0, false
	}
	if time.Now().After(info.exp) {
		delete(a.tokens, tok)
		return 0, false
	}
	return info.userID, true
}

// passwordHash encodes a password as "salt:hex(sha256(salt:password))".
func passwordHash(password string) string {
	salt := randomHex(16)
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return salt + ":" + hex.EncodeToString(sum[:])
}

func passwordVerify(got, stored string) bool {
	salt, hash, ok := strings.Cut(stored, ":")
	if !ok {
		return false
	}
	sum := sha256.Sum256([]byte(salt + ":" + got))
	want, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(sum[:], want) == 1
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
