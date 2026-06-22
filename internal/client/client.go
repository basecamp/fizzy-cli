// Package client provides an HTTP client for the Fizzy API.
package client

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

// Client is an HTTP client for the Fizzy API.
type Client struct {
	BaseURL    string
	Token      string
	Account    string
	HTTPClient *http.Client
	Verbose    bool
	// Sleeper is called for retry delays. Defaults to time.Sleep.
	// Override in tests with a no-op or recording function.
	Sleeper func(time.Duration)
}

// APIResponse represents a response from the API.
type APIResponse struct {
	StatusCode int
	Body       []byte
	Location   string
	LinkNext   string
	Data       any
}

// New creates a new API client.
func New(baseURL, token, account string) *Client {
	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Token:   token,
		Account: account,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// buildURL constructs the full API URL.
func (c *Client) buildURL(path string) string {
	// If path already starts with http, use as-is
	if strings.HasPrefix(path, "http") {
		return path
	}
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// Insert account into path if not present
	if c.Account != "" {
		accountPrefix := "/" + c.Account + "/"
		if !strings.HasPrefix(path, accountPrefix) && path != "/"+c.Account {
			path = "/" + c.Account + path
		}
	}
	return c.BaseURL + path
}

// Get performs a GET request.
func (c *Client) Get(path string) (*APIResponse, error) {
	return c.request("GET", path, nil)
}

// Post performs a POST request with JSON body.
func (c *Client) Post(path string, body any) (*APIResponse, error) {
	return c.request("POST", path, body)
}

// Patch performs a PATCH request with JSON body.
func (c *Client) Patch(path string, body any) (*APIResponse, error) {
	return c.request("PATCH", path, body)
}

// PatchMultipart performs a PATCH request with multipart form data.
func (c *Client) PatchMultipart(path, fileField, filePath string, fields map[string]string) (*APIResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to open file: %v", err))
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	if err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to create form file: %v", err))
	}

	if _, err = io.Copy(part, file); err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to copy file: %v", err))
	}

	for key, value := range fields {
		if err = writer.WriteField(key, value); err != nil {
			return nil, errors.NewError(fmt.Sprintf("Failed to write form field: %v", err))
		}
	}

	if err = writer.Close(); err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to finalize multipart body: %v", err))
	}

	reqURL := c.buildURL(path)
	req, err := http.NewRequestWithContext(context.Background(), "PATCH", reqURL, &buf)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to create request: %v", err))
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to read response: %v", err))
	}

	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Location:   resp.Header.Get("Location"),
	}

	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp.Data); err != nil {
			return apiResp, nil
		}
	}

	if resp.StatusCode >= 400 {
		return apiResp, c.errorFromResponse(resp.StatusCode, respBody, resp.Header)
	}

	return apiResp, nil
}

// Put performs a PUT request with JSON body.
func (c *Client) Put(path string, body any) (*APIResponse, error) {
	return c.request("PUT", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) (*APIResponse, error) {
	return c.request("DELETE", path, nil)
}

// GetHTML performs a GET request expecting an HTML response.
// Unlike Get, it sets Accept: text/html and does not attempt JSON parsing.
func (c *Client) GetHTML(path string) (*APIResponse, error) {
	requestURL := c.buildURL(path)
	req, err := http.NewRequestWithContext(context.Background(), "GET", requestURL, nil)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to create request: %v", err))
	}

	c.setHeaders(req)
	req.Header.Set("Accept", "text/html")

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "> GET %s (HTML)\n", requestURL)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to read response: %v", err))
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "< %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
	}

	if resp.StatusCode >= 400 {
		return apiResp, c.errorFromResponse(resp.StatusCode, respBody, resp.Header)
	}

	return apiResp, nil
}

func (c *Client) request(method, path string, body any) (*APIResponse, error) {
	requestURL := c.buildURL(path)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.NewError(fmt.Sprintf("Failed to marshal request body: %v", err))
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, requestURL, reqBody)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to create request: %v", err))
	}

	c.setHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "> %s %s\n", method, requestURL)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to read response: %v", err))
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "< %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Location:   resp.Header.Get("Location"),
		LinkNext:   parseLinkNext(resp.Header.Get("Link")),
	}

	// Check for error status codes before parsing JSON,
	// since error responses may not be JSON (e.g. HTML 401 pages)
	if resp.StatusCode >= 400 {
		return apiResp, c.errorFromResponse(resp.StatusCode, respBody, resp.Header)
	}

	// Parse JSON body if present
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp.Data); err != nil {
			return apiResp, errors.NewError(fmt.Sprintf("Failed to parse JSON response: %v", err))
		}
	}

	return apiResp, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "fizzy-cli/1.0")
}

func (c *Client) sleep(d time.Duration) {
	if c.Sleeper != nil {
		c.Sleeper(d)
	} else {
		time.Sleep(d)
	}
}

const maxRetries = 3

var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// doWithRetry wraps HTTPClient.Do with retry logic for 429 and 5xx responses.
// Only retries GET/DELETE/PUT on 5xx; POST/PATCH only retry on 429.
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	idempotent := req.Method == "GET" || req.Method == "DELETE" || req.Method == "PUT"

	for attempt := range maxRetries + 1 {
		resp, err := c.HTTPClient.Do(req)

		// Network error — retry idempotent methods with backoff
		if err != nil {
			if attempt < maxRetries && idempotent {
				c.sleep(time.Duration(1<<uint(attempt)) * time.Second)
				resetBody(req)
				continue
			}
			return nil, err
		}

		// 429: always retry (server explicitly says "try again")
		if resp.StatusCode == 429 {
			if attempt < maxRetries {
				delay := parseRetryAfter(resp.Header.Get("Retry-After"))
				_ = resp.Body.Close()
				c.sleep(delay)
				resetBody(req)
				continue
			}
		}

		// 5xx: retry idempotent methods with exponential backoff
		if resp.StatusCode >= 500 && idempotent && attempt < maxRetries {
			_ = resp.Body.Close()
			c.sleep(time.Duration(1<<uint(attempt)) * time.Second)
			resetBody(req)
			continue
		}

		return resp, nil
	}

	// unreachable, but the compiler needs it
	return c.HTTPClient.Do(req)
}

// resetBody rewinds the request body for retry. Uses GetBody (set automatically
// by http.NewRequestWithContext for *bytes.Reader, *bytes.Buffer, *strings.Reader).
func resetBody(req *http.Request) {
	if req.GetBody != nil {
		req.Body, _ = req.GetBody()
	}
}

// parseRetryAfter parses the Retry-After header value as seconds.
// Returns a default of 1 second if the header is missing or unparseable.
func parseRetryAfter(value string) time.Duration {
	const maxRetryDelay = 300 // 5 minutes; anything larger is almost certainly bogus

	if value == "" {
		return time.Second
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		if seconds > maxRetryDelay {
			seconds = maxRetryDelay
		}
		return time.Duration(seconds) * time.Second
	}
	// Try HTTP-date format
	if t, err := http.ParseTime(value); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			if delay > maxRetryDelay*time.Second {
				return maxRetryDelay * time.Second
			}
			return delay
		}
	}
	return time.Second
}

func (c *Client) errorFromResponse(status int, body []byte, header http.Header) error {
	// Try to parse error message from response
	var errResp struct {
		Error string `json:"error"`
	}

	message := http.StatusText(status)
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		message = errResp.Error
	}

	if status == 429 {
		retryAfter := 0
		if ra := header.Get("Retry-After"); ra != "" {
			if seconds, err := strconv.Atoi(ra); err == nil {
				retryAfter = seconds
			}
		}
		e := errors.FromHTTPStatus(429, message)
		if retryAfter > 0 {
			e = output.ErrRateLimit(retryAfter)
			if message != "" && message != http.StatusText(http.StatusTooManyRequests) {
				e.Message = message
			}
		}
		return e
	}

	return errors.FromHTTPStatus(status, message)
}

// parseLinkNext extracts the "next" URL from a Link header.
func parseLinkNext(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	matches := linkNextRe.FindStringSubmatch(linkHeader)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// GetWithPagination fetches all pages of a paginated endpoint.
func (c *Client) GetWithPagination(path string, fetchAll bool) (*APIResponse, error) {
	resp, err := c.Get(path)
	if err != nil {
		return resp, err
	}

	if !fetchAll || resp.LinkNext == "" {
		return resp, nil
	}

	// Collect all data
	var allData []any
	if arr, ok := resp.Data.([]any); ok {
		allData = append(allData, arr...)
	}

	// Fetch remaining pages
	nextURL := resp.LinkNext
	for nextURL != "" {
		pageResp, err := c.Get(nextURL)
		if err != nil {
			return nil, err
		}

		if arr, ok := pageResp.Data.([]any); ok {
			allData = append(allData, arr...)
		}

		nextURL = pageResp.LinkNext
	}

	resp.Data = allData
	resp.LinkNext = ""
	return resp, nil
}

// FollowLocation fetches the resource at the Location header.
func (c *Client) FollowLocation(location string) (*APIResponse, error) {
	if location == "" {
		return nil, nil
	}
	return c.Get(location)
}

// UploadFile uploads a file using the direct upload flow.
func (c *Client) UploadFile(filePath string) (*APIResponse, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to open file: %v", err))
	}
	defer func() { _ = file.Close() }()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to stat file: %v", err))
	}

	// Read file content for checksum
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to read file: %v", err))
	}

	filename := filepath.Base(filePath)
	contentType := detectContentType(filePath)
	checksum := computeChecksum(fileContent)

	// Step 1: Create blob
	blobReq := map[string]any{
		"blob": map[string]any{
			"filename":     filename,
			"byte_size":    fileInfo.Size(),
			"content_type": contentType,
			"checksum":     checksum,
		},
	}

	createResp, err := c.Post("/rails/active_storage/direct_uploads", blobReq)
	if err != nil {
		return nil, err
	}

	// Parse the response to get upload URL and signed_id
	blobData, ok := createResp.Data.(map[string]any)
	if !ok {
		return nil, errors.NewError("Invalid blob creation response")
	}

	directUploadData, ok := blobData["direct_upload"].(map[string]any)
	if !ok {
		return nil, errors.NewError("Missing direct_upload in response")
	}

	uploadURL, ok := directUploadData["url"].(string)
	if !ok {
		return nil, errors.NewError("Missing upload URL in response")
	}

	headers, _ := directUploadData["headers"].(map[string]any)

	signedID, ok := blobData["signed_id"].(string)
	if !ok {
		return nil, errors.NewError("Missing signed_id in response")
	}

	// Get attachable_sgid for use in action-text-attachment
	attachableSGID, _ := blobData["attachable_sgid"].(string)

	// Step 2: Upload file to the direct upload URL
	uploadReq, err := http.NewRequestWithContext(context.Background(), "PUT", uploadURL, bytes.NewReader(fileContent))
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to create upload request: %v", err))
	}

	// Set headers from the direct_upload response
	for key, value := range headers {
		if strVal, ok := value.(string); ok {
			uploadReq.Header.Set(key, strVal)
		}
	}

	uploadResp, err := c.doWithRetry(uploadReq)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Upload failed: %v", err))
	}
	defer func() { _ = uploadResp.Body.Close() }()

	if uploadResp.StatusCode >= 400 {
		body, _ := io.ReadAll(uploadResp.Body)
		return nil, errors.NewError(fmt.Sprintf("Upload failed: %d %s", uploadResp.StatusCode, string(body)))
	}

	// Return the signed_id and attachable_sgid
	responseData := map[string]any{
		"signed_id": signedID,
	}
	if attachableSGID != "" {
		responseData["attachable_sgid"] = attachableSGID
	}

	return &APIResponse{
		StatusCode: 200,
		Data:       responseData,
	}, nil
}

// UploadFileMultipart uploads a file using multipart form data.
func (c *Client) UploadFileMultipart(path, fieldName, filePath string, extraFields map[string]string) (*APIResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to open file: %v", err))
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to create form file: %v", err))
	}

	if _, err = io.Copy(part, file); err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to copy file: %v", err))
	}

	// Add extra fields
	for key, value := range extraFields {
		if err = writer.WriteField(key, value); err != nil {
			return nil, errors.NewError(fmt.Sprintf("Failed to write form field: %v", err))
		}
	}

	if err = writer.Close(); err != nil {
		return nil, errors.NewError(fmt.Sprintf("Failed to finalize multipart body: %v", err))
	}

	reqURL := c.buildURL(path)
	req, err := http.NewRequestWithContext(context.Background(), "POST", reqURL, &buf)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to create request: %v", err))
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Failed to read response: %v", err))
	}

	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Location:   resp.Header.Get("Location"),
	}

	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &apiResp.Data); err != nil {
			return apiResp, errors.NewError(fmt.Sprintf("Failed to parse JSON response: %v", err))
		}
	}

	if resp.StatusCode >= 400 {
		return apiResp, c.errorFromResponse(resp.StatusCode, respBody, resp.Header)
	}

	return apiResp, nil
}

// computeChecksum computes the base64-encoded MD5 checksum of content.
func computeChecksum(content []byte) string {
	hash := md5.Sum(content)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func detectContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	contentTypes := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".html": "text/html",
		".json": "application/json",
		".xml":  "application/xml",
		".zip":  "application/zip",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// ParsePage extracts page number from a URL query string.
func ParsePage(nextURL string) string {
	if nextURL == "" {
		return ""
	}
	u, err := url.Parse(nextURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("page")
}

// DownloadFile downloads a file from a URL (following redirects) and saves it to the specified path.
// The URL should be a relative path like /6085671/rails/active_storage/blobs/redirect/...
func (c *Client) DownloadFile(urlPath string, destPath string) error {
	requestURL := c.buildURL(urlPath)

	req, err := http.NewRequestWithContext(context.Background(), "GET", requestURL, nil)
	if err != nil {
		return errors.NewNetworkError(fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("User-Agent", "fizzy-cli/1.0")

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "> GET %s\n", requestURL)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return errors.NewNetworkError(fmt.Sprintf("Request failed: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "< %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return errors.NewError(fmt.Sprintf("Download failed: %d %s", resp.StatusCode, string(body)))
	}

	// Create the destination file
	out, err := os.Create(destPath)
	if err != nil {
		return errors.NewError(fmt.Sprintf("Failed to create file: %v", err))
	}

	// Copy the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()
		_ = os.Remove(destPath)
		return errors.NewError(fmt.Sprintf("Failed to write file: %v", err))
	}

	return out.Close()
}
