package httpx

import (
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/jfk9w-go/hikkabot/common/gox/fsx"
	Transport "github.com/jfk9w-go/hikkabot/common/httpx/transport"
	"github.com/jfk9w-go/hikkabot/common/logx"
)

// HTTP is an alias for *T.
type HTTP = *T

// Configure creates a HTTP instance from the Config.
func Configure(config *Config) HTTP {
	var (
		transport   *TransportConfig
		jar         http.CookieJar
		filetransit string
		err         error
	)

	if config != nil {
		transport = config.Transport

		if config.Cookies != nil {
			jar, err = cookiejar.New(nil)
			if err != nil {
				panic(err)
			}

			for ustr, config := range config.Cookies {
				cookies := make([]*http.Cookie, len(config))
				for i, c := range config {
					cookies[i] = c.toCookie()
				}

				u, err := url.Parse(ustr)
				if err != nil {
					panic(err)
				}

				jar.SetCookies(u, cookies)
			}
		}

		if config.TempStorage != "" {
			filetransit, err = fsx.Path(config.TempStorage)
			if err != nil {
				panic(err)
			}
		}
	}

	return &T{
		Client: &http.Client{
			Transport: ConfigureTransport(transport),
			Jar:       jar,
		},
		TempStorage: filetransit,
	}
}

// ConfigureTransport creates a Transport instance from the Config.
func ConfigureTransport(config *TransportConfig) http.RoundTripper {
	var (
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: time.Second,
			ResponseHeaderTimeout: time.Minute,
		}

		roundTripper http.RoundTripper = transport
	)

	if config != nil {
		if config.MaxIdleConns != nil {
			transport.MaxIdleConns = *config.MaxIdleConns
		}
		if config.IdleConnTimeout != nil {
			transport.IdleConnTimeout = config.IdleConnTimeout.Duration()
		}
		if config.TLSHandshakeTimeout != nil {
			transport.TLSHandshakeTimeout = config.TLSHandshakeTimeout.Duration()
		}
		if config.ExpectContinueTimeout != nil {
			transport.ExpectContinueTimeout = config.ExpectContinueTimeout.Duration()
		}
		if config.ResponseHeaderTimeout != nil {
			transport.ResponseHeaderTimeout = config.ResponseHeaderTimeout.Duration()
		}
		if config.Log != "" {
			roundTripper = &Transport.Logx{roundTripper, logx.Get(config.Log)}
		}
		if config.StatusCodes != nil {
			roundTripper = &Transport.StatusCodeChecker{roundTripper, config.StatusCodes}
		}
	}

	return roundTripper
}
