// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/registry/challenge_test.go

package registry

import "testing"

func TestParseAuthParams(t *testing.T) {
	in := `realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"`
	got := parseAuthParams(in)

	want := map[string]string{
		"realm":   "https://auth.docker.io/token",
		"service": "registry.docker.io",
		"scope":   "repository:library/nginx:pull",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseAuthParams[%q] = %q; want %q", k, got[k], v)
		}
	}
}

func TestParseAuthParamsUnquotedAndEmpty(t *testing.T) {
	if got := parseAuthParams(""); len(got) != 0 {
		t.Errorf("parseAuthParams(\"\") = %v; want empty map", got)
	}
	got := parseAuthParams("error=invalid_token,realm=example")
	if got["error"] != "invalid_token" || got["realm"] != "example" {
		t.Errorf("unquoted params parsed wrong: %v", got)
	}
}

func TestParseBearerChallenge(t *testing.T) {
	wwwAuth := `Bearer realm="https://auth.example.com/token",service="registry.example.com",scope="repository:app:pull"`
	got, err := parseBearerChallenge(wwwAuth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Realm != "https://auth.example.com/token" {
		t.Errorf("Realm = %q", got.Realm)
	}
	if got.Service != "registry.example.com" {
		t.Errorf("Service = %q", got.Service)
	}
	if got.Scope != "repository:app:pull" {
		t.Errorf("Scope = %q", got.Scope)
	}
}

func TestParseBearerChallengeErrors(t *testing.T) {
	cases := []struct {
		name    string
		wwwAuth string
	}{
		{"empty", ""},
		{"not bearer", `Basic realm="x"`},
		{"missing realm", `Bearer service="registry.example.com"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseBearerChallenge(c.wwwAuth); err == nil {
				t.Errorf("parseBearerChallenge(%q) expected error, got nil", c.wwwAuth)
			}
		})
	}
}
