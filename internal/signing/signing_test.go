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

func TestResolvePrivateKeyFromFileEnv(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join(dir, "custom.pem")
	if err := os.WriteFile(want, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UTILIVO_PRIVATE_KEY", "")
	t.Setenv("UTILIVO_PRIVATE_KEY_FILE", want)

	got, err := signing.ResolvePrivateKey("")
	if err != nil {
		t.Fatalf("ResolvePrivateKey: %v", err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolvePrivateKeyEnvOverFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "from-env.pem")
	fileEnvPath := filepath.Join(dir, "from-file.pem")
	for _, p := range []string{envPath, fileEnvPath} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("UTILIVO_PRIVATE_KEY", envPath)
	t.Setenv("UTILIVO_PRIVATE_KEY_FILE", fileEnvPath)

	got, err := signing.ResolvePrivateKey("")
	if err != nil {
		t.Fatalf("ResolvePrivateKey: %v", err)
	}
	if got != envPath {
		t.Fatalf("UTILIVO_PRIVATE_KEY should win: got %q want %q", got, envPath)
	}
}
