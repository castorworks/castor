package secretbox

import (
	"errors"
	"strings"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	ring, _ := NewKeyring(strings.Repeat("k", 32))
	box, err := ring.Box("totp")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := box.Seal("JBSWY3DPEHPK3PXP")
	b, _ := box.Seal("JBSWY3DPEHPK3PXP")
	if a == b || !strings.HasPrefix(a, "v1:") || strings.Contains(a, "JBSWY3DPEHPK3PXP") {
		t.Fatalf("ciphertexts must be prefixed, randomized and opaque: %q %q", a, b)
	}
	if plain, err := box.Open(a); err != nil || plain != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("Open = %q, %v", plain, err)
	}
}

func TestOpenRejectsWrongKeyPurposeAndTampering(t *testing.T) {
	ring, _ := NewKeyring(strings.Repeat("k", 32))
	other, _ := NewKeyring(strings.Repeat("x", 32))
	totp, _ := ring.Box("totp")
	oidc, _ := ring.Box("oidc")
	foreign, _ := other.Box("totp")
	sealed, _ := totp.Seal("secret")

	for name, box := range map[string]*Box{"other purpose": oidc, "other master key": foreign} {
		if _, err := box.Open(sealed); !errors.Is(err, ErrDecrypt) {
			t.Errorf("%s: err = %v, want ErrDecrypt", name, err)
		}
	}
	tampered := sealed[:len(sealed)-2] + "AA"
	for _, bad := range []string{tampered, "secret", "v1:not-base64!", "v1:"} {
		if _, err := totp.Open(bad); !errors.Is(err, ErrDecrypt) {
			t.Errorf("Open(%q) err = %v", bad, err)
		}
	}
	if _, err := NewKeyring(""); err == nil {
		t.Error("an empty master key must be rejected")
	}
}
