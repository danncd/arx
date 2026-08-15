package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"
	"unicode/utf8"
)

var fetchClient = newFetchClient()

func newFetchClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeDial
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return validateFetchURL(req.URL)
		},
	}
}

func validateFetchURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must use http or https")
	}
	if u.Hostname() == "" {
		return fmt.Errorf("url must include a host")
	}
	return nil
}

func safeDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	var addrs []net.IPAddr
	if ip := net.ParseIP(host); ip != nil {
		addrs = []net.IPAddr{{IP: ip}}
	} else {
		addrs, err = net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("host %q has no addresses", host)
	}
	for _, addr := range addrs {
		if !publicIP(addr.IP) {
			return nil, fmt.Errorf("host %q resolves to a private or local address", host)
		}
	}

	var dialErr error
	for _, addr := range addrs {
		if network == "tcp4" && addr.IP.To4() == nil || network == "tcp6" && addr.IP.To4() != nil {
			continue
		}
		conn, err := (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(addr.IP.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = err
	}
	if dialErr == nil {
		dialErr = fmt.Errorf("host %q has no address for %s", host, network)
	}
	return nil, dialErr
}

func publicIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return false
	}
	for _, prefix := range specialNetworks {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

var specialNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("fec0::/10"),
}

var Fetch = Tool{
	Name:        "fetch_url",
	Description: "Fetch an http(s) URL and return the response body as text.",
	Schema: json.RawMessage(`{"type":"object","properties":{` +
		`"url":{"type":"string","description":"Full http or https URL"}},` +
		`"required":["url"]}`),
	Run: func(ctx context.Context, args json.RawMessage) (string, error) {
		var a struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("bad arguments: %w", err)
		}
		u, err := url.Parse(a.URL)
		if err != nil {
			return "", err
		}
		if err := validateFetchURL(u); err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, "GET", a.URL, nil)
		if err != nil {
			return "", err
		}
		resp, err := fetchClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("HTTP %s", resp.Status)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxToolBytes+1))
		if err != nil {
			return "", fmt.Errorf("read body: %w", err)
		}
		out, truncated := clipUTF8(data, maxToolBytes)
		if !utf8.ValidString(out) {
			return "", fmt.Errorf("response is not UTF-8 text")
		}
		if truncated {
			out += "\n[truncated at 64KB]"
		}
		return out, nil
	},
}
