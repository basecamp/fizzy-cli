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
	"strings"
	"time"

	"github.com/basecamp/fizzy-cli/internal/errors"
)

// Client is an HTTP client for the Fizzy API.
type Client struct {
	BaseURL    string
	Token      string
	Account    string
	HTTPClient *http.Client
	Verbose    bool
}

// APIResponse represents a response from the API.
type APIResponse struct {
	StatusCode int
	Body       []byte
	Location   string
	LinkNext   string
	Data       interface{}
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
func (c *Client) Post(path string, body interface{}) (*APIResponse, error) {
	return c.request("POST", path, body)
}

// Patch performs a PATCH request with JSON body.
func (c *Client) Patch(path string, body interface{}) (*APIResponse, error) {
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

	resp, err := c.HTTPClient.Do(req)
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
		return apiResp, c.errorFromResponse(resp.StatusCode, respBody)
	}

	return apiResp, nil
}

// Put performs a PUT request with JSON body.
func (c *Client) Put(path string, body interface{}) (*APIResponse, error) {
	return c.request("PUT", path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) (*APIResponse, error) {
	return c.request("DELETE", path, nil)
}

func (c *Client) request(method, path string, body interface{}) (*APIResponse, error) {
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

	resp, err := c.HTTPClient.Do(req)
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
		return apiResp, c.errorFromResponse(resp.StatusCode, respBody)
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

func (c *Client) errorFromResponse(status int, body []byte) error {
	// Try to parse error message from response
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return errors.FromHTTPStatus(status, errResp.Error)
	}

	return errors.FromHTTPStatus(status, http.StatusText(status))
}

// parseLinkNext extracts the "next" URL from a Link header.
func parseLinkNext(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	// Parse Link header: <url>; rel="next"
	re := regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)
	matches := re.FindStringSubmatch(linkHeader)
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
	var allData []interface{}
	if arr, ok := resp.Data.([]interface{}); ok {
		allData = append(allData, arr...)
	}

	// Fetch remaining pages
	nextURL := resp.LinkNext
	for nextURL != "" {
		pageResp, err := c.Get(nextURL)
		if err != nil {
			return nil, err
		}

		if arr, ok := pageResp.Data.([]interface{}); ok {
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
	blobReq := map[string]interface{}{
		"blob": map[string]interface{}{
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
	blobData, ok := createResp.Data.(map[string]interface{})
	if !ok {
		return nil, errors.NewError("Invalid blob creation response")
	}

	directUploadData, ok := blobData["direct_upload"].(map[string]interface{})
	if !ok {
		return nil, errors.NewError("Missing direct_upload in response")
	}

	uploadURL, ok := directUploadData["url"].(string)
	if !ok {
		return nil, errors.NewError("Missing upload URL in response")
	}

	headers, _ := directUploadData["headers"].(map[string]interface{})

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

	uploadResp, err := c.HTTPClient.Do(uploadReq)
	if err != nil {
		return nil, errors.NewNetworkError(fmt.Sprintf("Upload failed: %v", err))
	}
	defer func() { _ = uploadResp.Body.Close() }()

	if uploadResp.StatusCode >= 400 {
		body, _ := io.ReadAll(uploadResp.Body)
		return nil, errors.NewError(fmt.Sprintf("Upload failed: %d %s", uploadResp.StatusCode, string(body)))
	}

	// Return the signed_id and attachable_sgid
	responseData := map[string]interface{}{
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

	resp, err := c.HTTPClient.Do(req)
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
		return apiResp, c.errorFromResponse(resp.StatusCode, respBody)
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

	resp, err := c.HTTPClient.Do(req)
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
