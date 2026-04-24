package forge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "_forge_session"

// SignIn writes a signed session cookie for userID.
func (c *Context) SignIn(userID int64) {
	token := signSessionToken(userID)
	http.SetCookie(c.Response, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// SignOut clears the session cookie.
func (c *Context) SignOut() {
	http.SetCookie(c.Response, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// CurrentUserID reads and verifies the session cookie, returning the user ID.
func (c *Context) CurrentUserID() (int64, bool) {
	cookie, err := c.Request.Cookie(sessionCookieName)
	if err != nil {
		return 0, false
	}
	return verifySessionToken(cookie.Value)
}

// signSessionToken creates "userID.timestamp.hmac".
func signSessionToken(userID int64) string {
	payload := fmt.Sprintf("%d.%d", userID, time.Now().Unix())
	return payload + "." + computeMAC(payload)
}

// verifySessionToken validates the token and returns the userID.
func verifySessionToken(token string) (int64, bool) {
	// format: userID.timestamp.mac
	lastDot := strings.LastIndex(token, ".")
	if lastDot < 0 {
		return 0, false
	}
	payload := token[:lastDot]
	mac := token[lastDot+1:]

	if !hmac.Equal([]byte(computeMAC(payload)), []byte(mac)) {
		return 0, false
	}

	firstDot := strings.Index(payload, ".")
	if firstDot < 0 {
		return 0, false
	}
	userID, err := strconv.ParseInt(payload[:firstDot], 10, 64)
	if err != nil {
		return 0, false
	}
	return userID, true
}

func computeMAC(payload string) string {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "dev-secret-please-set-SESSION_SECRET"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
