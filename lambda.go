package main

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Lambda handler function
func lambdaHandler(ctx context.Context, request events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	// Initialize store if not done
	if store.host.IP == "" {
		store.host.IP = getIPAddress()
		store.host.Name, _ = os.Hostname()
	}

	// Determine if multi-value headers mode is enabled
	multiValueMode := len(request.MultiValueHeaders) > 0

	// Create HTTP request from ALB request
	var bodyReader io.Reader
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			bodyReader = strings.NewReader(request.Body)
		} else {
			bodyReader = strings.NewReader(string(decoded))
		}
	} else {
		bodyReader = strings.NewReader(request.Body)
	}

	req := &http.Request{
		Method: request.HTTPMethod,
		URL: &url.URL{
			Path: request.Path,
		},
		Header: make(http.Header),
		Body:   io.NopCloser(bodyReader),
	}

	// Convert ALB headers to HTTP headers
	if multiValueMode {
		for key, values := range request.MultiValueHeaders {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	} else {
		for key, value := range request.Headers {
			req.Header.Set(key, value)
		}
	}

	// Parse query parameters
	// ALB passes query parameters without decoding, so we build RawQuery directly
	// to avoid double-encoding by url.Values.Encode().
	if multiValueMode && request.MultiValueQueryStringParameters != nil {
		var parts []string
		for key, vals := range request.MultiValueQueryStringParameters {
			for _, v := range vals {
				parts = append(parts, key+"="+v)
			}
		}
		req.URL.RawQuery = strings.Join(parts, "&")
	} else if request.QueryStringParameters != nil {
		var parts []string
		for key, value := range request.QueryStringParameters {
			parts = append(parts, key+"="+value)
		}
		req.URL.RawQuery = strings.Join(parts, "&")
	}

	// Create response recorder
	recorder := &ResponseRecorder{
		headers: make(http.Header),
		body:    strings.Builder{},
		status:  200,
	}

	// Process request using existing handler logic
	handleRequest(recorder, req)

	// Convert to ALB response
	response := events.ALBTargetGroupResponse{
		StatusCode: recorder.status,
		Body:       recorder.body.String(),
	}

	if multiValueMode {
		response.MultiValueHeaders = make(map[string][]string)
		for key, values := range recorder.headers {
			response.MultiValueHeaders[key] = values
		}
	} else {
		response.Headers = make(map[string]string)
		for key, values := range recorder.headers {
			if len(values) > 0 {
				response.Headers[key] = values[0]
			}
		}
	}

	return response, nil
}

// ResponseRecorder implements http.ResponseWriter for Lambda
type ResponseRecorder struct {
	headers http.Header
	body    strings.Builder
	status  int
}

func (r *ResponseRecorder) Header() http.Header {
	return r.headers
}

func (r *ResponseRecorder) Write(data []byte) (int, error) {
	return r.body.Write(data)
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

// handleRequest processes the request using existing gelbo logic
func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Save body content before logger.init consumes it
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// Set up logger in context (normally done by HandlerH2C + handlerWrapper)
	httpLogger := &HttpLogger{}
	ctx := context.WithValue(r.Context(), "logger", httpLogger)
	ctx = context.WithValue(ctx, "proto", "lambda")
	r = r.WithContext(ctx)

	httpLogger.init(r, 0)

	// Restore body so defaultHandler can read it for received_bytes
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// Route to appropriate handler based on path
	switch {
	case strings.HasPrefix(r.URL.Path, "/env/"):
		envHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/monitor/"):
		monitorHandler(w, r)
		return
	default:
		defaultHandler(w, r)
	}

	httpLogger.log()
}

// Main function for Lambda
func runLambda() {
	// Remove unsupported commands in Lambda environment
	delete(store.validatorForHttp, "cpu")
	delete(store.validatorForHttp, "memory")

	lambda.Start(lambdaHandler)
}
