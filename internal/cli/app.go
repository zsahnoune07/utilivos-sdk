package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/utilivo/utilivo-sdk/internal/archive"
	"github.com/utilivo/utilivo-sdk/internal/build"
	"github.com/utilivo/utilivo-sdk/internal/devrun"
	"github.com/utilivo/utilivo-sdk/internal/doctor"
	"github.com/utilivo/utilivo-sdk/internal/manifest"
	"github.com/utilivo/utilivo-sdk/internal/scaffold"
	"github.com/utilivo/utilivo-sdk/internal/signing"
	plugintesting "github.com/utilivo/utilivo-sdk/internal/testing"
	"github.com/utilivo/utilivo-sdk/internal/util"
	"github.com/utilivo/utilivo-sdk/internal/verify"
)

func Run(argv []string, stdout, stderr io.Writer) (int, error) {
	if len(argv) == 0 {
		printRootHelp(stdout)
		return util.ExitUsage, nil
	}
	cmd := normalizeAlias(filepath.Base(argv[0]), argv)
	args := argv[1:]

	switch cmd {
	case "utilivo":
		if len(args) == 0 {
			printRootHelp(stdout)
			return util.ExitUsage, nil
		}
		return runSubcommand(args[0], args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printRootHelp(stdout)
		return util.ExitOK, nil
	case "version", "-v", "--version":
		return runSubcommand(cmd, args, stdout, stderr)
	default:
		return runSubcommand(cmd, args, stdout, stderr)
	}
}

func normalizeAlias(invocation string, argv []string) string {
	name := filepath.Base(invocation)
	if name == "create" || name == "verify" || name == "test" || name == "build" || name == "run" {
		return name
	}
	if name == "utilivo" {
		return "utilivo"
	}
	if len(argv) > 0 && filepath.Base(argv[0]) == "utilivo" {
		return "utilivo"
	}
	return name
}

func runSubcommand(cmd string, args []string, stdout, stderr io.Writer) (int, error) {
	switch cmd {
	case "create":
		return runCreate(args, stdout, stderr)
	case "verify":
		return runVerify(args, stdout, stderr)
	case "test":
		return runTest(args, stdout, stderr)
	case "pack":
		return runPack(args, stdout, stderr)
	case "sign":
		return runSign(args, stdout, stderr)
	case "build":
		return runBuild(args, stdout, stderr)
	case "run":
		return runRun(args, stdout, stderr)
	case "inspect":
		return runInspect(args, stdout, stderr)
	case "doctor":
		return runDoctor(args, stdout, stderr)
	case "keygen":
		return runKeygen(args, stdout, stderr)
	case "clean":
		return runClean(args, stdout, stderr)
	case "manifest":
		return runManifest(args, stdout, stderr)
	case "version", "-v", "--version":
		return runVersion(args, stdout, stderr)
	case "help", "-h", "--help":
		printRootHelp(stdout)
		return util.ExitOK, nil
	default:
		printRootHelp(stderr)
		return util.ExitUsage, fmt.Errorf("unknown command: %s", cmd)
	}
}

func runCreate(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	name := fs.String("name", "", "Plugin display name")
	slug := fs.String("slug", "", "Plugin slug")
	category := fs.String("category", "", "Plugin category")
	tier := fs.String("tier", "", "Plugin tier (free|premium)")
	author := fs.String("author", "", "Plugin author")
	description := fs.String("description", "", "Plugin description")
	tags := fs.String("tags", "", "Comma-separated tags")
	template := fs.String("template", "basic", "Template type (basic|widget)")
	targetPath, parseArgs := splitLeadingPositional(args)
	if err := fs.Parse(parseArgs); err != nil {
		return util.ExitUsage, err
	}
	if targetPath == "" {
		if fs.NArg() != 1 {
			return util.ExitUsage, errors.New("usage: utilivo create <plugin-dir> [flags]")
		}
		targetPath = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return util.ExitUsage, errors.New("usage: utilivo create <plugin-dir> [flags]")
	}

	options := scaffold.Options{
		TargetPath:  targetPath,
		Name:        strings.TrimSpace(*name),
		Slug:        strings.TrimSpace(*slug),
		Category:    strings.TrimSpace(*category),
		Tier:        strings.TrimSpace(*tier),
		Author:      strings.TrimSpace(*author),
		Description: strings.TrimSpace(*description),
		Template:    strings.TrimSpace(*template),
	}
	if trimmed := strings.TrimSpace(*tags); trimmed != "" {
		parts := strings.Split(trimmed, ",")
		clean := make([]string, 0, len(parts))
		for _, part := range parts {
			t := strings.TrimSpace(part)
			if t != "" {
				clean = append(clean, t)
			}
		}
		options.Tags = clean
	}

	path, err := scaffold.Create(options)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	fmt.Fprintf(stdout, "created plugin scaffold at %s\n", path)
	return util.ExitOK, nil
}

func runVerify(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	strict := fs.Bool("strict", false, "Fail on warnings")
	dir := fs.String("dir", ".", "Project directory")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}

	result, err := verify.Run(*dir)
	if err != nil {
		return util.ExitValidationError, err
	}
	if *jsonOut {
		if err := util.PrintJSON(stdout, result); err != nil {
			return util.ExitUsage, err
		}
	} else {
		printVerifyHuman(stdout, result)
	}
	return result.Report.ExitCode(*strict), nil
}

func runTest(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	dir := fs.String("dir", ".", "Project directory")
	filter := fs.String("filter", "all", "all|builtin|external")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}

	mode := plugintesting.ExternalAuto
	switch strings.ToLower(strings.TrimSpace(*filter)) {
	case "all", "auto", "":
		mode = plugintesting.ExternalAuto
	case "builtin":
		mode = plugintesting.ExternalSkip
	case "external":
		mode = plugintesting.ExternalOnly
	default:
		return util.ExitUsage, fmt.Errorf("invalid --filter value %q", *filter)
	}

	result, err := plugintesting.Run(*dir, mode)
	if err != nil {
		return util.ExitTestFailure, err
	}
	if *jsonOut {
		if err := util.PrintJSON(stdout, result); err != nil {
			return util.ExitUsage, err
		}
	} else {
		printTestHuman(stdout, result)
	}
	if !result.OK {
		return util.ExitTestFailure, nil
	}
	return util.ExitOK, nil
}

func runPack(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	dir := fs.String("dir", ".", "Project directory")
	outDir := fs.String("out", "", "Output directory")
	skipVerify := fs.Bool("skip-verify", false, "Skip verification")
	strict := fs.Bool("strict", false, "Fail on warnings during verify")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}
	if !*skipVerify {
		v, err := verify.Run(*dir)
		if err != nil {
			return util.ExitValidationError, err
		}
		if code := v.Report.ExitCode(*strict); code != util.ExitOK {
			if *jsonOut {
				_ = util.PrintJSON(stdout, v)
			} else {
				printVerifyHuman(stdout, v)
			}
			return code, nil
		}
	}
	artifacts, err := build.Pack(*dir, *outDir)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	if *jsonOut {
		if err := util.PrintJSON(stdout, artifacts); err != nil {
			return util.ExitUsage, err
		}
	} else {
		printPackHuman(stdout, artifacts)
	}
	return util.ExitOK, nil
}

func runSign(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	keyPath := fs.String("key", "", "Private key path")
	out := fs.String("out", "", "Output signature file path")
	filePath, parseArgs := splitLeadingPositional(args)
	if err := fs.Parse(parseArgs); err != nil {
		return util.ExitUsage, err
	}
	if filePath == "" {
		if fs.NArg() != 1 {
			return util.ExitUsage, errors.New("usage: utilivo sign <archive-or-file> [--key path] [--out file.sig]")
		}
		filePath = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return util.ExitUsage, errors.New("usage: utilivo sign <archive-or-file> [--key path] [--out file.sig]")
	}
	resolved, err := signing.ResolvePrivateKey(*keyPath)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	if strings.TrimSpace(*out) == "" {
		*out = filePath + ".sig"
	}
	sig, err := signing.SignFile(resolved, filePath)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	if err := signing.WriteSignature(*out, sig); err != nil {
		return util.ExitBuildFailure, err
	}

	payload := map[string]string{
		"file": filePath,
		"sig":  *out,
	}
	if *jsonOut {
		if err := util.PrintJSON(stdout, payload); err != nil {
			return util.ExitUsage, err
		}
	} else {
		fmt.Fprintf(stdout, "signature written: %s\n", *out)
	}
	return util.ExitOK, nil
}

func runBuild(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	dir := fs.String("dir", ".", "Project directory")
	outDir := fs.String("out", "", "Output directory")
	skipTests := fs.Bool("skip-tests", false, "Skip tests")
	strict := fs.Bool("strict", false, "Fail on verify warnings")
	key := fs.String("key", "", "Private key path")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}

	v, err := verify.Run(*dir)
	if err != nil {
		return util.ExitValidationError, err
	}
	if code := v.Report.ExitCode(*strict); code != util.ExitOK {
		if *jsonOut {
			_ = util.PrintJSON(stdout, v)
		} else {
			printVerifyHuman(stdout, v)
			fmt.Fprintf(stdout, "next: fix validation issues above, then retry `utilivo build --dir %s`\n", *dir)
		}
		return code, nil
	}

	var tr *plugintesting.Result
	if !*skipTests {
		t, err := plugintesting.Run(*dir, plugintesting.ExternalAuto)
		if err != nil {
			return util.ExitTestFailure, err
		}
		tr = &t
		if !t.OK {
			if *jsonOut {
				_ = util.PrintJSON(stdout, t)
			} else {
				printTestHuman(stdout, t)
				fmt.Fprintln(stdout, "next: fix failing tests above or use --skip-tests for local packaging only")
			}
			return util.ExitTestFailure, nil
		}
	}

	artifacts, err := build.Pack(*dir, *outDir)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	resolvedKey, err := signing.ResolvePrivateKey(*key)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	if err := build.SignArchive(&artifacts, resolvedKey); err != nil {
		return util.ExitBuildFailure, err
	}

	if *jsonOut {
		payload := build.BuildResult{Artifacts: artifacts, VerifyResult: v, TestResult: tr}
		if err := util.PrintJSON(stdout, payload); err != nil {
			return util.ExitUsage, err
		}
	} else {
		printBuildHuman(stdout, artifacts)
		fmt.Fprintln(stdout, "next: inspect the archive with `utilivo inspect` before shipping")
	}
	return util.ExitOK, nil
}

func runRun(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", ".", "Project directory")
	host := fs.String("host", "127.0.0.1", "Host interface for local runtime")
	port := fs.Int("port", 5173, "Port (use 0 to auto-select)")
	noOpen := fs.Bool("no-open", false, "Do not open the browser automatically")
	theme := fs.String("theme", "system", "Theme mode (system|light|dark)")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}
	if fs.NArg() > 0 {
		return util.ExitUsage, errors.New("usage: utilivo run [--dir .] [--host 127.0.0.1] [--port 5173|0] [--no-open] [--theme system|light|dark]")
	}

	themeValue := strings.ToLower(strings.TrimSpace(*theme))
	if themeValue != "system" && themeValue != "light" && themeValue != "dark" {
		return util.ExitUsage, fmt.Errorf("invalid --theme value %q (allowed: system|light|dark)", *theme)
	}

	server, err := devrun.Start(devrun.Options{
		Root:  *dir,
		Host:  strings.TrimSpace(*host),
		Port:  *port,
		Theme: themeValue,
	})
	if err != nil {
		return util.ExitBuildFailure, fmt.Errorf("run failed: %w", err)
	}

	fmt.Fprintln(stdout, "local runtime ready")
	fmt.Fprintf(stdout, "- project: %s\n", server.Root())
	fmt.Fprintf(stdout, "- entry: %s\n", server.EntryPath())
	fmt.Fprintf(stdout, "- url: %s\n", server.URL())
	fmt.Fprintf(stdout, "- theme: %s\n", themeValue)

	if !*noOpen {
		if err := server.OpenBrowser(); err != nil {
			fmt.Fprintf(stdout, "warning: could not open browser automatically (%v)\n", err)
			fmt.Fprintf(stdout, "next: open %s manually\n", server.URL())
		}
	}
	fmt.Fprintln(stdout, "press Ctrl+C to stop")

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh
	signal.Stop(signalCh)

	if err := server.Close(); err != nil {
		return util.ExitBuildFailure, fmt.Errorf("stop local runtime: %w", err)
	}

	fmt.Fprintln(stdout, "local runtime stopped")
	return util.ExitOK, nil
}

func runInspect(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	archivePath, parseArgs := splitLeadingPositional(args)
	if err := fs.Parse(parseArgs); err != nil {
		return util.ExitUsage, err
	}
	if archivePath == "" {
		if fs.NArg() != 1 {
			return util.ExitUsage, errors.New("usage: utilivo inspect <archive.ubb>")
		}
		archivePath = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return util.ExitUsage, errors.New("usage: utilivo inspect <archive.ubb>")
	}
	info, err := archive.Inspect(archivePath)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	if *jsonOut {
		if err := util.PrintJSON(stdout, info); err != nil {
			return util.ExitUsage, err
		}
	} else {
		printInspectHuman(stdout, info)
	}
	return util.ExitOK, nil
}

func runDoctor(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	dir := fs.String("dir", ".", "Project directory")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}
	r := doctor.Run(*dir)
	if *jsonOut {
		if err := util.PrintJSON(stdout, r); err != nil {
			return util.ExitUsage, err
		}
	} else {
		for _, check := range r.Checks {
			fmt.Fprintf(stdout, "[%s] %s: %s\n", strings.ToUpper(string(check.Status)), check.Name, check.Details)
		}
	}
	if !r.OK {
		return util.ExitUsage, nil
	}
	return util.ExitOK, nil
}

func runKeygen(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	out := fs.String("out", "", "Output directory (default ~/.config/utilivo/keys)")
	name := fs.String("name", "dev_ed25519", "Key name prefix")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}
	pair, err := signing.GenerateKeyPair(*out, *name)
	if err != nil {
		return util.ExitBuildFailure, err
	}
	if *jsonOut {
		if err := util.PrintJSON(stdout, pair); err != nil {
			return util.ExitUsage, err
		}
	} else {
		fmt.Fprintf(stdout, "private key: %s\n", pair.PrivatePath)
		fmt.Fprintf(stdout, "public key: %s\n", pair.PublicPath)
	}
	return util.ExitOK, nil
}

func runClean(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", ".", "Project directory")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}
	base, err := filepath.Abs(*dir)
	if err != nil {
		return util.ExitUsage, err
	}
	dist := filepath.Join(base, "dist")
	if !util.Exists(dist) {
		fmt.Fprintln(stdout, "nothing to clean")
		return util.ExitOK, nil
	}
	entries, err := os.ReadDir(dist)
	if err != nil {
		return util.ExitUsage, err
	}
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".ubb") || strings.HasSuffix(name, ".utilivo.zip") || strings.HasSuffix(name, ".sig") || strings.HasSuffix(name, ".sha256") || strings.HasSuffix(name, ".manifest.json") {
			if err := os.Remove(filepath.Join(dist, name)); err == nil {
				removed++
			}
		}
	}
	fmt.Fprintf(stdout, "removed %d artifact(s) from %s\n", removed, dist)
	return util.ExitOK, nil
}

func runManifest(args []string, stdout, _ io.Writer) (int, error) {
	if len(args) == 0 {
		return util.ExitUsage, errors.New("usage: utilivo manifest <lint|format|bump>")
	}
	sub := args[0]
	subArgs := args[1:]
	mPath := "manifest.json"

	switch sub {
	case "lint":
		m, err := manifest.Load(mPath)
		if err != nil {
			return util.ExitValidationError, err
		}
		r := manifest.Validate(m)
		if err := util.PrintJSON(stdout, r); err != nil {
			return util.ExitUsage, err
		}
		return r.ExitCode(false), nil
	case "format":
		m, err := manifest.Load(mPath)
		if err != nil {
			return util.ExitValidationError, err
		}
		if err := manifest.Save(mPath, m); err != nil {
			return util.ExitUsage, err
		}
		fmt.Fprintln(stdout, "manifest.json formatted")
		return util.ExitOK, nil
	case "bump":
		if len(subArgs) != 1 {
			return util.ExitUsage, errors.New("usage: utilivo manifest bump <patch|minor|major>")
		}
		m, err := manifest.Load(mPath)
		if err != nil {
			return util.ExitValidationError, err
		}
		version, err := manifest.BumpVersion(m.Version, subArgs[0])
		if err != nil {
			return util.ExitUsage, err
		}
		m.Version = version
		if err := manifest.Save(mPath, m); err != nil {
			return util.ExitUsage, err
		}
		fmt.Fprintf(stdout, "manifest version bumped to %s\n", version)
		return util.ExitOK, nil
	default:
		return util.ExitUsage, fmt.Errorf("unknown manifest subcommand: %s", sub)
	}
}

func runVersion(args []string, stdout, _ io.Writer) (int, error) {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "Output JSON")
	if err := fs.Parse(args); err != nil {
		return util.ExitUsage, err
	}
	if fs.NArg() > 0 {
		return util.ExitUsage, errors.New("usage: utilivo version [--json]")
	}
	if *jsonOut {
		payload := map[string]string{"version": util.Version}
		if err := util.PrintJSON(stdout, payload); err != nil {
			return util.ExitUsage, err
		}
	} else {
		fmt.Fprintln(stdout, util.Version)
	}
	return util.ExitOK, nil
}

func printRootHelp(w io.Writer) {
	fmt.Fprint(w, `utilivo - Utilivo plugin SDK CLI

Usage:
  utilivo <command> [options]

Commands:
  create     Create a strict plugin scaffold
  verify     Validate plugin structure and manifest
  test       Run built-in and optional external tests
  run        Start local browser-ready dev runtime
  pack       Build deterministic unsigned .ubb archive artifacts
  sign       Sign an archive or file with Ed25519 key
  build      Verify + test + pack + sign
  inspect    Inspect built archive contents and metadata
  doctor     Check local SDK environment
  keygen     Generate Ed25519 dev signing keys
  clean      Remove generated dist artifacts
  manifest   Manifest helpers (lint|format|bump)
  version    Print CLI version

Aliases:
  create, verify, test, run, build can be invoked as wrapper commands if installed.
`)
}

func printVerifyHuman(w io.Writer, result verify.Result) {
	fmt.Fprintf(w, "verify: errors=%d warnings=%d\n", result.Report.Errors, result.Report.Warnings)
	for _, issue := range result.Report.Issues {
		if issue.Path != "" {
			fmt.Fprintf(w, "- [%s] %s (%s): %s\n", strings.ToUpper(string(issue.Severity)), issue.Code, issue.Path, issue.Message)
		} else {
			fmt.Fprintf(w, "- [%s] %s: %s\n", strings.ToUpper(string(issue.Severity)), issue.Code, issue.Message)
		}
	}
}

func printTestHuman(w io.Writer, result plugintesting.Result) {
	fmt.Fprintf(w, "test: ok=%t tests=%d failures=%d\n", result.OK, result.Tests, result.Failures)
	for _, issue := range result.Issues {
		if issue.Path != "" {
			fmt.Fprintf(w, "- [%s] %s (%s): %s\n", strings.ToUpper(string(issue.Severity)), issue.Code, issue.Path, issue.Message)
		} else {
			fmt.Fprintf(w, "- [%s] %s: %s\n", strings.ToUpper(string(issue.Severity)), issue.Code, issue.Message)
		}
	}
	if result.ExternalRun {
		fmt.Fprintf(w, "external tests: passed=%t\n", result.ExternalPassed)
	}
}

func printPackHuman(w io.Writer, artifacts build.Artifacts) {
	fmt.Fprintln(w, "pack complete")
	fmt.Fprintf(w, "- archive: %s\n", artifacts.ArchivePath)
	fmt.Fprintf(w, "- manifest: %s\n", artifacts.ManifestPath)
	fmt.Fprintf(w, "- sha256: %s\n", artifacts.SHA256Path)
}

func printBuildHuman(w io.Writer, artifacts build.Artifacts) {
	fmt.Fprintln(w, "build complete")
	fmt.Fprintf(w, "- archive: %s\n", artifacts.ArchivePath)
	fmt.Fprintf(w, "- manifest: %s\n", artifacts.ManifestPath)
	fmt.Fprintf(w, "- sha256: %s\n", artifacts.SHA256Path)
	fmt.Fprintf(w, "- sig: %s\n", artifacts.SignaturePath)
}

func printInspectHuman(w io.Writer, info archive.Inspection) {
	fmt.Fprintf(w, "archive: %s\n", info.ArchivePath)
	fmt.Fprintf(w, "sha256: %s\n", info.SHA256)
	fmt.Fprintf(w, "entries: %d\n", info.EntryCount)
	if info.Manifest != nil {
		fmt.Fprintf(w, "plugin: %s (%s)\n", info.Manifest.Slug, info.Manifest.Version)
		fmt.Fprintf(w, "tier/category: %s / %s\n", info.Manifest.Tier, info.Manifest.Category)
	}
	fmt.Fprintf(w, "signature present: %t\n", info.HasSignature)
}

func splitLeadingPositional(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}
