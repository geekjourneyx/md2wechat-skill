package remotefile

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

func TestDownloadAllowsHTTPSFakeIPAfterPublicDNSValidation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "images.example.com" {
			t.Fatalf("Host = %q, want original hostname", r.Host)
		}
		_, _ = w.Write([]byte("image"))
	}))
	defer server.Close()

	serverAddress := serverAddress(t, server.URL)
	baseTransport := server.Client().Transport.(*http.Transport).Clone()
	withPublicDNSResolver(t, func(ctx context.Context, host string) ([]net.IP, error) {
		if host != "images.example.com" {
			t.Fatalf("public DNS host = %q, want hostname only", host)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > publicDNSLookupTimeout {
			t.Fatalf("public DNS deadline = %v, want a live deadline within %s", deadline, publicDNSLookupTimeout)
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	})
	withDownloadSeams(t, func(host string) ([]net.IP, error) {
		if host != "images.example.com" {
			t.Fatalf("resolved host = %q", host)
		}
		return []net.IP{net.ParseIP("198.18.0.86")}, nil
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "93.184.216.34:443" {
			return nil, fmt.Errorf("dialed address = %q, want public DNS address", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, func() *http.Transport { return baseTransport.Clone() })

	result, err := Download(context.Background(), "https://images.example.com/file.png", Options{MaxBytes: 10})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = result.Cleanup() }()
	assertFileContents(t, result.Path, "image")
}

func TestDownloadFakeIPPublicDNSValidationFailsClosed(t *testing.T) {
	tests := []struct {
		name             string
		rawURL           string
		systemIPs        []net.IP
		publicIPs        []net.IP
		publicErr        error
		wantKind         ErrorKind
		wantPublicLookup bool
	}{
		{
			name:      "HTTP hostname",
			rawURL:    "http://images.example.com/file.png",
			systemIPs: []net.IP{net.ParseIP("198.18.0.86")},
			wantKind:  ErrorBlockedAddress,
		},
		{
			name:     "HTTPS IP literal",
			rawURL:   "https://198.18.0.86/file.png",
			wantKind: ErrorBlockedAddress,
		},
		{
			name:      "mixed fake and private system answers",
			rawURL:    "https://images.example.com/file.png",
			systemIPs: []net.IP{net.ParseIP("198.18.0.86"), net.ParseIP("10.0.0.2")},
			wantKind:  ErrorBlockedAddress,
		},
		{
			name:      "mixed fake and public system answers",
			rawURL:    "https://images.example.com/file.png",
			systemIPs: []net.IP{net.ParseIP("198.18.0.86"), net.ParseIP("93.184.216.34")},
			wantKind:  ErrorBlockedAddress,
		},
		{
			name:             "public DNS error",
			rawURL:           "https://images.example.com/file.png?token=not-for-dns",
			systemIPs:        []net.IP{net.ParseIP("198.18.0.86")},
			publicErr:        errors.New("public resolver unavailable"),
			wantKind:         ErrorTransport,
			wantPublicLookup: true,
		},
		{
			name:             "empty public DNS answer",
			rawURL:           "https://images.example.com/file.png",
			systemIPs:        []net.IP{net.ParseIP("198.18.0.86")},
			wantKind:         ErrorTransport,
			wantPublicLookup: true,
		},
		{
			name:             "private public DNS answer",
			rawURL:           "https://images.example.com/file.png",
			systemIPs:        []net.IP{net.ParseIP("198.18.0.86")},
			publicIPs:        []net.IP{net.ParseIP("10.0.0.2")},
			wantKind:         ErrorBlockedAddress,
			wantPublicLookup: true,
		},
		{
			name:             "fake public DNS answer",
			rawURL:           "https://images.example.com/file.png",
			systemIPs:        []net.IP{net.ParseIP("198.18.0.86")},
			publicIPs:        []net.IP{net.ParseIP("198.18.0.87")},
			wantKind:         ErrorBlockedAddress,
			wantPublicLookup: true,
		},
		{
			name:             "mixed public and special public DNS answers",
			rawURL:           "https://images.example.com/file.png",
			systemIPs:        []net.IP{net.ParseIP("198.18.0.86")},
			publicIPs:        []net.IP{net.ParseIP("93.184.216.34"), net.ParseIP("192.0.2.1")},
			wantKind:         ErrorBlockedAddress,
			wantPublicLookup: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			publicLookups := 0
			withPublicDNSResolver(t, func(_ context.Context, host string) ([]net.IP, error) {
				publicLookups++
				if host != "images.example.com" {
					t.Fatalf("public DNS host = %q, want hostname only", host)
				}
				return tc.publicIPs, tc.publicErr
			})
			withDownloadSeams(t, func(host string) ([]net.IP, error) {
				if tc.systemIPs == nil {
					t.Fatalf("IP literal %q should not use system DNS", host)
				}
				if host != "images.example.com" {
					t.Fatalf("system DNS host = %q, want hostname only", host)
				}
				return tc.systemIPs, nil
			}, func(context.Context, string, string) (net.Conn, error) {
				t.Fatal("blocked address must not be dialed")
				return nil, nil
			}, nil)

			_, err := Download(context.Background(), tc.rawURL, Options{MaxBytes: 10})
			assertDownloadError(t, err, tc.wantKind, 0)
			if got := publicLookups > 0; got != tc.wantPublicLookup {
				t.Fatalf("public DNS used = %t, want %t", got, tc.wantPublicLookup)
			}
			if strings.Contains(fmt.Sprint(err), "not-for-dns") {
				t.Fatalf("error leaked URL query: %v", err)
			}
		})
	}
}

func TestDownloadRevalidatesFakeIPOnEveryHTTPSRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "https://cdn.example.com/final", http.StatusFound)
		case "/final":
			_, _ = w.Write([]byte("redirected"))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	serverAddress := serverAddress(t, server.URL)
	baseTransport := server.Client().Transport.(*http.Transport).Clone()
	var publicLookups []string
	withPublicDNSResolver(t, func(_ context.Context, host string) ([]net.IP, error) {
		publicLookups = append(publicLookups, host)
		if host == "cdn.example.com" {
			return []net.IP{net.ParseIP("93.184.216.35")}, nil
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	})
	withDownloadSeams(t, func(host string) ([]net.IP, error) {
		switch host {
		case "images.example.com":
			return []net.IP{net.ParseIP("198.18.0.86")}, nil
		case "cdn.example.com":
			return []net.IP{net.ParseIP("198.18.0.87")}, nil
		default:
			t.Fatalf("unexpected system DNS host %q", host)
			return nil, nil
		}
	}, func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "93.184.216.34:443" && address != "93.184.216.35:443" {
			return nil, fmt.Errorf("dialed address = %q, want public DNS address", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, func() *http.Transport { return baseTransport.Clone() })

	result, err := Download(context.Background(), "https://images.example.com/start", Options{MaxBytes: 20})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = result.Cleanup() }()
	assertFileContents(t, result.Path, "redirected")
	if got, want := strings.Join(publicLookups, ","), "images.example.com,cdn.example.com"; got != want {
		t.Fatalf("public DNS lookups = %q, want %q", got, want)
	}
}

func TestPublicDNSLookupUsesLiteralDoHForAAndAAAA(t *testing.T) {
	var queryTypes []string
	transport := &closeTrackingRoundTripper{roundTrip: func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Scheme != "https" || req.URL.Host != "1.1.1.1" || req.URL.Path != "/dns-query" {
			t.Fatalf("DoH request = %s %s, want fixed literal HTTPS endpoint", req.Method, req.URL)
		}
		if req.Header.Get("Accept") != "application/dns-json" {
			t.Fatalf("Accept = %q, want application/dns-json", req.Header.Get("Accept"))
		}
		query := req.URL.Query()
		if len(query) != 2 || query.Get("name") != "images.example.com" {
			t.Fatalf("DoH query = %v, want hostname and type only", query)
		}
		queryType := query.Get("type")
		queryTypes = append(queryTypes, queryType)
		body := `{"Status":0,"Answer":[{"type":5,"data":"alias.example.com."},{"type":1,"data":"93.184.216.34"}]}`
		if queryType == "AAAA" {
			body = `{"Status":0,"Answer":[{"type":28,"data":"2606:2800:220:1:248:1893:25c8:1946"}]}`
		}
		return dohTestResponse(req, http.StatusOK, body), nil
	}}
	withPublicDNSTransport(t, transport)

	ips, err := lookupIPsUsingPublicDNS(context.Background(), "images.example.com")
	if err != nil {
		t.Fatalf("lookupIPsUsingPublicDNS() error = %v", err)
	}
	if got, want := strings.Join(queryTypes, ","), "A,AAAA"; got != want {
		t.Fatalf("query types = %q, want %q", got, want)
	}
	if got, want := joinIPs(ips), "93.184.216.34,2606:2800:220:1:248:1893:25c8:1946"; got != want {
		t.Fatalf("IPs = %q, want %q", got, want)
	}
	if transport.closeCalls != 1 {
		t.Fatalf("CloseIdleConnections() calls = %d, want 1", transport.closeCalls)
	}
}

func TestPublicDNSLookupFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{name: "transport error", err: errors.New("offline")},
		{name: "HTTP status", status: http.StatusBadGateway, body: `{}`},
		{name: "malformed JSON", status: http.StatusOK, body: `{`},
		{name: "DNS status", status: http.StatusOK, body: `{"Status":2}`},
		{name: "empty answer", status: http.StatusOK, body: `{"Status":0}`},
		{name: "invalid IP answer", status: http.StatusOK, body: `{"Status":0,"Answer":[{"type":1,"data":"not-an-ip"}]}`},
		{name: "oversized response", status: http.StatusOK, body: strings.Repeat("x", publicDNSDoHResponseLimit+1)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withPublicDNSTransport(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if tc.err != nil {
					return nil, tc.err
				}
				return dohTestResponse(req, tc.status, tc.body), nil
			}))
			if _, err := lookupIPsUsingPublicDNS(context.Background(), "images.example.com"); err == nil {
				t.Fatal("lookupIPsUsingPublicDNS() accepted an unverified DoH response")
			}
		})
	}
}

func TestPublicDNSLookupRejectsRedirect(t *testing.T) {
	requests := 0
	withPublicDNSTransport(t, roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		response := dohTestResponse(req, http.StatusFound, "")
		response.Header.Set("Location", "https://attacker.example/dns-query")
		return response, nil
	}))

	if _, err := lookupIPsUsingPublicDNS(context.Background(), "images.example.com"); err == nil {
		t.Fatal("lookupIPsUsingPublicDNS() followed a redirect")
	}
	if requests != 1 {
		t.Fatalf("DoH requests = %d, want 1", requests)
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

func TestDownloadAllowsFiveRedirects(t *testing.T) {
	redirects := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirects++
		if redirects <= 5 {
			http.Redirect(w, r, "http://redirect.example/next", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("after-five"))
	}))
	defer server.Close()
	serverAddress := serverAddress(t, server.URL)
	withDownloadSeams(t, publicResolver, func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, nil)

	result, err := Download(context.Background(), "http://redirect.example/start", Options{MaxBytes: 20})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = result.Cleanup() }()
	assertFileContents(t, result.Path, "after-five")
}

func TestDownloadEnforcesDeclaredAndChunkedSizeLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/declared":
			w.Header().Set("Content-Length", "11")
			_, _ = w.Write([]byte("eleven bytes"))
		case "/chunked":
			_, _ = io.WriteString(w, "ten bytes!")
			w.(http.Flusher).Flush()
			_, _ = io.WriteString(w, "!")
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()
	serverAddress := serverAddress(t, server.URL)
	withDownloadSeams(t, publicResolver, func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverAddress)
	}, nil)

	response, err := server.Client().Get(server.URL + "/chunked")
	if err != nil {
		t.Fatalf("read chunked response: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.ContentLength != -1 {
		t.Fatalf("chunked ContentLength = %d, want -1", response.ContentLength)
	}

	for _, path := range []string{"declared", "chunked"} {
		t.Run(path, func(t *testing.T) {
			before := tempDownloadFiles(t)
			_, err := Download(context.Background(), "http://public.example/"+path, Options{MaxBytes: 10})
			assertDownloadError(t, err, ErrorTooLarge, 0)
			after := tempDownloadFiles(t)
			if strings.Join(before, "\n") != strings.Join(after, "\n") {
				t.Fatalf("temporary files leaked: before=%v after=%v", before, after)
			}
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
		case "/201":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		case "/204":
			w.WriteHeader(http.StatusNoContent)
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

	boundedCreated, err := Download(context.Background(), "http://public.example/201", Options{MaxBytes: 10})
	if err != nil {
		t.Fatalf("bounded Download() 201 error = %v", err)
	}
	defer func() { _ = boundedCreated.Cleanup() }()
	assertFileContents(t, boundedCreated.Path, "created")

	legacyResult, err := DownloadUnbounded(context.Background(), "http://public.example/ok", 0)
	if err != nil {
		t.Fatalf("DownloadUnbounded() error = %v", err)
	}
	defer func() { _ = legacyResult.Cleanup() }()
	assertFileContents(t, legacyResult.Path, "ok")

	for _, tc := range []struct {
		path   string
		status int
	}{
		{"201", http.StatusCreated},
		{"204", http.StatusNoContent},
	} {
		t.Run("legacy requires 200/"+tc.path, func(t *testing.T) {
			_, err := DownloadUnbounded(context.Background(), "http://public.example/"+tc.path, 0)
			assertDownloadError(t, err, ErrorHTTPStatus, tc.status)
		})
	}
}

func TestResolvePublicIPsBlocksIANAAdditionalSpecialUseRanges(t *testing.T) {
	cases := []struct {
		name string
		ip   string
	}{
		{"IPv4 6to4 relay anycast", "192.88.99.1"},
		{"IPv4 AS112-v4", "192.31.196.1"},
		{"IPv4 AMT", "192.52.193.1"},
		{"IPv4 direct delegation", "192.175.48.1"},
		{"IPv6 discard-only", "100::1"},
		{"IPv6 locally assigned translation", "64:ff9b:1::1"},
		{"IPv6 benchmarking", "2001:2::1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withDownloadSeams(t, func(host string) ([]net.IP, error) {
				if host != "special.example" {
					t.Fatalf("resolved host = %q", host)
				}
				return []net.IP{net.ParseIP(tc.ip)}, nil
			}, nil, nil)
			_, err := resolvePublicIPs(context.Background(), "special.example", false)
			assertDownloadError(t, err, ErrorBlockedAddress, 0)
		})
	}
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

func TestRepositoryContractAllowsOnlyWechatUnboundedDownload(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	if err := verifyOnlyWechatUsesUnboundedDownload(repoRoot); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryContractRejectsUnboundedDownloadOutsideWechat(t *testing.T) {
	repoRoot := t.TempDir()
	writeRepositorySource(t, repoRoot, "internal/wechat/service.go", `package wechat
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
func download() { remotefile.DownloadUnbounded(nil, "", 0) }`)
	writeRepositorySource(t, repoRoot, "sync/bluesky.go", `package sync
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
func download() { remotefile.DownloadUnbounded(nil, "", 0) }`)

	if err := verifyOnlyWechatUsesUnboundedDownload(repoRoot); err == nil {
		t.Fatal("repository contract accepted an unbounded sync download")
	}
}

func TestRepositoryContractRejectsMultipleUnboundedDownloadsInWechatWrapper(t *testing.T) {
	repoRoot := t.TempDir()
	writeRepositorySource(t, repoRoot, "internal/wechat/service.go", `package wechat
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
func DownloadFile() { remotefile.DownloadUnbounded(nil, "", 0) }
func anotherCall() { remotefile.DownloadUnbounded(nil, "", 0) }`)

	if err := verifyOnlyWechatUsesUnboundedDownload(repoRoot); err == nil {
		t.Fatal("repository contract accepted multiple unbounded downloads in the WeChat wrapper")
	}
}

func TestRepositoryContractRejectsUnboundedDownloadOutsideDownloadFile(t *testing.T) {
	repoRoot := t.TempDir()
	writeRepositorySource(t, repoRoot, "internal/wechat/service.go", `package wechat
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
func anotherCall() { remotefile.DownloadUnbounded(nil, "", 0) }`)

	if err := verifyOnlyWechatUsesUnboundedDownload(repoRoot); err == nil {
		t.Fatal("repository contract accepted an unbounded download outside DownloadFile")
	}
}

func TestRepositoryContractRejectsUnboundedDownloadInVariableInitializer(t *testing.T) {
	repoRoot := t.TempDir()
	writeRepositorySource(t, repoRoot, "internal/wechat/service.go", `package wechat
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
var invoke = func() { remotefile.DownloadUnbounded(nil, "", 0) }
func DownloadFile() { remotefile.DownloadUnbounded(nil, "", 0) }`)

	if err := verifyOnlyWechatUsesUnboundedDownload(repoRoot); err == nil {
		t.Fatal("repository contract accepted an unbounded download in a variable initializer")
	}
}

func TestRepositoryContractRejectsUnboundedDownloadInDownloadFileAnonymousFunction(t *testing.T) {
	repoRoot := t.TempDir()
	writeRepositorySource(t, repoRoot, "internal/wechat/service.go", `package wechat
import "github.com/geekjourneyx/md2wechat-skill/internal/remotefile"
func DownloadFile() { func() { remotefile.DownloadUnbounded(nil, "", 0) }() }`)

	if err := verifyOnlyWechatUsesUnboundedDownload(repoRoot); err == nil {
		t.Fatal("repository contract accepted an unbounded download in a DownloadFile anonymous function")
	}
}

func writeRepositorySource(t *testing.T, repoRoot, relativePath, source string) {
	t.Helper()
	path := filepath.Join(repoRoot, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatalf("write fixture source: %v", err)
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

func withPublicDNSResolver(t *testing.T, resolver func(context.Context, string) ([]net.IP, error)) {
	t.Helper()
	oldResolver := downloadPublicLookupIP
	downloadPublicLookupIP = resolver
	t.Cleanup(func() { downloadPublicLookupIP = oldResolver })
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type closeTrackingRoundTripper struct {
	roundTrip  roundTripperFunc
	closeCalls int
}

func (t *closeTrackingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.roundTrip(req)
}

func (t *closeTrackingRoundTripper) CloseIdleConnections() { t.closeCalls++ }

func withPublicDNSTransport(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	oldFactory := newPublicDNSTransport
	newPublicDNSTransport = func() http.RoundTripper { return transport }
	t.Cleanup(func() { newPublicDNSTransport = oldFactory })
}

func dohTestResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func joinIPs(ips []net.IP) string {
	values := make([]string, 0, len(ips))
	for _, ip := range ips {
		values = append(values, ip.String())
	}
	return strings.Join(values, ",")
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

func verifyOnlyWechatUsesUnboundedDownload(repoRoot string) error {
	allowedFile := filepath.Join(repoRoot, "internal", "wechat", "service.go")
	callers := 0

	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		aliases := make(map[string]bool)
		dotImport := false
		for _, spec := range file.Imports {
			if strings.Trim(spec.Path.Value, "\"") != "github.com/geekjourneyx/md2wechat-skill/internal/remotefile" {
				continue
			}
			alias := "remotefile"
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			if alias == "." {
				dotImport = true
			} else {
				aliases[alias] = true
			}
		}

		calls, outsideDownloadFile := unboundedDownloadCalls(file, aliases, dotImport)
		if calls == 0 {
			return nil
		}
		if filepath.Clean(path) != allowedFile || outsideDownloadFile {
			return fmt.Errorf("%s uses remotefile.DownloadUnbounded; only internal/wechat/service.go may use it", path)
		}
		callers += calls
		return nil
	})
	if err != nil {
		return err
	}
	if callers != 1 {
		return fmt.Errorf("remotefile.DownloadUnbounded callers = %d, want exactly 1 in internal/wechat/service.go", callers)
	}
	return nil
}

func unboundedDownloadCalls(file *ast.File, aliases map[string]bool, dotImport bool) (int, bool) {
	var downloadFileBody *ast.BlockStmt
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "DownloadFile" {
			downloadFileBody = function.Body
			break
		}
	}

	calls := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isUnboundedDownloadCall(call, aliases, dotImport) {
			return true
		}
		calls++
		return true
	})

	directDownloadFileCalls := 0
	if downloadFileBody != nil {
		ast.Inspect(downloadFileBody, func(node ast.Node) bool {
			if _, ok := node.(*ast.FuncLit); ok {
				return false
			}
			call, ok := node.(*ast.CallExpr)
			if ok && isUnboundedDownloadCall(call, aliases, dotImport) {
				directDownloadFileCalls++
			}
			return true
		})
	}
	return calls, calls != directDownloadFileCalls
}

func isUnboundedDownloadCall(call *ast.CallExpr, aliases map[string]bool, dotImport bool) bool {
	switch function := call.Fun.(type) {
	case *ast.SelectorExpr:
		identifier, ok := function.X.(*ast.Ident)
		return ok && function.Sel.Name == "DownloadUnbounded" && aliases[identifier.Name]
	case *ast.Ident:
		return dotImport && function.Name == "DownloadUnbounded"
	default:
		return false
	}
}
