package signing_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/utilivo/utilivo-sdk/internal/signing"
)

func TestSignAndVerify(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pair, err := signing.GenerateKeyPair(dir, "test")
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	artifact := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(artifact, []byte("hello utilivo"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	sig, err := signing.SignFile(pair.PrivatePath, artifact)
	if err != nil {
		t.Fatalf("sign file: %v", err)
	}
	sigPath := filepath.Join(dir, "artifact.sig")
	if err := signing.WriteSignature(sigPath, sig); err != nil {
		t.Fatalf("write signature: %v", err)
	}

	if err := signing.VerifyFileSignature(pair.PublicPath, artifact, sigPath); err != nil {
		t.Fatalf("verify signature: %v", err)
	}

	if err := os.WriteFile(artifact, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("tamper artifact: %v", err)
	}
	if err := signing.VerifyFileSignature(pair.PublicPath, artifact, sigPath); err == nil {
		t.Fatal("expected signature verification failure after tamper")
	}
}
