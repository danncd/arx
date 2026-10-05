package web

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxPage = 4 << 20

type Client struct {
	http        *http.Client
	searchURL   string
	duckURL     string
	searchMutex sync.Mutex
	searchCache map[string]*searchEntry
}

func New() *Client {
	transport := &http.Transport{DialContext: publicDial, ForceAttemptHTTP2: true, ResponseHeaderTimeout: 10 * time.Second, TLSHandshakeTimeout: 10 * time.Second}
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second, CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("Too many redirects")
		}
		return validateURL(request.URL)
	}}
	return &Client{http: client, searchURL: "https://www.bing.com/search", duckURL: "https://html.duckduckgo.com/html/"}
}

func validateURL(address *url.URL) error {
	if (address.Scheme != "https" && address.Scheme != "http") || address.Hostname() == "" || address.User != nil {
		return errors.New("Use a public HTTP or HTTPS URL without credentials")
	}
	return nil
}

func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified() && !net.IPv4(100, 64, 0, 0).Equal(ip.Mask(net.CIDRMask(10, 32)))
}

func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("Host has no address")
	}
	for _, entry := range ips {
		if !publicIP(entry.IP) {
			return nil, errors.New("Web requests cannot access private or local network addresses")
		}
	}
	dialer := net.Dialer{Timeout: 8 * time.Second}
	var last error
	for _, entry := range ips {
		connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(entry.IP.String(), port))
		if err == nil {
			return connection, nil
		}
		last = err
	}
	return nil, last
}

func (c *Client) get(ctx context.Context, address string) ([]byte, string, string, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, "", "", errors.New("Invalid URL")
	}
	if err := validateURL(parsed); err != nil {
		return nil, "", "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", "", err
	}
	request.Header.Set("User-Agent", "Arx/0.1")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml,text/plain,application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", "", errors.New("Web request failed: " + response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxPage+1))
	if err != nil {
		return nil, "", "", err
	}
	if len(body) > maxPage {
		return nil, "", "", errors.New("Page exceeds the 4 MiB limit")
	}
	kind := strings.ToLower(response.Header.Get("Content-Type"))
	if kind == "" {
		kind = http.DetectContentType(body)
	}
	return body, response.Request.URL.String(), kind, nil
}
