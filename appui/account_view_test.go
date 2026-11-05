package appui

import (
	"testing"

	"chordly/internal/state"
	"chordly/internal/ytm"
	"chordly/router"
)

func TestAccountPage_Guest(t *testing.T) {
	state.UserAccount.Set("")
	ActiveSession.Set(nil)

	r := router.New(&router.Route{Segment: "account", Name: "account"})
	ctx := router.RouteContext{
		Router: r,
		Match:  router.Match{Path: "/account"},
	}

	w := AccountPage(ctx)
	if w == nil {
		t.Fatal("expected non-nil AccountPage widget in guest state")
	}
}

func TestAccountPage_SignedIn(t *testing.T) {
	state.UserAccount.Set("Alex Listener")
	ActiveSession.Set(&ytm.TokenBundle{
		Cookie:      "SAPISID=abcd1234",
		SAPISID:     "abcd1234",
		Source:      ytm.TokenSourceInnerTune,
		AccountName: "Alex Listener",
		Cookies: map[string]string{
			"SAPISID": "abcd1234",
			"SID":     "sid123",
		},
		MissingRecommended: []string{"LOGIN_INFO"},
	})

	r := router.New(&router.Route{Segment: "account", Name: "account"})
	ctx := router.RouteContext{
		Router: r,
		Match:  router.Match{Path: "/account"},
	}

	w := AccountPage(ctx)
	if w == nil {
		t.Fatal("expected non-nil AccountPage widget in signed-in state")
	}
}
