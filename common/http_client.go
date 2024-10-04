package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
)

// An HTTPClient manages communication with the runner API.
type HTTPClient struct {
	Client   *http.Client
	Endpoint string
}

// defaultClient is the default http.Client.
var defaultClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// New returns a new client.
func NewHttpClient(endpoint string, insecure bool) *HTTPClient {
	c := &HTTPClient{
		Endpoint: endpoint,
		Client:   defaultClient,
	}
	if insecure {
		c.Client = &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			},
		}
	}
	return c
}

func (p *HTTPClient) Retry(ctx context.Context, path, method string, in, out interface{}, headers map[string]string, b backoff.BackOffContext, ignoreStatusCode bool, retries int) (*http.Response, error) {
	retryCounter := 0
	for {
		res, err := p.DoJson(ctx, path, method, in, out, headers)
		retryCounter++
		if retryCounter > retries {
			return res, err
		}
		// do not retry on Canceled or DeadlineExceeded
		if ctxErr := ctx.Err(); ctxErr != nil {
			logrus.Errorf("http: context canceled")
			return res, ctxErr
		}

		duration := b.NextBackOff()

		if res != nil {
			// Check the response code. We retry on 500-range
			// responses to allow the server time to recover, as
			// 500's are typically not permanent errors and may
			// relate to outages on the server side.
			if (ignoreStatusCode && err != nil) || res.StatusCode > 501 {
				logrus.Errorf("url: %s server error: re-connect and re-try: %s", path, err)
				if duration == backoff.Stop {
					logrus.Errorf("max retry limit reached")
					return nil, err
				}
				time.Sleep(duration)
				continue
			}
		} else if err != nil {
			logrus.Errorf("http: request error: %s", err)
			if duration == backoff.Stop {
				logrus.Errorf("max retry limit reached")
				return nil, err
			}
			time.Sleep(duration)
			continue
		}
		return res, err
	}
}

func (p *HTTPClient) DoJson(ctx context.Context, path, method string, in, out interface{}, headers map[string]string) (*http.Response, error) {
	headers["Content-Type"] = "application/json"
	var buf = &bytes.Buffer{}
	// marshal the input payload into json format and copy
	// to an io.ReadCloser.
	if in != nil {
		if err := json.NewEncoder(buf).Encode(in); err != nil {
			logrus.Errorf("could not encode input payload: %s", err)
		}
	}
	res, body, err := p.do(ctx, path, method, headers, buf)
	if err != nil {
		return res, err
	}
	if nil == out {
		return res, nil
	}
	if jsonErr := json.Unmarshal(body, out); jsonErr != nil {
		return res, jsonErr
	}

	return res, nil
}

func (p *HTTPClient) CreateBackoff(ctx context.Context, maxElapsedTime time.Duration) backoff.BackOffContext {
	exp := backoff.NewExponentialBackOff()
	exp.MaxElapsedTime = maxElapsedTime
	return backoff.WithContext(exp, ctx)
}

// do is a helper function that posts a signed http request with
// the input encoded and response decoded from json.
func (p *HTTPClient) do(ctx context.Context, path, method string, headers map[string]string, in *bytes.Buffer) (*http.Response, []byte, error) {
	endpoint := p.Endpoint + path
	req, err := http.NewRequest(method, endpoint, in)
	if err != nil {
		return nil, nil, err
	}
	req = req.WithContext(ctx)

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	res, err := p.Client.Do(req)
	if res != nil {
		defer func() {
			// drain the response body so we can reuse
			// this connection.
			if _, err = io.Copy(io.Discard, io.LimitReader(res.Body, 4096)); err != nil {
				logrus.Errorf("could not drain response body: %s", err)
			}
			res.Body.Close()
		}()
	}
	if err != nil {
		return res, nil, err
	}

	// if the response body return no content we exit
	// immediately. We do not read or unmarshal the response
	// and we do not return an error.
	if res.StatusCode == 204 {
		return res, nil, nil
	}

	// else read the response body into a byte slice.
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res, nil, err
	}

	if res.StatusCode > 299 {
		// if the response body includes an error message
		// we should return the error string.
		if len(body) != 0 {
			return res, body, errors.New(
				string(body),
			)
		}
		// if the response body is empty we should return
		// the default status code text.
		return res, body, errors.New(
			http.StatusText(res.StatusCode),
		)
	}
	return res, body, nil
}
