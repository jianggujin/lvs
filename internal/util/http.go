package util

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

type HttpClientOption func(*http.Client)

func WithTimeout(timeout time.Duration) HttpClientOption {
	return func(c *http.Client) {
		c.Timeout = timeout
	}
}

func WithProxy(proxy *url.URL) HttpClientOption {
	return func(c *http.Client) {
		if proxy == nil {
			return
		}
		if c.Transport != nil {
			if value, ok := c.Transport.(*http.Transport); ok {
				value.Proxy = http.ProxyURL(proxy)
			}
			return
		}
		def := http.DefaultTransport.(*http.Transport)
		c.Transport = &http.Transport{
			Proxy:                 http.ProxyURL(proxy),
			DialContext:           def.DialContext,
			ForceAttemptHTTP2:     def.ForceAttemptHTTP2,
			MaxIdleConns:          def.MaxIdleConns,
			IdleConnTimeout:       def.IdleConnTimeout,
			TLSHandshakeTimeout:   def.TLSHandshakeTimeout,
			ExpectContinueTimeout: def.ExpectContinueTimeout,
		}
	}
}

func WithProxyStr(proxyStr string) HttpClientOption {
	if proxyStr == "" {
		return func(_ *http.Client) {
		}
	}

	proxy, _ := url.Parse(proxyStr)
	return WithProxy(proxy)
}

func NewHttpClient(opts ...HttpClientOption) *http.Client {
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
