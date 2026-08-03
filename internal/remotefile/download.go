// Package remotefile downloads remote files after applying network safety bounds.
package remotefile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Options configures a bounded remote download.
type Options struct {
	MaxBytes int64
	Timeout  time.Duration
}

// Result is a downloaded temporary file. Cleanup removes Path.
type Result struct {
	Path    string
	Cleanup func() error
}

// ErrorKind identifies a stable downloader failure category.
type ErrorKind string

const (
	ErrorInvalidInput   ErrorKind = "invalid_input"
	ErrorBlockedAddress ErrorKind = "blocked_address"
	ErrorTooLarge       ErrorKind = "too_large"
	ErrorHTTPStatus     ErrorKind = "http_status"
	ErrorTransport      ErrorKind = "transport"
	ErrorCancelled      ErrorKind = "cancelled"
)

// Error reports a categorized download failure. Callers should inspect it with errors.As.
type Error struct {
	Kind       ErrorKind
	StatusCode int
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return "remote download error"
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("remote download %s (status %d): %v", e.Kind, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("remote download %s: %v", e.Kind, e.Err)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error { return e.Err }

var (
	downloadLookupIP    = net.LookupIP
	downloadDialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	newDownloadTransport = func() *http.Transport {
		return http.DefaultTransport.(*http.Transport).Clone()
	}
)

// Download fetches rawURL to a 0600 temporary file, bounded by options.MaxBytes.
func Download(ctx context.Context, rawURL string, options Options) (Result, error) {
	if options.MaxBytes <= 0 {
		return Result{}, downloadError(ErrorInvalidInput, 0, errors.New("max bytes must be positive"))
	}
	if err := ctx.Err(); err != nil {
		return Result{}, contextDownloadError(ctx, err)
	}

	parsedURL, err := parseURL(rawURL)
	if err != nil {
		return Result{}, err
	}

	binder := &addressBinder{resolved: make(map[string][]net.IP)}
	if err := binder.prepare(parsedURL); err != nil {
		return Result{}, err
	}

	transport := newDownloadTransport()
	transport.Proxy = nil
	transport.DialContext = binder.dialContext
	transport.DisableKeepAlives = true // redirects must use their freshly validated binding.
	client := &http.Client{Transport: transport, Timeout: options.Timeout}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return downloadError(ErrorTransport, 0, errors.New("stopped after 5 redirects"))
		}
		return binder.prepare(req.URL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return Result{}, downloadError(ErrorInvalidInput, 0, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		var typed *Error
		if errors.As(err, &typed) {
			return Result{}, typed
		}
		return Result{}, contextDownloadError(ctx, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Result{}, downloadError(ErrorHTTPStatus, resp.StatusCode, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode))
	}
	if resp.ContentLength > options.MaxBytes {
		return Result{}, downloadError(ErrorTooLarge, 0, fmt.Errorf("content length %d exceeds limit %d", resp.ContentLength, options.MaxBytes))
	}

	ext := filepath.Ext(parsedURL.Path)
	if ext == "" {
		ext = ".jpg"
	}
	tmpFile, err := os.CreateTemp("", "md2wechat-download-*"+ext)
	if err != nil {
		return Result{}, downloadError(ErrorTransport, 0, fmt.Errorf("create temporary file: %w", err))
	}
	tmpPath := tmpFile.Name()
	cleanupOnError := func(err error) (Result, error) {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return Result{}, contextDownloadError(ctx, err)
	}

	body := io.Reader(resp.Body)
	if options.MaxBytes < math.MaxInt64 {
		body = io.LimitReader(resp.Body, options.MaxBytes+1)
	}
	written, err := io.Copy(tmpFile, body)
	if err != nil {
		return cleanupOnError(err)
	}
	if written > options.MaxBytes {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return Result{}, downloadError(ErrorTooLarge, 0, fmt.Errorf("response exceeds limit %d", options.MaxBytes))
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return Result{}, downloadError(ErrorTransport, 0, fmt.Errorf("close temporary file: %w", err))
	}

	return Result{Path: tmpPath, Cleanup: func() error { return os.Remove(tmpPath) }}, nil
}

// DownloadUnbounded is the legacy compatibility path used only by the WeChat wrapper.
func DownloadUnbounded(ctx context.Context, rawURL string, timeout time.Duration) (Result, error) {
	return Download(ctx, rawURL, Options{MaxBytes: math.MaxInt64, Timeout: timeout})
}

func parseURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL == nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		if err == nil {
			err = errors.New("URL must include a scheme and host")
		}
		return nil, downloadError(ErrorInvalidInput, 0, err)
	}
	if err := validateURL(parsedURL); err != nil {
		return nil, err
	}
	return parsedURL, nil
}

func validateURL(parsedURL *url.URL) error {
	if parsedURL == nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return downloadError(ErrorInvalidInput, 0, errors.New("URL must include a scheme and host"))
	}
	if parsedURL.User != nil {
		return downloadError(ErrorInvalidInput, 0, errors.New("URL credentials are not allowed"))
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return downloadError(ErrorInvalidInput, 0, fmt.Errorf("unsupported scheme %q", parsedURL.Scheme))
	}
	port := parsedURL.Port()
	if port != "" && port != "80" && port != "443" {
		return downloadError(ErrorInvalidInput, 0, fmt.Errorf("disallowed port %q", port))
	}
	return nil
}

type addressBinder struct {
	mu       sync.Mutex
	resolved map[string][]net.IP
}

func (b *addressBinder) prepare(parsedURL *url.URL) error {
	if err := validateURL(parsedURL); err != nil {
		return err
	}
	host := strings.ToLower(parsedURL.Hostname())
	port := parsedURL.Port()
	if port == "" {
		if parsedURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	ips, err := resolvePublicIPs(host)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.resolved[net.JoinHostPort(host, port)] = ips
	b.mu.Unlock()
	return nil
}

func (b *addressBinder) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	address = net.JoinHostPort(strings.ToLower(host), port)
	b.mu.Lock()
	ips := append([]net.IP(nil), b.resolved[address]...)
	b.mu.Unlock()
	if len(ips) == 0 {
		return nil, fmt.Errorf("remote address %q was not validated", address)
	}
	var lastErr error
	for _, ip := range ips {
		conn, err := downloadDialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func resolvePublicIPs(host string) ([]net.IP, error) {
	lowerHost := strings.ToLower(strings.TrimSpace(host))
	if lowerHost == "" || lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") {
		return nil, downloadError(ErrorBlockedAddress, 0, fmt.Errorf("host %q is not publicly routable", host))
	}
	if ip := net.ParseIP(lowerHost); ip != nil {
		if isBlockedIP(ip) {
			return nil, downloadError(ErrorBlockedAddress, 0, fmt.Errorf("IP %s is not publicly routable", ip))
		}
		return []net.IP{ip}, nil
	}
	ips, err := downloadLookupIP(host)
	if err != nil {
		return nil, downloadError(ErrorTransport, 0, fmt.Errorf("resolve %q: %w", host, err))
	}
	if len(ips) == 0 {
		return nil, downloadError(ErrorTransport, 0, fmt.Errorf("resolve %q: no addresses", host))
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, downloadError(ErrorBlockedAddress, 0, fmt.Errorf("IP %s is not publicly routable", ip))
		}
	}
	return ips, nil
}

func isBlockedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap()
	blocked := []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("198.18.0.0/15"),
		netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
		netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("::/128"), netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fc00::/7"), netip.MustParsePrefix("fe80::/10"), netip.MustParsePrefix("ff00::/8"), netip.MustParsePrefix("2001:db8::/32"),
	}
	for _, prefix := range blocked {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func downloadError(kind ErrorKind, status int, err error) *Error {
	return &Error{Kind: kind, StatusCode: status, Err: err}
}

func contextDownloadError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return downloadError(ErrorCancelled, 0, err)
	}
	return downloadError(ErrorTransport, 0, err)
}
