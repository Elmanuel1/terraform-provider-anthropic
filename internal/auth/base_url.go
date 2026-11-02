package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type baseURLOverride struct {
	inner Credentials
	base  *url.URL
}

// ParseBaseURL checks that raw is an absolute http or https URL with no query
// or fragment, such as https://api.anthropic.com or http://agentd:8080/prefix.
func ParseBaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSuffix(raw, "/"))
	if err != nil {
		return nil, err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%q is not an absolute http or https URL", raw)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%q has a query or fragment", raw)
	}
	return u, nil
}

// WithBaseURL wraps any Credentials so each request goes to base instead of
// BaseURL: the scheme and host are replaced and base's path is prefixed. A nil
// base leaves requests unchanged.
func WithBaseURL(creds Credentials, base *url.URL) Credentials {
	if base == nil || creds == nil {
		return creds
	}
	return baseURLOverride{inner: creds, base: base}
}

func (b baseURLOverride) Authenticate(ctx context.Context, req *http.Request) error {
	if err := b.inner.Authenticate(ctx, req); err != nil {
		return err
	}
	req.URL.Scheme = b.base.Scheme
	req.URL.Host = b.base.Host
	req.Host = b.base.Host
	req.URL.Path = b.base.Path + req.URL.Path
	if req.URL.RawPath != "" {
		req.URL.RawPath = b.base.EscapedPath() + req.URL.RawPath
	}
	return nil
}
