package mihomo

import (
	"net/http"
	"time"
)

type clientOptions struct {
	secret     string
	httpClient *http.Client
	timeout    time.Duration
	userAgent  string
}

// Option configures a Mihomo Client.
type Option func(*clientOptions)

// WithSecret sets the authentication secret token for Mihomo External Controller.
func WithSecret(secret string) Option {
	return func(o *clientOptions) {
		o.secret = secret
	}
}

// WithHTTPClient specifies a custom http.Client for REST requests.
func WithHTTPClient(client *http.Client) Option {
	return func(o *clientOptions) {
		if client != nil {
			o.httpClient = client
		}
	}
}

// WithTimeout sets the default timeout for HTTP requests.
func WithTimeout(timeout time.Duration) Option {
	return func(o *clientOptions) {
		if timeout > 0 {
			o.timeout = timeout
		}
	}
}

// WithUserAgent customizes the User-Agent header sent to the controller.
func WithUserAgent(ua string) Option {
	return func(o *clientOptions) {
		o.userAgent = ua
	}
}
