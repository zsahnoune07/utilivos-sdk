package devrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/utilivo/utilivo-sdk/internal/manifest"
	"github.com/utilivo/utilivo-sdk/internal/util"
)

const (
	defaultHost            = "127.0.0.1"
	defaultTheme           = "system"
	livePollTimeout        = 25 * time.Second
	revisionPollInterval   = 200 * time.Millisecond
	watcherPollInterval    = 700 * time.Millisecond
	defaultShutdownTimeout = 3 * time.Second
)

type Options struct {
	Root  string
	Host  string
	Port  int
	Theme string
}

type LiveResponse struct {
	Revision string `json:"revision"`
	Changed  bool   `json:"changed"`
}

type Server struct {
	root       string
	entryRel   string
	entryURL   string
	host       string
	port       int
	theme      string
	manifest   manifest.Manifest
	startedAt  time.Time
	baseURL    string
	fileServer http.Handler

	httpServer *http.Server
	listener   net.Listener

	revision *revisionTracker
	stopCh   chan struct{}
	watchWg  sync.WaitGroup
}

type revisionTracker struct {
	mu    sync.RWMutex
	value string
}

func (t *revisionTracker) Set(value string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.value = value
}

func (t *revisionTracker) Current() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.value
}

func Start(opts Options) (*Server, error) {
	absRoot, err := normalizeRoot(opts.Root)
	if err != nil {
		return nil, err
	}

	m, err := manifest.Load(filepath.Join(absRoot, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("load manifest.json: %w", err)
	}
	if report := manifest.Validate(m); report.Errors > 0 {
		return nil, fmt.Errorf("manifest validation failed with %d error(s); run `utilivo verify --dir %s`", report.Errors, absRoot)
	}

	host := strings.TrimSpace(opts.Host)
	if host == "" {
		host = defaultHost
	}

	port := opts.Port
	if port < 0 || port > 65535 {
		return nil, errors.New("--port must be between 0 and 65535")
	}

	theme := strings.ToLower(strings.TrimSpace(opts.Theme))
	if theme == "" {
		theme = defaultTheme
	}
	if theme != "system" && theme != "light" && theme != "dark" {
		return nil, fmt.Errorf("invalid --theme value %q (allowed: system|light|dark)", theme)
	}

	entryRel := strings.TrimSpace(m.Entry)
	if entryRel == "" {
		entryRel = "tool/index.html"
	}
	entryRel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(entryRel)))
	if strings.HasPrefix(entryRel, "../") || entryRel == ".." {
		return nil, fmt.Errorf("manifest entry %q escapes project root", m.Entry)
	}
	entryAbs := filepath.Join(absRoot, filepath.FromSlash(entryRel))
	if !util.Exists(entryAbs) {
		return nil, fmt.Errorf("manifest entry is missing: %s", entryRel)
	}
	st, err := os.Stat(entryAbs)
	if err != nil {
		return nil, fmt.Errorf("read manifest entry %s: %w", entryRel, err)
	}
	if st.IsDir() {
		return nil, fmt.Errorf("manifest entry must point to a file, got directory: %s", entryRel)
	}

	rev, err := computeRevision(absRoot)
	if err != nil {
		return nil, fmt.Errorf("compute initial project revision: %w", err)
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("start local runtime server: %w", err)
	}

	actualPort := port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}

	server := &Server{
		root:       absRoot,
		entryRel:   entryRel,
		entryURL:   "/" + entryRel,
		host:       host,
		port:       actualPort,
		theme:      theme,
		manifest:   m,
		startedAt:  time.Now().UTC(),
		baseURL:    fmt.Sprintf("http://%s:%d", host, actualPort),
		fileServer: http.FileServer(http.Dir(absRoot)),
		revision:   &revisionTracker{value: rev},
		stopCh:     make(chan struct{}),
		listener:   listener,
	}

	httpServer := &http.Server{
		Handler:           http.HandlerFunc(server.route),
		ReadHeaderTimeout: 5 * time.Second,
	}
	server.httpServer = httpServer

	server.watchWg.Add(1)
	go server.watchFiles()

	go func() {
		_ = httpServer.Serve(listener)
	}()

	return server, nil
}

func normalizeRoot(root string) (string, error) {
	cleaned := strings.TrimSpace(root)
	if cleaned == "" {
		cleaned = "."
	}
	absRoot, err := filepath.Abs(filepath.Clean(cleaned))
	if err != nil {
		return "", err
	}
	if !util.IsDir(absRoot) {
		return "", fmt.Errorf("project directory does not exist: %s", absRoot)
	}
	return absRoot, nil
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean("/" + r.URL.Path)

	switch {
	case cleanPath == "/":
		s.serveShell(w, r)
		return
	case cleanPath == "/__utilivo__/manifest":
		s.serveManifest(w, r)
		return
	case cleanPath == "/__utilivo__/revision":
		s.serveRevision(w, r)
		return
	case cleanPath == "/__utilivo__/live":
		s.serveLive(w, r)
		return
	case strings.TrimPrefix(cleanPath, "/") == s.entryRel:
		s.serveEntryWithPrelude(w, r)
		return
	default:
		s.serveStatic(w, r, cleanPath)
		return
	}
}

func (s *Server) URL() string {
	return s.baseURL
}

func (s *Server) Root() string {
	return s.root
}

func (s *Server) EntryPath() string {
	return s.entryRel
}

func (s *Server) Revision() string {
	return s.revision.Current()
}

func (s *Server) OpenBrowser() error {
	return OpenBrowser(s.baseURL)
}

func (s *Server) Close() error {
	close(s.stopCh)
	s.watchWg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Server) serveShell(w http.ResponseWriter, _ *http.Request) {
	setNoStore(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	revision := s.revision.Current()
	entryURL := fmt.Sprintf("%s?uv_rev=%s", s.entryURL, urlQueryEscape(revision))
	permissionText := strings.Join(s.manifest.Permissions, ", ")
	if strings.TrimSpace(permissionText) == "" {
		permissionText = "none"
	}

	_, _ = fmt.Fprintf(w, "<!doctype html>\n")
	_, _ = fmt.Fprintf(w, "<html lang=\"en\">\n")
	_, _ = fmt.Fprintf(w, "  <head>\n")
	_, _ = fmt.Fprintf(w, "    <meta charset=\"utf-8\" />\n")
	_, _ = fmt.Fprintf(w, "    <meta name=\"viewport\" content=\"width=device-width,initial-scale=1\" />\n")
	_, _ = fmt.Fprintf(w, "    <title>%s · utilivo run</title>\n", html.EscapeString(s.manifest.Name))
	_, _ = fmt.Fprintf(w, "    <style>\n")
	_, _ = fmt.Fprintf(w, "      :root { color-scheme: light dark; }\n")
	_, _ = fmt.Fprintf(w, "      * { box-sizing: border-box; }\n")
	_, _ = fmt.Fprintf(w, "      body { margin: 0; font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, sans-serif; background: #0b1220; color: #dbe7ff; min-height: 100vh; }\n")
	_, _ = fmt.Fprintf(w, "      .uv-dev-shell { display: flex; flex-direction: column; min-height: 100vh; }\n")
	_, _ = fmt.Fprintf(w, "      .uv-dev-shell__bar { display: grid; gap: 6px; padding: 10px 14px; border-bottom: 1px solid #26324a; background: #101a2c; }\n")
	_, _ = fmt.Fprintf(w, "      .uv-dev-shell__title { font-size: 13px; font-weight: 700; margin: 0; }\n")
	_, _ = fmt.Fprintf(w, "      .uv-dev-shell__meta { font-size: 12px; opacity: 0.9; margin: 0; }\n")
	_, _ = fmt.Fprintf(w, "      .uv-dev-shell__frame { flex: 1; border: 0; width: 100%%; background: #fff; }\n")
	_, _ = fmt.Fprintf(w, "      code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }\n")
	_, _ = fmt.Fprintf(w, "    </style>\n")
	_, _ = fmt.Fprintf(w, "  </head>\n")
	_, _ = fmt.Fprintf(w, "  <body>\n")
	_, _ = fmt.Fprintf(w, "    <main class=\"uv-dev-shell\">\n")
	_, _ = fmt.Fprintf(w, "      <header class=\"uv-dev-shell__bar\">\n")
	_, _ = fmt.Fprintf(w, "        <p class=\"uv-dev-shell__title\">utilivo run · %s (%s)</p>\n", html.EscapeString(s.manifest.Slug), html.EscapeString(s.manifest.Version))
	_, _ = fmt.Fprintf(w, "        <p class=\"uv-dev-shell__meta\">entry: <code>%s</code> · theme: <code>%s</code> · permissions: <code>%s</code></p>\n", html.EscapeString(s.entryRel), html.EscapeString(s.theme), html.EscapeString(permissionText))
	_, _ = fmt.Fprintf(w, "      </header>\n")
	_, _ = fmt.Fprintf(w, "      <iframe id=\"uv-dev-frame\" class=\"uv-dev-shell__frame\" src=\"%s\" title=\"Utilivo plugin runtime\"></iframe>\n", html.EscapeString(entryURL))
	_, _ = fmt.Fprintf(w, "    </main>\n")
	_, _ = fmt.Fprintf(w, "  </body>\n")
	_, _ = fmt.Fprintf(w, "</html>\n")
}

func (s *Server) serveManifest(w http.ResponseWriter, _ *http.Request) {
	setNoStore(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"manifest": s.manifest,
		"runtime": map[string]any{
			"mode":       "dev",
			"entry":      s.entryRel,
			"theme":      s.theme,
			"project":    s.root,
			"revision":   s.revision.Current(),
			"started_at": s.startedAt.Format(time.RFC3339),
		},
	})
}

func (s *Server) serveRevision(w http.ResponseWriter, _ *http.Request) {
	setNoStore(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"revision": s.revision.Current()})
}

func (s *Server) serveLive(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	since := strings.TrimSpace(r.URL.Query().Get("since"))
	ctx, cancel := context.WithTimeout(r.Context(), livePollTimeout)
	defer cancel()
	revision, changed := s.waitForRevision(ctx, since)

	_ = json.NewEncoder(w).Encode(LiveResponse{
		Revision: revision,
		Changed:  changed,
	})
}

func (s *Server) waitForRevision(ctx context.Context, since string) (string, bool) {
	current := s.revision.Current()
	if since == "" || current != since {
		return current, current != since
	}

	ticker := time.NewTicker(revisionPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return s.revision.Current(), false
		case <-ticker.C:
			revision := s.revision.Current()
			if revision != since {
				return revision, true
			}
		}
	}
}

func (s *Server) serveEntryWithPrelude(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	entryAbs := filepath.Join(s.root, filepath.FromSlash(s.entryRel))
	b, err := os.ReadFile(entryAbs)
	if err != nil {
		http.Error(w, "unable to load entry html", http.StatusInternalServerError)
		return
	}

	injected := injectPrelude(string(b), s.buildPreludeScript(r.Host))
	_, _ = w.Write([]byte(injected))
}

func (s *Server) buildPreludeScript(requestHost string) string {
	hostPayload := map[string]any{
		"mode": "dev",
		"manifest": map[string]any{
			"id":          s.manifest.ID,
			"slug":        s.manifest.Slug,
			"name":        s.manifest.Name,
			"version":     s.manifest.Version,
			"entry":       s.entryRel,
			"permissions": s.manifest.Permissions,
			"ui_type":     s.manifest.UIType,
		},
		"runtime": map[string]any{
			"project_root":  s.root,
			"entry_path":    s.entryRel,
			"live_endpoint": "/__utilivo__/live",
			"revision":      s.revision.Current(),
			"origin":        fmt.Sprintf("http://%s", requestHost),
			"started_at":    s.startedAt.Format(time.RFC3339),
			"sandbox": map[string]any{
				"permissions":             s.manifest.Permissions,
				"content_security_policy": s.manifest.ContentSecurityPolicy,
			},
		},
		"theme": map[string]any{
			"requested": s.theme,
		},
	}
	payloadJSON, _ := json.Marshal(hostPayload)

	return fmt.Sprintf(`
(() => {
  const hostPayload = %s;

  function deepFreeze(value) {
    if (!value || typeof value !== "object" || Object.isFrozen(value)) {
      return value;
    }
    Object.freeze(value);
    Object.keys(value).forEach((key) => deepFreeze(value[key]));
    return value;
  }

  const requestedTheme = hostPayload.theme?.requested || "system";
  const resolvedTheme = requestedTheme === "system"
    ? (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
    : requestedTheme;

  if (requestedTheme === "light" || requestedTheme === "dark") {
    document.documentElement.setAttribute("data-uv-theme", resolvedTheme);
  }

  const utilivoHost = deepFreeze({
    mode: "dev",
    manifest: hostPayload.manifest,
    runtime: hostPayload.runtime,
    theme: {
      requested: requestedTheme,
      resolved: resolvedTheme,
    },
  });

  if (!Object.prototype.hasOwnProperty.call(window, "UtilivoHost")) {
    Object.defineProperty(window, "UtilivoHost", {
      value: utilivoHost,
      writable: false,
      configurable: false,
      enumerable: true,
    });
  }

  const applyThemePayload = () => {
    if (window.UtilivoUI && typeof window.UtilivoUI.applyThemePayload === "function") {
      window.UtilivoUI.applyThemePayload({ mode: resolvedTheme, tokens: {} });
    }
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", applyThemePayload, { once: true });
  } else {
    applyThemePayload();
  }

  function ensureDevErrorOverlay() {
    let overlay = document.getElementById("__uv-dev-error-overlay");
    if (overlay) {
      return overlay;
    }

    overlay = document.createElement("div");
    overlay.id = "__uv-dev-error-overlay";
    overlay.style.position = "fixed";
    overlay.style.right = "12px";
    overlay.style.bottom = "12px";
    overlay.style.maxWidth = "min(760px, calc(100%% - 24px))";
    overlay.style.maxHeight = "70vh";
    overlay.style.overflow = "auto";
    overlay.style.zIndex = "2147483647";
    overlay.style.padding = "12px";
    overlay.style.borderRadius = "10px";
    overlay.style.background = "#2a0f17";
    overlay.style.color = "#ffe8ef";
    overlay.style.border = "1px solid #dd4f77";
    overlay.style.font = "12px/1.45 ui-monospace, SFMono-Regular, Menlo, monospace";
    overlay.style.display = "none";

    const title = document.createElement("strong");
    title.textContent = "utilivo run error";
    title.style.display = "block";
    title.style.marginBottom = "8px";

    const message = document.createElement("pre");
    message.id = "__uv-dev-error-message";
    message.style.whiteSpace = "pre-wrap";
    message.style.margin = "0";

    const note = document.createElement("p");
    note.style.margin = "10px 0 0";
    note.style.font = "11px/1.35 ui-sans-serif, system-ui, sans-serif";
    note.textContent = "Fix the error and save. utilivo run reloads the tool automatically.";

    overlay.appendChild(title);
    overlay.appendChild(message);
    overlay.appendChild(note);
    document.body.appendChild(overlay);
    return overlay;
  }

  function showRuntimeError(kind, detail) {
    const overlay = ensureDevErrorOverlay();
    const message = document.getElementById("__uv-dev-error-message");
    if (message) {
      message.textContent = "[" + kind + "]\n" + detail;
    }
    overlay.style.display = "block";
  }

  window.addEventListener("error", (event) => {
    const detail = [
      event.message || "Unknown runtime error",
      event.filename ? String(event.filename) + ":" + String(event.lineno || 0) + ":" + String(event.colno || 0) : "",
      event.error && event.error.stack ? event.error.stack : "",
    ].filter(Boolean).join("\n");
    showRuntimeError("error", detail);
  });

  window.addEventListener("unhandledrejection", (event) => {
    const reason = event.reason;
    const detail = reason && reason.stack ? reason.stack : String(reason);
    showRuntimeError("unhandledrejection", detail);
  });

  let since = hostPayload.runtime.revision;
  let stopped = false;

  async function wait(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  async function pollLive() {
    while (!stopped) {
      try {
        const response = await fetch(hostPayload.runtime.live_endpoint + "?since=" + encodeURIComponent(since), {
          cache: "no-store",
        });
        if (!response.ok) {
          await wait(350);
          continue;
        }

        const payload = await response.json();
        if (payload && typeof payload.revision === "string" && payload.revision.length > 0) {
          if (payload.changed && payload.revision !== since) {
            window.location.reload();
            return;
          }
          since = payload.revision;
        }
      } catch (_) {
        await wait(450);
      }
    }
  }

  pollLive();
  window.addEventListener("beforeunload", () => {
    stopped = true;
  });
})();
`, string(payloadJSON))
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request, cleanedPath string) {
	setNoStore(w)
	req := r.Clone(r.Context())
	req.URL.Path = cleanedPath
	s.fileServer.ServeHTTP(w, req)
}

func (s *Server) watchFiles() {
	defer s.watchWg.Done()
	ticker := time.NewTicker(watcherPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			rev, err := computeRevision(s.root)
			if err != nil {
				continue
			}
			s.revision.Set(rev)
		}
	}
}

func computeRevision(root string) (string, error) {
	records := make([]string, 0, 128)

	manifestPath := filepath.Join(root, "manifest.json")
	if stat, err := os.Stat(manifestPath); err == nil && !stat.IsDir() {
		records = append(records, statRecord("manifest.json", stat))
	}

	toolRoot := filepath.Join(root, "tool")
	if util.IsDir(toolRoot) {
		walkErr := filepath.WalkDir(toolRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := strings.ToLower(strings.TrimSpace(d.Name()))
				if name == ".git" || name == "node_modules" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}

			stat, err := d.Info()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			records = append(records, statRecord(filepath.ToSlash(rel), stat))
			return nil
		})
		if walkErr != nil {
			return "", walkErr
		}
	}

	sort.Strings(records)
	h := sha256.New()
	for _, record := range records {
		_, _ = h.Write([]byte(record))
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func statRecord(rel string, stat fs.FileInfo) string {
	return fmt.Sprintf("%s|%d|%d", rel, stat.Size(), stat.ModTime().UnixNano())
}

func injectPrelude(htmlDoc string, prelude string) string {
	tag := "</head>"
	lower := strings.ToLower(htmlDoc)
	idx := strings.LastIndex(lower, tag)
	script := "\n<script>" + prelude + "\n</script>\n"
	if idx == -1 {
		return script + htmlDoc
	}
	return htmlDoc[:idx] + script + htmlDoc[idx:]
}

func setNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"\"", "%22",
		"#", "%23",
		"&", "%26",
		"+", "%2B",
		",", "%2C",
		"/", "%2F",
		":", "%3A",
		";", "%3B",
		"=", "%3D",
		"?", "%3F",
		"@", "%40",
	)
	return replacer.Replace(value)
}
