package appui

import (
	"testing"

	"chordly/internal/state"
	"chordly/internal/ytm"
	"chordly/router"
)

func TestSignInView_Creation(t *testing.T) {
	r := router.New(&router.Route{Segment: "login", Name: "login"})
	ctx := router.RouteContext{
		Router: r,
		Match:  router.Match{Path: "/login"},
	}

	w := SignInPage(ctx)
	if w == nil {
		t.Fatal("expected non-nil SignInPage widget")
	}
}

func TestLoginPage_Alias(t *testing.T) {
	r := router.New(&router.Route{Segment: "login", Name: "login"})
	ctx := router.RouteContext{
		Router: r,
		Match:  router.Match{Path: "/login"},
	}

	w := LoginPage(ctx)
	if w == nil {
		t.Fatal("expected non-nil LoginPage widget")
	}
}

func TestSignInView_ProcessTokenInput_Empty(t *testing.T) {
	sigAuthError.Set("")
	sigStagedBundle.Set(nil)

	processTokenInput("   ")

	if sigAuthError.Get() == "" {
		t.Error("expected error message for empty input")
	}
	if sigStagedBundle.Get() != nil {
		t.Error("expected nil staged bundle for empty input")
	}
}

func TestSignInView_ProcessTokenInput_OAuthRejected(t *testing.T) {
	sigAuthError.Set("")
	sigStagedBundle.Set(nil)

	processTokenInput("ya29.a0AfH6SMB_fake_oauth_bearer_token")

	errStr := sigAuthError.Get()
	if errStr == "" {
		t.Fatal("expected error for OAuth token")
	}
	if !containsString(errStr, "Google OAuth tokens") {
		t.Errorf("expected explicit OAuth rejection copy, got: %s", errStr)
	}
	if sigStagedBundle.Get() != nil {
		t.Error("expected nil staged bundle")
	}
}

func TestSignInView_ProcessTokenInput_ValidCookie(t *testing.T) {
	sigAuthError.Set("")
	sigStagedBundle.Set(nil)

	cookieInput := "SAPISID=1234567890abcdef; SID=some_valid_session_id; __Secure-3PSID=secure_id;"
	processTokenInput(cookieInput)

	if sigAuthError.Get() != "" {
		t.Fatalf("unexpected error for valid cookie: %s", sigAuthError.Get())
	}

	staged := sigStagedBundle.Get()
	if staged == nil {
		t.Fatal("expected non-nil staged bundle")
	}
	if staged.SAPISID != "1234567890abcdef" {
		t.Errorf("expected SAPISID 1234567890abcdef, got %s", staged.SAPISID)
	}
	if staged.MaskedSAPISID() != "••••cdef" {
		t.Errorf("expected masked SAPISID ••••cdef, got %s", staged.MaskedSAPISID())
	}
}

func TestSignInView_ApplyValidatedSession(t *testing.T) {
	r := router.New(&router.Route{Segment: "login", Name: "login"})
	bundle := &ytm.TokenBundle{
		Cookie:      "SAPISID=abc12345",
		SAPISID:     "abc12345",
		AccountName: "Tester Music",
		Cookies:     map[string]string{"SAPISID": "abc12345"},
	}

	state.RouteReturn.Set("/library")
	applyValidatedSession(r, bundle)

	if state.UserAccount.Get() != "Tester Music" {
		t.Errorf("expected user account 'Tester Music', got %s", state.UserAccount.Get())
	}
	if ActiveSession.Get() != bundle {
		t.Error("expected ActiveSession to match bundle")
	}
	if sigStagedBundle.Get() != nil {
		t.Error("expected sigStagedBundle to be cleared")
	}
	if sigInputText.Get() != "" {
		t.Error("expected sigInputText to be cleared")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && (s[:len(substr)] == substr || containsString(s[1:], substr))))
}
