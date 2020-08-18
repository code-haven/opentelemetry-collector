// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package confighttp

import (
	"compress/gzip"
	"compress/zlib"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/rs/cors"

	"go.opentelemetry.io/collector/config/configtls"
)

type HTTPClientSettings struct {
	// The target URL to send data to (e.g.: http://some.url:9411/v1/trace).
	Endpoint string `mapstructure:"endpoint"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `mapstructure:",squash"`

	// ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
	ReadBufferSize int `mapstructure:"read_buffer_size"`

	// WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
	WriteBufferSize int `mapstructure:"write_buffer_size"`

	// Timeout parameter configures `http.Client.Timeout`.
	Timeout time.Duration `mapstructure:"timeout,omitempty"`

	// Additional headers attached to each HTTP request sent by the client.
	// Existing header values are overwritten if collision happens.
	Headers map[string]string `mapstructure:"headers,omitempty"`
}

func (hcs *HTTPClientSettings) ToClient() (*http.Client, error) {
	tlsCfg, err := hcs.TLSSetting.LoadTLSConfig()
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	if hcs.ReadBufferSize > 0 {
		transport.ReadBufferSize = hcs.ReadBufferSize
	}
	if hcs.WriteBufferSize > 0 {
		transport.WriteBufferSize = hcs.WriteBufferSize
	}
	var clientTransport http.RoundTripper

	if hcs.Headers != nil && len(hcs.Headers) > 0 {
		clientTransport = &clientInterceptorRoundTripper{
			transport: transport,
			headers:   hcs.Headers,
		}
	} else {
		clientTransport = transport
	}

	return &http.Client{
		Transport: clientTransport,
		Timeout:   hcs.Timeout,
	}, nil
}

// Custom RoundTripper that add headers
type clientInterceptorRoundTripper struct {
	transport http.RoundTripper
	headers   map[string]string
}

// Custom RoundTrip that add headers
func (interceptor *clientInterceptorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range interceptor.headers {
		req.Header.Set(k, v)
	}
	// Send the request to Cortex
	response, err := interceptor.transport.RoundTrip(req)

	return response, err
}

type HTTPServerSettings struct {
	// Endpoint configures the listening address for the server.
	Endpoint string `mapstructure:"endpoint"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *configtls.TLSServerSetting `mapstructure:"tls_settings, omitempty"`

	// CorsOrigins are the allowed CORS origins for HTTP/JSON requests to grpc-gateway adapter
	// for the OTLP receiver. See github.com/rs/cors
	// An empty list means that CORS is not enabled at all. A wildcard (*) can be
	// used to match any origin or one or more characters of an origin.
	CorsOrigins []string `mapstructure:"cors_allowed_origins"`

	// EnableDecompression if true, a middleware is configured that decompresses the body
	// of incoming HTTP requests based on the compression format in the Content-Encoding header.
	// Currently, it has support for gzip and deflate/zlib.
	EnableDecompression bool `mapstructure:"enable_decompression"`
}

func (hss *HTTPServerSettings) ToListener() (net.Listener, error) {
	listener, err := net.Listen("tcp", hss.Endpoint)
	if err != nil {
		return nil, err
	}

	if hss.TLSSetting != nil {
		var tlsCfg *tls.Config
		tlsCfg, err = hss.TLSSetting.LoadTLSConfig()
		if err != nil {
			return nil, err
		}
		listener = tls.NewListener(listener, tlsCfg)
	}
	return listener, nil
}

func (hss *HTTPServerSettings) ToServer(handler http.Handler) *http.Server {
	if len(hss.CorsOrigins) > 0 {
		co := cors.Options{AllowedOrigins: hss.CorsOrigins}
		handler = cors.New(co).Handler(handler)
	}
	if hss.EnableDecompression {
		handler = httpDecompressionHandler(handler)
	}
	return &http.Server{
		Handler: handler,
	}
}

// httpDecompressionHandler is a middleware that helps offload the task of handling compressed
// HTTP requests by identifying the compression format in the "Content-Encoding" header and re-writing
// request body so that the handlers further in the chain can work on decompressed data.
// It supports gzip and deflate/zlib compression.
func httpDecompressionHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				// TODO: OTLP expects error of type google.rpc.Status
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer gr.Close()
			// "Content-Encoding" header is removed to avoid decompressing twice
			// in case the next handler(s) have implemented a similar mechanism.
			r.Header.Del("Content-Encoding")
			r.Body = gr
		case "deflate", "zlib":
			zr, err := zlib.NewReader(r.Body)
			if err != nil {
				// TODO: OTLP expects error of type google.rpc.Status
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer zr.Close()
			r.Header.Del("Content-Encoding")
			r.Body = zr
		}
		handler.ServeHTTP(w, r)
	})
}
