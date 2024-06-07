package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	config ClientConfig
}

type Response interface {
	SetHeader(http.Header)
}

type httpHeader http.Header

func (h *httpHeader) SetHeader(header http.Header) {
	*h = httpHeader(header)
}

func (h *httpHeader) Header() http.Header {
	return http.Header(*h)
}

func (h *httpHeader) GetRateLimitHeaders() RateLimitHeaders {
	return newRateLimitHeaders(h.Header())
}

// NewClient create new Anthropic API client
func NewClient(apikey string, opts ...ClientOption) *Client {
	return &Client{
		config: newConfig(apikey, opts...),
	}
}

func (c *Client) sendRequest(req *http.Request, v Response) error {
	res, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	v.SetHeader(res.Header)

	if err := c.handlerRequestError(res); err != nil {
		return err
	}

	if err = json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}

	return nil
}

func (c *Client) handlerRequestError(resp *http.Response) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return &RequestError{
				StatusCode: resp.StatusCode,
				Err:        err,
			}
		}

		if c.IsVertexAI() && resp.StatusCode == 401 {
			var errRes VertexAIErrorResponse
			err := json.Unmarshal(bodyBytes, &errRes)
			if err != nil || errRes.Error == nil {
				reqErr := RequestError{
					StatusCode: resp.StatusCode,
					Err:        err,
					RawBody:    bodyBytes,
				}
				return &reqErr
			}
			return fmt.Errorf("error, status code: %d, message: %w", resp.StatusCode, errRes.Error)
		} else {
			var errRes ErrorResponse
			err := json.Unmarshal(bodyBytes, &errRes)
			if err != nil || errRes.Error == nil {
				reqErr := RequestError{
					StatusCode: resp.StatusCode,
					Err:        err,
					RawBody:    bodyBytes,
				}
				return &reqErr
			}

			return fmt.Errorf("error, status code: %d, message: %w", resp.StatusCode, errRes.Error)
		}
	}
	return nil
}

func (c *Client) fullURL(suffix string, model string) string {
	if isVertexAI(c.config.APIVersion) {
		// replace the first slash with a colon
		return fmt.Sprintf("%s/%s:%s", c.config.BaseURL, translateVertexModel(model), suffix[1:])
	} else {
		return fmt.Sprintf("%s%s", c.config.BaseURL, suffix)
	}
}

type requestSetter func(req *http.Request)

func withBetaVersion(version string) requestSetter {
	return func(req *http.Request) {
		req.Header.Set("anthropic-beta", version)
	}
}

func (c *Client) newRequest(ctx context.Context, method, urlSuffix string, body any, requestSetters ...requestSetter) (req *http.Request, err error) {
	// if the body implements the ModelGetter interface, use the model from the body
	model := ""
	if isVertexAI(c.config.APIVersion) && body != nil {
		if vertexAISupport, ok := body.(VertexAISupport); ok {
			model = vertexAISupport.GetModel()
			vertexAISupport.SetAnthropicVersion(c.config.APIVersion)
		} else {
			return nil, fmt.Errorf("this call not supported by the Vertex AI API")
		}
	}

	var reqBody []byte
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err = http.NewRequestWithContext(ctx, method, c.fullURL(urlSuffix, model), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	apiKey := c.config.apikey
	if c.config.apiKeyFunc != nil {
		apiKey = c.config.apiKeyFunc()
	}

	if isVertexAI(c.config.APIVersion) {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		req.Header.Set("X-Api-Key", apiKey)
		req.Header.Set("Anthropic-Version", c.config.APIVersion)
	}

	for _, setter := range requestSetters {
		setter(req)
	}

	return req, nil
}

func (c *Client) newStreamRequest(ctx context.Context, method, urlSuffix string, body any, requestSetters ...requestSetter) (req *http.Request,
	err error) {
	req, err = c.newRequest(ctx, method, urlSuffix, body, requestSetters...)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	return req, nil
}

func (c *Client) IsVertexAI() bool {
	return isVertexAI(c.config.APIVersion)
}
