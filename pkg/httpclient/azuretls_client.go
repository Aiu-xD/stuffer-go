package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	azuretls "github.com/Noooste/azuretls-client"
	"universal-checker/pkg/types"
)

// AzureTLSClient wraps the azuretls-client to provide a standard HTTP client interface
type AzureTLSClient struct {
	client  *azuretls.Session
	proxy   *types.Proxy
	timeout time.Duration
}

// NewAzureTLSClient creates a new AzureTLS client with optional proxy support
func NewAzureTLSClient(proxy *types.Proxy, timeout time.Duration) (*AzureTLSClient, error) {
	session := azuretls.NewSession()
	
	// Apply Chrome browser fingerprint for better compatibility
	err := session.ApplyJa3("771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0", "chrome")
	if err != nil {
		// If JA3 fails, continue without it
		session.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	
	// Configure proxy if provided
	if proxy != nil {
		proxyURL := fmt.Sprintf("%s://%s:%d", string(proxy.Type), proxy.Host, proxy.Port)
		if err := session.SetProxy(proxyURL); err != nil {
			return nil, fmt.Errorf("failed to set proxy: %v", err)
		}
	}
	
	// Set timeout
	if timeout > 0 {
		session.SetTimeout(timeout)
	} else {
		session.SetTimeout(30 * time.Second)
	}
	
	// Disable certificate verification for compatibility
	session.InsecureSkipVerify = true
	
	return &AzureTLSClient{
		client:  session,
		proxy:   proxy,
		timeout: timeout,
	}, nil
}

// Do executes an HTTP request using azuretls-client
func (c *AzureTLSClient) Do(req *http.Request) (*http.Response, error) {
	// Handle context timeout
	if req.Context() != nil {
		if deadline, ok := req.Context().Deadline(); ok {
			timeout := time.Until(deadline)
			if timeout > 0 {
				c.client.SetTimeout(timeout)
			}
		}
	}
	
	// Set headers on session
	c.client.OrderedHeaders = azuretls.OrderedHeaders{}
	for name, values := range req.Header {
		for _, value := range values {
			c.client.OrderedHeaders = append(c.client.OrderedHeaders, []string{name, value})
		}
	}
	
	var resp *azuretls.Response
	var err error
	
	// Handle different HTTP methods
	switch req.Method {
	case "GET":
		resp, err = c.client.Get(req.URL.String())
	case "POST":
		var body interface{}
		if req.Body != nil {
			bodyBytes, readErr := io.ReadAll(req.Body)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read request body: %v", readErr)
			}
			body = bodyBytes
			req.Body.Close()
		}
		resp, err = c.client.Post(req.URL.String(), body)
	case "PUT":
		var body interface{}
		if req.Body != nil {
			bodyBytes, readErr := io.ReadAll(req.Body)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read request body: %v", readErr)
			}
			body = bodyBytes
			req.Body.Close()
		}
		resp, err = c.client.Put(req.URL.String(), body)
	case "DELETE":
		resp, err = c.client.Delete(req.URL.String())
	case "HEAD":
		resp, err = c.client.Head(req.URL.String())
	case "OPTIONS":
		resp, err = c.client.Options(req.URL.String())
	case "PATCH":
		var body interface{}
		if req.Body != nil {
			bodyBytes, readErr := io.ReadAll(req.Body)
			if readErr != nil {
				return nil, fmt.Errorf("failed to read request body: %v", readErr)
			}
			body = bodyBytes
			req.Body.Close()
		}
		resp, err = c.client.Patch(req.URL.String(), body)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", req.Method)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Convert azuretls response to http.Response
	httpResp := &http.Response{
		Status:        fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
		StatusCode:    resp.StatusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(string(resp.Body))),
		ContentLength: int64(len(resp.Body)),
		Request:       req,
	}
	
	// Convert headers from fhttp.Header to http.Header
	for name, values := range resp.Header {
		for _, value := range values {
			httpResp.Header.Add(name, value)
		}
	}
	
	return httpResp, nil
}

// Get performs a GET request
func (c *AzureTLSClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post performs a POST request
func (c *AzureTLSClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(req)
}

// PostForm performs a POST request with form data
func (c *AzureTLSClient) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// SetProxy updates the proxy configuration
func (c *AzureTLSClient) SetProxy(proxy *types.Proxy) error {
	c.proxy = proxy
	if proxy != nil {
		proxyURL := fmt.Sprintf("%s://%s:%d", string(proxy.Type), proxy.Host, proxy.Port)
		return c.client.SetProxy(proxyURL)
	}
	return c.client.SetProxy("")
}

// SetTimeout updates the timeout configuration
func (c *AzureTLSClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.client.SetTimeout(timeout)
}

// Close closes the client and cleans up resources
func (c *AzureTLSClient) Close() error {
	// azuretls-client doesn't have an explicit close method
	// but we can clear the session
	c.client = nil
	return nil
}

// GetProxy returns the current proxy configuration
func (c *AzureTLSClient) GetProxy() *types.Proxy {
	return c.proxy
}

// HTTPClientInterface defines the interface that both standard http.Client and AzureTLSClient implement
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
	Post(url, contentType string, body io.Reader) (*http.Response, error)
	PostForm(url string, data url.Values) (*http.Response, error)
}

// Ensure AzureTLSClient implements HTTPClientInterface
var _ HTTPClientInterface = (*AzureTLSClient)(nil)
