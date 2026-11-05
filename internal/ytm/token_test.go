package ytm

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestParseTokenEmpty(t *testing.T) {
	if _, err := ParseToken(""); err != ErrEmptyToken {
		t.Fatalf("ParseToken(\"\") = %v; want %v", err, ErrEmptyToken)
	}
}

func TestParseTokenOAuthRejected(t *testing.T) {
	_, err := ParseToken("ya29.a0AfH6SMBx")
	if err != ErrOAuthTokenUnsupported {
		t.Fatalf("ParseToken(oauth) = %v; want %v", err, ErrOAuthTokenUnsupported)
	}
}

func TestParseTokenUnrecognized(t *testing.T) {
	if _, err := ParseToken("totally-not-a-session"); err != ErrUnrecognizedFormat {
		t.Fatalf("ParseToken(garbage) = %v; want %v", err, ErrUnrecognizedFormat)
	}
}

func TestParseTokenInputTooLarge(t *testing.T) {
	big := strings.Repeat("a", maxTokenBytes+1)
	if _, err := ParseToken(big); err != ErrInputTooLarge {
		t.Fatalf("ParseToken(big) = %v; want %v", err, ErrInputTooLarge)
	}
}

func TestParseTokenJSONBundle(t *testing.T) {
	raw := `{"source":"innertune","cookie":"SID=abc; HSID=def; SAPISID=sid123","visitorData":"CgtXYZ","authorization":null,"accountName":"Rexa"}`
	bundle, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("ParseToken(json) = %v; want nil", err)
	}
	if bundle.Source != TokenSourceJSON {
		t.Fatalf("Source = %q; want %q", bundle.Source, TokenSourceJSON)
	}
	if !strings.Contains(bundle.Cookie, "SAPISID=sid123") {
		t.Fatalf("Cookie = %q; want SAPISID", bundle.Cookie)
	}
	if bundle.VisitorData != "CgtXYZ" {
		t.Fatalf("VisitorData = %q; want CgtXYZ", bundle.VisitorData)
	}
	if bundle.AccountName != "Rexa" {
		t.Fatalf("AccountName = %q; want Rexa", bundle.AccountName)
	}
}

func TestParseTokenJSONAliases(t *testing.T) {
	raw := `{"cookies":"SID=a; HSID=b; SAPISID=c","visitor_data":"CgtQ"}`
	bundle, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("ParseToken(json aliases) = %v; want nil", err)
	}
	if bundle.Cookie == "" || !strings.Contains(bundle.Cookie, "HSID=b") {
		t.Fatalf("Cookie = %q; want parsed aliases", bundle.Cookie)
	}
	if bundle.VisitorData != "CgtQ" {
		t.Fatalf("VisitorData = %q; want CgtQ", bundle.VisitorData)
	}
}

func TestParseTokenJSONArray(t *testing.T) {
	raw := `[{"name":"SID","value":"a","domain":"youtube.com"},{"name":"SAPISID","value":"c","domain":"youtube.com"}]`
	bundle, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("ParseToken(json array) = %v; want nil", err)
	}
	if !strings.Contains(bundle.Cookie, "SID=a") || !strings.Contains(bundle.Cookie, "SAPISID=c") {
		t.Fatalf("Cookie = %q; want pairs", bundle.Cookie)
	}
}

func TestParseTokenBase64Cookie(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("SID=abc; HSID=def; SAPISID=sid99"))
	bundle, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("ParseToken(base64) = %v; want nil", err)
	}
	if bundle.Source != TokenSourceCookieHeader {
		t.Fatalf("Source = %q; want %q", bundle.Source, TokenSourceCookieHeader)
	}
	if !strings.Contains(bundle.Cookie, "SAPISID=sid99") {
		t.Fatalf("Cookie = %q; want SAPISID", bundle.Cookie)
	}
}

func TestParseTokenCookieHeader(t *testing.T) {
	raw := "SID=abc; HSID=def; SSID=ghi; APISID=jkl; SAPISID=sid123"
	bundle, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("ParseToken(cookie header) = %v; want nil", err)
	}
	if bundle.Source != TokenSourceCookieHeader {
		t.Fatalf("Source = %q; want %q", bundle.Source, TokenSourceCookieHeader)
	}
}

func TestParseTokenCookieHeaderMissingRequired(t *testing.T) {
	_, err := ParseToken("SID=abc; HSID=def; SSID=ghi; APISID=jkl")
	if !errors.Is(err, ErrMissingRequired) {
		t.Fatalf("ParseToken(missing SAPISID) = %v; want %v", err, ErrMissingRequired)
	}
}

func TestParseTokenNetscape(t *testing.T) {
	raw := "# Netscape HTTP Cookie File\nyoutube.com\tTRUE\t/\tTRUE\t2000000000\tSAPISID\tsid999\n"
	bundle, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("ParseToken(netscape) = %v; want nil", err)
	}
	if bundle.Source != TokenSourceCookiesTxt {
		t.Fatalf("Source = %q; want %q", bundle.Source, TokenSourceCookiesTxt)
	}
	if !strings.Contains(bundle.Cookie, "SAPISID=sid999") {
		t.Fatalf("Cookie = %q; want SAPISID", bundle.Cookie)
	}
}

func TestParseTokenDerivesSAPISID(t *testing.T) {
	raw := "SID=abc; __Secure-3PSID=psid1; __Secure-3PAPISID=papid1"
	bundle, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("ParseToken(derive) = %v; want nil", err)
	}
	if bundle.SAPISID != "papid1" {
		t.Fatalf("SAPISID = %q; want derived papid1", bundle.SAPISID)
	}
}

func TestParseTokenInvalidJSON(t *testing.T) {
	if _, err := ParseToken("{not-json"); err != ErrInvalidJSON {
		t.Fatalf("ParseToken(bad json) = %v; want %v", err, ErrInvalidJSON)
	}
}

func TestTokenParsersRegistered(t *testing.T) {
	found := map[string]bool{}
	for _, p := range tokenParsers {
		found[p.Name()] = true
	}
	for _, want := range []string{"cssCookie", "netscape", "json", "base64"} {
		if !found[want] {
			t.Fatalf("registry missing parser %q; got %v", want, found)
		}
	}
}
