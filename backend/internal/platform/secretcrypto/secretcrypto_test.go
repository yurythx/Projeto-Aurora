package secretcrypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

func testKey() string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", KeySize)))
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c, err := NewFromBase64Key(testKey())
	if err != nil {
		t.Fatalf("NewFromBase64Key: %v", err)
	}

	const plaintext = "s3gr3d0-do-client-keycloak"
	enc, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == plaintext {
		t.Fatal("Encrypt: ciphertext igual ao plaintext — não cifrou nada")
	}

	got, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("Decrypt: got %q, want %q", got, plaintext)
	}
}

func TestEncryptDecrypt_EmptyStringIsSentinel(t *testing.T) {
	c, err := NewFromBase64Key(testKey())
	if err != nil {
		t.Fatalf("NewFromBase64Key: %v", err)
	}

	enc, err := c.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt(\"\"): %v", err)
	}
	if enc != "" {
		t.Fatalf("Encrypt(\"\") deveria devolver \"\", devolveu %q", enc)
	}

	got, err := c.Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt(\"\"): %v", err)
	}
	if got != "" {
		t.Fatalf("Decrypt(\"\") deveria devolver \"\", devolveu %q", got)
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	c1, err := NewFromBase64Key(testKey())
	if err != nil {
		t.Fatalf("NewFromBase64Key: %v", err)
	}
	otherKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("z", KeySize)))
	c2, err := NewFromBase64Key(otherKey)
	if err != nil {
		t.Fatalf("NewFromBase64Key (outra chave): %v", err)
	}

	enc, err := c1.Encrypt("segredo")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := c2.Decrypt(enc); err != ErrInvalidCiphertext {
		t.Fatalf("Decrypt com chave errada: got err=%v, want ErrInvalidCiphertext", err)
	}
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	c, err := NewFromBase64Key(testKey())
	if err != nil {
		t.Fatalf("NewFromBase64Key: %v", err)
	}

	enc, err := c.Encrypt("segredo")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // adultera o último byte (parte do tag de autenticação do GCM)
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := c.Decrypt(tampered); err != ErrInvalidCiphertext {
		t.Fatalf("Decrypt de ciphertext adulterado: got err=%v, want ErrInvalidCiphertext", err)
	}
}

func TestNewFromBase64Key_RejectsWrongSize(t *testing.T) {
	shortKey := base64.StdEncoding.EncodeToString([]byte("muito-curta"))
	if _, err := NewFromBase64Key(shortKey); err == nil {
		t.Fatal("esperava erro para chave com tamanho != 32 bytes")
	}
}

func TestNewFromBase64Key_RejectsInvalidBase64(t *testing.T) {
	if _, err := NewFromBase64Key("não-é-base64!!!"); err == nil {
		t.Fatal("esperava erro para valor não-base64")
	}
}
