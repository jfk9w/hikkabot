package httpx

import (
	"net/http"

	"github.com/jfk9w-go/hikkabot/common/gox/jsonx"
)

type (
	// TransportConfig allows to configure Transport via JSON.
	TransportConfig struct {

		// MaxIdleConns configures http.Transport. Default is 100.
		MaxIdleConns *int `json:"max_idle_conns"`

		// IdleConnTimeout configures http.Transport. Default is 90 seconds.
		IdleConnTimeout *jsonx.Duration `json:"idle_conn_timeout"`

		// TLSHandshakeTimeout configures http.Transport. Default is 10 seconds.
		TLSHandshakeTimeout *jsonx.Duration `json:"tls_handshake_timeout"`

		// ExpectContinueTimeout configures http.Transport. Default is 1 second.
		ExpectContinueTimeout *jsonx.Duration `json:"expect_continue_timeout"`

		// ResponseHeaderTimeout configures http.Transport's ResponseHeaderTimeout. Default is 1 minute.
		ResponseHeaderTimeout *jsonx.Duration `json:"response_header_timeout"`

		// Log is the transport logger name. If Log is not set, requests and responses will not be logged.
		Log string `json:"log"`

		// StatusCodes are valid status codes for the client. Default is [200].
		StatusCodes []int `json:"status_codes"`
	}

	// CookieConfig allows to configure http.CookieJar via JSON.
	CookieConfig struct {

		// Name configures http.Cookie.
		Name string `json:"name"`

		// Value configures http.Cookie.
		Value string `json:"value"`

		// Path configures http.Cookie.
		Path *string `json:"path"`

		// Domain configures http.Cookie.
		Domain *string `json:"domain"`
	}

	// Config encapsulates TransportConfig and CookieConfig.
	Config struct {

		// Transport is TransportConfig.
		Transport *TransportConfig `json:"transport"`

		// Cookies specifies CookieConfigs for domains.
		Cookies map[string][]CookieConfig `json:"cookies"`

		// TempStorage allows to specify a temporary directory for the client.
		// This directory will be used for storing automatically created received Files (i.e. Files with an empty Path).
		TempStorage string `json:"temp_storage"`

		// Headers are the default headers which will be set to each request.
		Headers map[string]string
	}
)

// WithStatusCodes copies the Config and sets StatusCodes to the corresponding value.
func (c *Config) WithStatusCodes(statusCodes ...int) *Config {
	if c == nil {
		c = new(Config)
	}

	if c.Transport == nil {
		c.Transport = new(TransportConfig)
	}

	c.Transport.StatusCodes = statusCodes

	return c
}

func (cc *CookieConfig) toCookie() *http.Cookie {
	cookie := &http.Cookie{
		Name:  cc.Name,
		Value: cc.Value,
	}

	if cc.Path != nil {
		cookie.Path = *cc.Path
	}

	if cc.Domain != nil {
		cookie.Domain = *cc.Domain
	}

	return cookie
}
