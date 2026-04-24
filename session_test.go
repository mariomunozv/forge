package forge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignAndVerifySessionToken(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")

	token := signSessionToken(42)
	id, ok := verifySessionToken(token)
	if !ok {
		t.Fatal("expected valid token")
	}
	if id != 42 {
		t.Fatalf("expected userID 42, got %d", id)
	}
}

func TestVerifySessionToken_Invalid(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")

	cases := []string{
		"",
		"notavalidtoken",
		"42.1234567890.BADSIG",
	}
	for _, c := range cases {
		_, ok := verifySessionToken(c)
		if ok {
			t.Errorf("expected invalid for %q", c)
		}
	}
}

func TestVerifySessionToken_WrongSecret(t *testing.T) {
	t.Setenv("SESSION_SECRET", "secret-A")
	token := signSessionToken(7)

	t.Setenv("SESSION_SECRET", "secret-B")
	_, ok := verifySessionToken(token)
	if ok {
		t.Error("token signed with different secret should not verify")
	}
}

func TestContext_SignInSignOut(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := NewContext(w, r, nil)

	ctx.SignIn(99)

	// Extract cookie from response and attach to new request
	resp := w.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie after SignIn")
	}

	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.AddCookie(cookies[0])
	w2 := httptest.NewRecorder()
	ctx2 := NewContext(w2, r2, nil)

	id, ok := ctx2.CurrentUserID()
	if !ok {
		t.Fatal("expected valid session")
	}
	if id != 99 {
		t.Fatalf("expected userID 99, got %d", id)
	}

	// SignOut clears the cookie
	ctx2.SignOut()
	resp2 := w2.Result()
	for _, c := range resp2.Cookies() {
		if c.Name == sessionCookieName && c.MaxAge == -1 {
			return
		}
	}
	t.Error("expected cookie cleared after SignOut")
}

func TestContext_CurrentUserID_NoCookie(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := NewContext(w, r, nil)

	_, ok := ctx.CurrentUserID()
	if ok {
		t.Error("expected no session without cookie")
	}
}
