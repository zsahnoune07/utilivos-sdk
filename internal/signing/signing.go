package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"utilivos/sdk/internal/util"
)

type KeyPair struct {
	PrivatePath string `json:"private_path"`
	PublicPath  string `json:"public_path"`
}

func DefaultKeyPaths() (privatePath, publicPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	base := filepath.Join(home, ".config", "utilivos", "keys")
	return filepath.Join(base, "dev_ed25519_private.pem"), filepath.Join(base, "dev_ed25519_public.pem"), nil
}

func GenerateKeyPair(outDir, name string) (KeyPair, error) {
	if strings.TrimSpace(name) == "" {
		name = "dev_ed25519"
	}
	outDir, err := util.ExpandHome(outDir)
	if err != nil {
		return KeyPair{}, err
	}
	if strings.TrimSpace(outDir) == "" {
		priv, _, err := DefaultKeyPaths()
		if err != nil {
			return KeyPair{}, err
		}
		outDir = filepath.Dir(priv)
	}
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return KeyPair{}, err
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return KeyPair{}, err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return KeyPair{}, err
	}

	privPath := filepath.Join(outDir, name+"_private.pem")
	pubPath := filepath.Join(outDir, name+"_public.pem")

	if err := os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}), 0o600); err != nil {
		return KeyPair{}, err
	}
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), 0o644); err != nil {
		return KeyPair{}, err
	}

	return KeyPair{PrivatePath: privPath, PublicPath: pubPath}, nil
}

func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	path, err := util.ExpandHome(path)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode private key PEM")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := k.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not ed25519")
	}
	return priv, nil
}

func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	path, err := util.ExpandHome(path)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode public key PEM")
	}
	k, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := k.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("public key is not ed25519")
	}
	return pub, nil
}

func SignFile(privateKeyPath, filePath string) ([]byte, error) {
	priv, err := LoadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	sig := ed25519.Sign(priv, digest[:])
	return sig, nil
}

func WriteSignature(path string, sig []byte) error {
	encoded := base64.StdEncoding.EncodeToString(sig) + "\n"
	return os.WriteFile(path, []byte(encoded), 0o644)
}

func ReadSignature(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func VerifyFileSignature(publicKeyPath, filePath, signaturePath string) error {
	pub, err := LoadPublicKey(publicKeyPath)
	if err != nil {
		return err
	}
	sig, err := ReadSignature(signaturePath)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(b)
	if !ed25519.Verify(pub, digest[:], sig) {
		return errors.New("signature verification failed")
	}
	return nil
}

func ResolvePrivateKey(path string) (string, error) {
	if strings.TrimSpace(path) != "" {
		return util.ExpandHome(path)
	}
	env := strings.TrimSpace(os.Getenv("UTILIVO_PRIVATE_KEY"))
	if env != "" {
		return util.ExpandHome(env)
	}
	keyFile := strings.TrimSpace(os.Getenv("UTILIVO_PRIVATE_KEY_FILE"))
	if keyFile != "" {
		return util.ExpandHome(keyFile)
	}
	priv, _, err := DefaultKeyPaths()
	if err != nil {
		return "", err
	}
	if !util.Exists(priv) {
		return "", fmt.Errorf("no signing key found; run `utilivos keygen` or pass --key")
	}
	return priv, nil
}
