// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/run_build/run_flag_helpers_test.go

package run_build

import "testing"

func TestParseBytes(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"", 0, false},
		{"1024", 1024, false},
		{"1k", 1024, false},
		{"1K", 1024, false},
		{"2m", 2 * 1024 * 1024, false},
		{"3g", 3 * 1024 * 1024 * 1024, false},
		{"bogus", 0, true},
		{"12x", 0, true},
	}

	for _, c := range cases {
		got, err := parseBytes(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseBytes(%q) expected error, got nil", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseBytes(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseBytes(%q) = %d; want %d", c.in, got, c.want)
		}
	}
}

func TestParseUlimits(t *testing.T) {
	got, err := parseUlimits([]string{"nofile=1024", "nproc=100:200"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 ulimits, got %d", len(got))
	}
	if got[0] != (UlimitFlag{Name: "nofile", Soft: 1024, Hard: 1024}) {
		t.Errorf("nofile parsed wrong: %+v", got[0])
	}
	if got[1] != (UlimitFlag{Name: "nproc", Soft: 100, Hard: 200}) {
		t.Errorf("nproc parsed wrong: %+v", got[1])
	}
}

func TestParseUlimitsErrors(t *testing.T) {
	cases := [][]string{
		{"nofile"},          // missing '='
		{"nofile=a"},        // non-numeric soft
		{"nofile=1:2:3"},    // too many parts
		{"nofile=1:bad"},    // non-numeric hard
	}
	for _, c := range cases {
		if _, err := parseUlimits(c); err == nil {
			t.Errorf("parseUlimits(%v) expected error, got nil", c)
		}
	}
}

func TestParseRestartPolicy(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantMax  int
		wantErr  bool
	}{
		{"", "", 0, false},
		{"no", "no", 0, false},
		{"always", "always", 0, false},
		{"unless-stopped", "unless-stopped", 0, false},
		{"on-failure", "on-failure", 0, false},
		{"on-failure:5", "on-failure", 5, false},
		{"on-failure:bad", "on-failure", 0, true},
		{"bogus", "", 0, true},
	}

	for _, c := range cases {
		got, err := parseRestartPolicy(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseRestartPolicy(%q) expected error, got nil", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRestartPolicy(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got.Name != c.wantName || got.MaximumRetryCount != c.wantMax {
			t.Errorf("parseRestartPolicy(%q) = %+v; want name=%q max=%d", c.in, got, c.wantName, c.wantMax)
		}
	}
}
