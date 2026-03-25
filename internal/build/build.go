package build

import (
	"fmt"
	"path/filepath"

	"github.com/utilivo/utilivo-sdk/internal/archive"
	"github.com/utilivo/utilivo-sdk/internal/manifest"
	"github.com/utilivo/utilivo-sdk/internal/signing"
	plugintesting "github.com/utilivo/utilivo-sdk/internal/testing"
	"github.com/utilivo/utilivo-sdk/internal/util"
	"github.com/utilivo/utilivo-sdk/internal/verify"
)

type Artifacts struct {
	Root          string `json:"root"`
	DistDir       string `json:"dist_dir"`
	ArchivePath   string `json:"archive_path"`
	ManifestPath  string `json:"manifest_path"`
	SHA256Path    string `json:"sha256_path"`
	SignaturePath string `json:"signature_path,omitempty"`
	Hash          string `json:"sha256"`
	Slug          string `json:"slug"`
	Version       string `json:"version"`
}

type BuildOptions struct {
	Root       string
	OutDir     string
	SkipTests  bool
	Strict     bool
	PrivateKey string
}

type BuildResult struct {
	Artifacts    Artifacts             `json:"artifacts"`
	VerifyResult verify.Result         `json:"verify"`
	TestResult   *plugintesting.Result `json:"test,omitempty"`
}

func Pack(root, outDir string) (Artifacts, error) {
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return Artifacts{}, err
	}
	m, err := manifest.Load(filepath.Join(absRoot, "manifest.json"))
	if err != nil {
		return Artifacts{}, err
	}
	if outDir == "" {
		outDir = filepath.Join(absRoot, "dist")
	} else if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(absRoot, outDir)
	}
	absOut, err := filepath.Abs(filepath.Clean(outDir))
	if err != nil {
		return Artifacts{}, err
	}
	if err := util.EnsureDir(absOut); err != nil {
		return Artifacts{}, err
	}

	entries, err := archive.CollectPackageFiles(absRoot)
	if err != nil {
		return Artifacts{}, err
	}
	entries, err = archive.OptimizeManifestIcon(absRoot, m, entries)
	if err != nil {
		return Artifacts{}, err
	}

	prefix := fmt.Sprintf("%s-%s", m.Slug, m.Version)
	archivePath := filepath.Join(absOut, prefix+".ubb")
	manifestPath := filepath.Join(absOut, prefix+".manifest.json")
	shaPath := filepath.Join(absOut, prefix+".sha256")

	if err := archive.CreateDeterministicZip(archivePath, entries); err != nil {
		return Artifacts{}, err
	}
	if err := util.CopyFile(filepath.Join(absRoot, "manifest.json"), manifestPath, 0o644); err != nil {
		return Artifacts{}, err
	}
	h, err := archive.SHA256File(archivePath)
	if err != nil {
		return Artifacts{}, err
	}
	if err := archive.WriteSHA256File(shaPath, archivePath, h); err != nil {
		return Artifacts{}, err
	}

	return Artifacts{
		Root:         absRoot,
		DistDir:      absOut,
		ArchivePath:  archivePath,
		ManifestPath: manifestPath,
		SHA256Path:   shaPath,
		Hash:         h,
		Slug:         m.Slug,
		Version:      m.Version,
	}, nil
}

func SignArchive(artifacts *Artifacts, privateKeyPath string) error {
	sig, err := signing.SignFile(privateKeyPath, artifacts.ArchivePath)
	if err != nil {
		return err
	}
	sigPath := filepath.Join(artifacts.DistDir, fmt.Sprintf("%s-%s.sig", artifacts.Slug, artifacts.Version))
	if err := signing.WriteSignature(sigPath, sig); err != nil {
		return err
	}
	artifacts.SignaturePath = sigPath
	return nil
}

func Build(opts BuildOptions) (BuildResult, error) {
	v, err := verify.Run(opts.Root)
	if err != nil {
		return BuildResult{}, err
	}
	if v.Report.ExitCode(opts.Strict) != util.ExitOK {
		return BuildResult{VerifyResult: v}, fmt.Errorf("verification failed")
	}

	var tr *plugintesting.Result
	if !opts.SkipTests {
		t, err := plugintesting.Run(opts.Root, plugintesting.ExternalAuto)
		if err != nil {
			return BuildResult{}, err
		}
		tr = &t
		if !t.OK {
			return BuildResult{VerifyResult: v, TestResult: tr}, fmt.Errorf("tests failed")
		}
	}

	artifacts, err := Pack(opts.Root, opts.OutDir)
	if err != nil {
		return BuildResult{VerifyResult: v, TestResult: tr}, err
	}
	if opts.PrivateKey != "" {
		if err := SignArchive(&artifacts, opts.PrivateKey); err != nil {
			return BuildResult{Artifacts: artifacts, VerifyResult: v, TestResult: tr}, err
		}
	}
	return BuildResult{Artifacts: artifacts, VerifyResult: v, TestResult: tr}, nil
}
