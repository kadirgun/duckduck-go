package duckduckgo

import (
	"time"
)

// Option configures a DuckDuckGo client.
type Option func(*DuckDuckGo)

// WithTimeout sets a hard deadline for the entire request lifecycle.
// Default is 30 seconds. Use 0 to disable the timeout.
func WithTimeout(seconds int) Option {
	return func(d *DuckDuckGo) {
		d.timeout = time.Second * time.Duration(seconds)
	}
}

// WithProxy configures the client to use the specified proxy URL.
func WithProxy(proxyUrl string) Option {
	return func(d *DuckDuckGo) {
		d.proxy = proxyUrl
	}
}

// WithUserAgent overrides the default User-Agent header.
func WithUserAgent(ua string) Option {
	return func(d *DuckDuckGo) {
		d.userAgent = ua
	}
}

// WithRetryCount sets how many times to retry the initial search request on failure.
// Default is 0 (no retries).
func WithRetryCount(n int) Option {
	return func(d *DuckDuckGo) {
		d.retryCount = n
	}
}

// WithProxyFallback enables falling back to a direct connection if the proxy fails.
func WithProxyFallback() Option {
	return func(d *DuckDuckGo) {
		d.proxyFallback = true
	}
}
