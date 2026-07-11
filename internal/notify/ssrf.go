package notify

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func ValidateHTTPSURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("only https URLs are allowed")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("url host is required")
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("url host is required")
	}
	if err := rejectBlockedHost(host); err != nil {
		return nil, err
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("dns lookup failed: %w", err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("url resolves to blocked address")
		}
	}
	return u, nil
}

func rejectBlockedHost(host string) error {
	lower := strings.ToLower(strings.Trim(host, "[]"))
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("localhost is not allowed")
	}
	if ip := net.ParseIP(lower); ip != nil && isBlockedIP(ip) {
		return fmt.Errorf("blocked ip address")
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	// AWS/GCP metadata
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	return false
}

func ValidateHTTPMethod(method string) error {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "POST", "PUT":
		return nil
	default:
		return fmt.Errorf("method %q is not allowed", method)
	}
}
