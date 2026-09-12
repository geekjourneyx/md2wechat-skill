// Package remotefile downloads remote files after applying network safety bounds.
package remotefile

import (
	"context"
	"encoding/json"
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
	downloadLookupIP       = net.LookupIP
	downloadPublicLookupIP = lookupIPsUsingPublicDNS
	downloadDialContext    = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	newDownloadTransport = func() *http.Transport {
		return http.DefaultTransport.(*http.Transport).Clone()
	}
	newPublicDNSTransport = func() http.RoundTripper {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		return transport
	}
)

// Download fetches rawURL to a 0600 temporary file, bounded by options.MaxBytes.
func Download(ctx context.Context, rawURL string, options Options) (Result, error) {
	return download(ctx, rawURL, options, false)
}

func download(ctx context.Context, rawURL string, options Options, requireStatusOK bool) (Result, error) {
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
	if err := binder.prepare(ctx, parsedURL); err != nil {
		return Result{}, err
	}

	transport := newDownloadTransport()
	transport.Proxy = nil
	transport.DialContext = binder.dialContext
	transport.DisableKeepAlives = true // redirects must use their freshly validated binding.
	client := &http.Client{Transport: transport, Timeout: options.Timeout}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 {
			return downloadError(ErrorTransport, 0, errors.New("stopped after 5 redirects"))
		}
		return binder.prepare(req.Context(), req.URL)
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

	if (requireStatusOK && resp.StatusCode != http.StatusOK) || (!requireStatusOK && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices)) {
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
	return download(ctx, rawURL, Options{MaxBytes: math.MaxInt64, Timeout: timeout}, true)
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

func (b *addressBinder) prepare(ctx context.Context, parsedURL *url.URL) error {
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
	ips, err := resolvePublicIPs(ctx, host, parsedURL.Scheme == "https")
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

func resolvePublicIPs(ctx context.Context, host string, allowFakeIPFallback bool) ([]net.IP, error) {
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
	if allowFakeIPFallback && allIPsMatchPrefix(ips, clashFakeIPPrefix) {
		lookupCtx, cancel := context.WithTimeout(ctx, publicDNSLookupTimeout)
		defer cancel()
		publicIPs, err := downloadPublicLookupIP(lookupCtx, lowerHost)
		if err != nil {
			return nil, contextDownloadError(ctx, fmt.Errorf("verify public address for %q: %w", host, err))
		}
		if len(publicIPs) == 0 {
			return nil, downloadError(ErrorTransport, 0, fmt.Errorf("verify public address for %q: no addresses", host))
		}
		for _, ip := range publicIPs {
			if isBlockedIP(ip) {
				return nil, downloadError(ErrorBlockedAddress, 0, fmt.Errorf("IP %s is not publicly routable", ip))
			}
		}
		return publicIPs, nil
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, downloadError(ErrorBlockedAddress, 0, fmt.Errorf("IP %s is not publicly routable", ip))
		}
	}
	return ips, nil
}

const (
	publicDNSDoHEndpoint      = "https://1.1.1.1/dns-query"
	publicDNSLookupTimeout    = 5 * time.Second
	publicDNSDoHResponseLimit = 64 << 10
)

var clashFakeIPPrefix = netip.MustParsePrefix("198.18.0.0/15")

func lookupIPsUsingPublicDNS(ctx context.Context, host string) ([]net.IP, error) {
	client := &http.Client{
		Transport: newPublicDNSTransport(),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("public DNS redirects are not allowed")
		},
	}
	defer client.CloseIdleConnections()
	var ips []net.IP
	for _, recordType := range []string{"A", "AAAA"} {
		endpoint, err := url.Parse(publicDNSDoHEndpoint)
		if err != nil {
			return nil, err
		}
		query := endpoint.Query()
		query.Set("name", host)
		query.Set("type", recordType)
		endpoint.RawQuery = query.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/dns-json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := readPublicDNSResponse(resp)
		if err != nil {
			return nil, err
		}
		var payload publicDNSResponse
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("decode public DNS response: %w", err)
		}
		if payload.Status != 0 {
			return nil, fmt.Errorf("public DNS status %d", payload.Status)
		}
		for _, answer := range payload.Answer {
			if (recordType == "A" && answer.Type != 1) || (recordType == "AAAA" && answer.Type != 28) {
				continue
			}
			ip := net.ParseIP(strings.TrimSpace(answer.Data))
			if ip == nil || (answer.Type == 1 && ip.To4() == nil) || (answer.Type == 28 && ip.To4() != nil) {
				return nil, fmt.Errorf("public DNS returned an invalid IP answer")
			}
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("public DNS returned no IP addresses")
	}
	return ips, nil
}

type publicDNSResponse struct {
	Status int               `json:"Status"`
	Answer []publicDNSAnswer `json:"Answer"`
}

type publicDNSAnswer struct {
	Type int    `json:"type"`
	Data string `json:"data"`
}

func readPublicDNSResponse(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, errors.New("public DNS returned an empty response")
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, publicDNSDoHResponseLimit+1))
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read public DNS response: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close public DNS response: %w", closeErr)
	}
	if len(body) > publicDNSDoHResponseLimit {
		return nil, errors.New("public DNS response exceeds limit")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("public DNS HTTP status %d", resp.StatusCode)
	}
	return body, nil
}

func allIPsMatchPrefix(ips []net.IP, prefix netip.Prefix) bool {
	if len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok || !prefix.Contains(addr.Unmap()) {
			return false
		}
	}
	return true
}

func isBlockedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap()
	for _, prefix := range specialUsePrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

var specialUsePrefixes = []netip.Prefix{
	// IANA IPv4 Special-Purpose Address Registry.
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.31.196.0/24"),
	netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("192.175.48.0/24"),
	clashFakeIPPrefix,
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	// IANA IPv6 Special-Purpose Address Registry.
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:3::/32"),
	netip.MustParsePrefix("2001:4:112::/48"),
	netip.MustParsePrefix("2001:10::/28"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:30::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2620:4f:8000::/48"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("ff00::/8"),
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
