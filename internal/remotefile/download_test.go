package remotefile

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDownloadRejectsInvalidInputsBeforeNetwork(t *testing.T) {
	cases := []struct {
		name string
		url  string
		opts Options
	}{
		{"malformed URL", "://not-a-url", Options{MaxBytes: 10}},
		{"URL credentials", "https://user:pass@example.com/image.png", Options{MaxBytes: 10}},
		{"disallowed scheme", "ftp://example.com/image.png", Options{MaxBytes: 10}},
		{"disallowed port", "https://example.com:8080/image.png", Options{MaxBytes: 10}},
		{"non-positive limit", "https://example.com/image.png", Options{MaxBytes: 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Download(context.Background(), tc.url, tc.opts)
			assertDownloadError(t, err, ErrorInvalidInput, 0)
		})
	}
}

func TestDownloadBlocksPrivateInitialAndResolvedAddresses(t *testing.T) {
	cases := []struct {
		name string
		url  string
		ips  []net.IP
	}{
		{"literal private IP", "http://10.0.0.2/image.png", nil},
		{"resolved private IP", "https://internal.example/image.png", []net.IP{net.ParseIP("10.0.0.2")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withDownloadSeams(t, func(host string) ([]net.IP, error) {
				if tc.ips == nil {
					t.Fatalf("literal IP should not be resolved")
				}
				return tc.ips, nil
			}, nil, nil)
			_, err := Download(context.Background(), tc.url, Options{MaxBytes: 10})
			assertDownloadError(t, err, ErrorBlockedAddress, 0)
		})
	}
}

func TestDownloadUsesValidatedAddressInsteadOfSecondDNSLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "rebind.example" {
			t.Fatalf("Host = %q, want original hostname", r.Host)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	serverAddress := serverAddress(t, server.URL)
	lookups := 0
	var dialed string
	withDownloadSeams(t, func(host string) ([]net.IP, error) {
		lookups++
		if host != "rebind.example" {
			t.Fatalf("resolved host = %q", host)
		}
		if lookups == 1 {
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		}
		return []net.IP{net.ParseIP("10.0.0.2")}, nil
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed = address
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, nil)

	result, err := Download(context.Background(), "http://rebind.example/file.png", Options{MaxBytes: 10})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = result.Cleanup() }()
	if lookups != 1 {
		t.Fatalf("lookups = %d, want 1", lookups)
	}
	if dialed != "93.184.216.34:80" {
		t.Fatalf("dialed address = %q, want validated address", dialed)
	}
	assertFileContents(t, result.Path, "ok")
}

func TestDownloadFollowsPublicHTTPSRedirectsAndBlocksPrivateRedirects(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "https://example.com/final", http.StatusFound)
		case "/final":
			_, _ = w.Write([]byte("secure"))
		case "/private":
			http.Redirect(w, r, "https://private.example/final", http.StatusFound)
		case "/bad-scheme":
			http.Redirect(w, r, "ftp://example.com/final", http.StatusFound)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	serverAddress := serverAddress(t, server.URL)
	baseTransport := server.Client().Transport.(*http.Transport).Clone()
	withDownloadSeams(t, func(host string) ([]net.IP, error) {
		switch host {
		case "example.com":
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		case "private.example":
			return []net.IP{net.ParseIP("10.0.0.2")}, nil
		default:
			t.Fatalf("unexpected resolver host %q", host)
			return nil, nil
		}
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, func() *http.Transport { return baseTransport.Clone() })

	t.Run("public HTTPS redirect", func(t *testing.T) {
		result, err := Download(context.Background(), "https://example.com/start", Options{MaxBytes: 10})
		if err != nil {
			t.Fatalf("Download() error = %v", err)
		}
		defer func() { _ = result.Cleanup() }()
		assertFileContents(t, result.Path, "secure")
	})
	t.Run("private redirect", func(t *testing.T) {
		_, err := Download(context.Background(), "https://example.com/private", Options{MaxBytes: 10})
		assertDownloadError(t, err, ErrorBlockedAddress, 0)
	})
	t.Run("redirect validates scheme", func(t *testing.T) {
		_, err := Download(context.Background(), "https://example.com/bad-scheme", Options{MaxBytes: 10})
		assertDownloadError(t, err, ErrorInvalidInput, 0)
	})
}

func TestDownloadRejectsMoreThanFiveRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://redirect.example"+r.URL.Path+"?next=1", http.StatusFound)
	}))
	defer server.Close()
	serverAddress := serverAddress(t, server.URL)
	withDownloadSeams(t, publicResolver, func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, nil)

	_, err := Download(context.Background(), "http://redirect.example/start", Options{MaxBytes: 10})
	assertDownloadError(t, err, ErrorTransport, 0)
}

func TestDownloadEnforcesDeclaredAndChunkedSizeLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/declared":
			w.Header().Set("Content-Length", "11")
			_, _ = w.Write([]byte("eleven bytes"))
		case "/chunked":
			_, _ = io.WriteString(w, "eleven bytes")
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	serverAddress := serverAddress(t, server.URL)
	withDownloadSeams(t, publicResolver, func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, nil)

	for _, path := range []string{"declared", "chunked"} {
		t.Run(path, func(t *testing.T) {
			_, err := Download(context.Background(), "http://public.example/"+path, Options{MaxBytes: 10})
			assertDownloadError(t, err, ErrorTooLarge, 0)
		})
	}
}

func TestDownloadPreservesHTTPStatusAndCleansUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("ok"))
		case "/404":
			w.WriteHeader(http.StatusNotFound)
		case "/429":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/500":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	serverAddress := serverAddress(t, server.URL)
	withDownloadSeams(t, publicResolver, func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, nil)

	for _, tc := range []struct {
		path   string
		status int
	}{
		{"404", 404}, {"429", 429}, {"500", 500},
	} {
		t.Run(tc.path, func(t *testing.T) {
			_, err := Download(context.Background(), "http://public.example/"+tc.path, Options{MaxBytes: 10})
			assertDownloadError(t, err, ErrorHTTPStatus, tc.status)
		})
	}

	result, err := Download(context.Background(), "http://public.example/ok", Options{MaxBytes: 10})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	info, err := os.Stat(result.Path)
	if err != nil {
		t.Fatalf("stat downloaded file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("temporary file mode = %o, want 600", info.Mode().Perm())
	}
	if err := result.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(result.Path); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains after cleanup: %v", err)
	}

	legacyResult, err := DownloadUnbounded(context.Background(), "http://public.example/ok", 0)
	if err != nil {
		t.Fatalf("DownloadUnbounded() error = %v", err)
	}
	defer func() { _ = legacyResult.Cleanup() }()
	assertFileContents(t, legacyResult.Path, "ok")
}

func TestDownloadClassifiesCancellationAndPartialWriteFailure(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := Download(ctx, "https://public.example/file", Options{MaxBytes: 10})
		assertDownloadError(t, err, ErrorCancelled, 0)
	})

	t.Run("partial body write failure cleans temporary file", func(t *testing.T) {
		before := tempDownloadFiles(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "10")
			_, _ = w.Write([]byte("partial"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			panic("close connection after a partial body")
		}))
		defer server.Close()
		serverAddress := serverAddress(t, server.URL)
		withDownloadSeams(t, publicResolver, func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
		}, nil)
		_, err := Download(context.Background(), "http://public.example/partial", Options{MaxBytes: 20})
		assertDownloadError(t, err, ErrorTransport, 0)
		after := tempDownloadFiles(t)
		if strings.Join(before, "\n") != strings.Join(after, "\n") {
			t.Fatalf("temporary files leaked: before=%v after=%v", before, after)
		}
	})
}

func TestSyncDraftPackagesDoNotUseUnboundedDownload(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	syncDir := filepath.Join(filepath.Dir(thisFile), "..", "syncdraft")
	if _, err := os.Stat(syncDir); os.IsNotExist(err) {
		return
	}
	if err := verifySyncPackagesUseBoundedDownload(syncDir); err != nil {
		t.Fatal(err)
	}
}

func TestVerifySyncPackagesRejectsUnboundedDownload(t *testing.T) {
	syncDir := t.TempDir()
	for _, tc := range []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "bounded download is allowed",
			source: `package syncdraft
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
func download() { remotefile.Download(nil, "", remotefile.Options{MaxBytes: 1}) }`,
		},
		{
			name: "unbounded download is rejected",
			source: `package syncdraft
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
func download() { remotefile.DownloadUnbounded(nil, "", 0) }`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(syncDir, "adapter.go")
			if err := os.WriteFile(file, []byte(tc.source), 0600); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			err := verifySyncPackagesUseBoundedDownload(syncDir)
			if tc.wantErr && err == nil {
				t.Fatal("unbounded sync download was accepted")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("bounded sync download was rejected: %v", err)
			}
		})
	}
}

func assertDownloadError(t *testing.T, err error, wantKind ErrorKind, wantStatus int) {
	t.Helper()
	var downloadErr *Error
	if !errors.As(err, &downloadErr) {
		t.Fatalf("error = %T %v, want *Error", err, err)
	}
	if downloadErr.Kind != wantKind || downloadErr.StatusCode != wantStatus {
		t.Fatalf("error = kind %q status %d, want kind %q status %d", downloadErr.Kind, downloadErr.StatusCode, wantKind, wantStatus)
	}
}

func withDownloadSeams(t *testing.T, resolver func(string) ([]net.IP, error), dial func(context.Context, string, string) (net.Conn, error), transport func() *http.Transport) {
	t.Helper()
	oldResolver, oldDial, oldTransport := downloadLookupIP, downloadDialContext, newDownloadTransport
	if resolver != nil {
		downloadLookupIP = resolver
	}
	if dial != nil {
		downloadDialContext = dial
	}
	if transport != nil {
		newDownloadTransport = transport
	}
	t.Cleanup(func() {
		downloadLookupIP, downloadDialContext, newDownloadTransport = oldResolver, oldDial, oldTransport
	})
}

func publicResolver(host string) ([]net.IP, error) {
	if host != "public.example" && host != "redirect.example" {
		return nil, fmt.Errorf("unexpected resolver host %q", host)
	}
	return []net.IP{net.ParseIP("93.184.216.34")}, nil
}

func serverAddress(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	return parsed.Host
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("file contents = %q, want %q", got, want)
	}
}

func tempDownloadFiles(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(os.TempDir(), "md2wechat-download-*"))
	if err != nil {
		t.Fatalf("glob temporary files: %v", err)
	}
	return paths
}

func verifySyncPackagesUseBoundedDownload(syncDir string) error {
	packages, err := parser.ParseDir(token.NewFileSet(), syncDir, nil, 0)
	if err != nil {
		return fmt.Errorf("parse sync package: %w", err)
	}
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			aliases := make(map[string]bool)
			for _, spec := range file.Imports {
				if strings.Trim(spec.Path.Value, "\"") != "github.com/geekjourneyx/md2wechat-skill/internal/remotefile" {
					continue
				}
				alias := "remotefile"
				if spec.Name != nil {
					alias = spec.Name.Name
				}
				aliases[alias] = true
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "DownloadUnbounded" {
					return true
				}
				ident, ok := selector.X.(*ast.Ident)
				if ok && aliases[ident.Name] {
					err = fmt.Errorf("%s calls remotefile.DownloadUnbounded; sync packages must use bounded Download", file.Name.Name)
				}
				return true
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
