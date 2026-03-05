package commands

import (
	stderrors "errors"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

// MockClient is a mock implementation of client.API for testing.
type MockClient struct {
	// Response to return for each method
	GetResponse               *client.APIResponse
	PostResponse              *client.APIResponse
	PatchResponse             *client.APIResponse
	PutResponse               *client.APIResponse
	DeleteResponse            *client.APIResponse
	GetWithPaginationResponse *client.APIResponse
	FollowLocationResponse    *client.APIResponse
	UploadFileResponse        *client.APIResponse

	PatchMultipartResponse *client.APIResponse

	// Errors to return for each method
	GetError               error
	PostError              error
	PatchError             error
	PatchMultipartError    error
	PutError               error
	DeleteError            error
	GetWithPaginationError error
	FollowLocationError    error
	UploadFileError        error
	DownloadFileError      error

	// Captured calls for verification
	GetCalls               []MockCall
	PostCalls              []MockCall
	PatchCalls             []MockCall
	PatchMultipartCalls    []MockCall
	PutCalls               []MockCall
	DeleteCalls            []MockCall
	GetWithPaginationCalls []MockCall
	FollowLocationCalls    []string
	UploadFileCalls        []string
	DownloadFileCalls      []MockDownloadCall
}

// MockDownloadCall represents a captured download call.
type MockDownloadCall struct {
	URLPath  string
	DestPath string
}

// MockCall represents a captured API call.
type MockCall struct {
	Path string
	Body any
}

// NewMockClient creates a new mock client with default success responses.
func NewMockClient() *MockClient {
	return &MockClient{
		GetResponse: &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		},
		PostResponse: &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/resource/123",
			Data:       map[string]any{"id": "123"},
		},
		PatchResponse: &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		},
		PatchMultipartResponse: &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		},
		PutResponse: &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		},
		DeleteResponse: &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		},
		GetWithPaginationResponse: &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		},
		FollowLocationResponse: &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		},
		UploadFileResponse: &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"signed_id": "test-signed-id"},
		},
	}
}

func (m *MockClient) Get(path string) (*client.APIResponse, error) {
	m.GetCalls = append(m.GetCalls, MockCall{Path: path})
	if m.GetError != nil {
		return nil, m.GetError
	}
	return m.GetResponse, nil
}

func (m *MockClient) Post(path string, body any) (*client.APIResponse, error) {
	m.PostCalls = append(m.PostCalls, MockCall{Path: path, Body: body})
	if m.PostError != nil {
		return nil, m.PostError
	}
	return m.PostResponse, nil
}

func (m *MockClient) Patch(path string, body any) (*client.APIResponse, error) {
	m.PatchCalls = append(m.PatchCalls, MockCall{Path: path, Body: body})
	if m.PatchError != nil {
		return nil, m.PatchError
	}
	return m.PatchResponse, nil
}

func (m *MockClient) PatchMultipart(path, fileField, filePath string, fields map[string]string) (*client.APIResponse, error) {
	m.PatchMultipartCalls = append(m.PatchMultipartCalls, MockCall{Path: path, Body: map[string]any{
		"file_field": fileField,
		"file_path":  filePath,
		"fields":     fields,
	}})
	if m.PatchMultipartError != nil {
		return nil, m.PatchMultipartError
	}
	return m.PatchMultipartResponse, nil
}

func (m *MockClient) Put(path string, body any) (*client.APIResponse, error) {
	m.PutCalls = append(m.PutCalls, MockCall{Path: path, Body: body})
	if m.PutError != nil {
		return nil, m.PutError
	}
	return m.PutResponse, nil
}

func (m *MockClient) Delete(path string) (*client.APIResponse, error) {
	m.DeleteCalls = append(m.DeleteCalls, MockCall{Path: path})
	if m.DeleteError != nil {
		return nil, m.DeleteError
	}
	return m.DeleteResponse, nil
}

func (m *MockClient) GetWithPagination(path string, fetchAll bool) (*client.APIResponse, error) {
	m.GetWithPaginationCalls = append(m.GetWithPaginationCalls, MockCall{Path: path, Body: fetchAll})
	if m.GetWithPaginationError != nil {
		return nil, m.GetWithPaginationError
	}
	return m.GetWithPaginationResponse, nil
}

func (m *MockClient) FollowLocation(location string) (*client.APIResponse, error) {
	m.FollowLocationCalls = append(m.FollowLocationCalls, location)
	if m.FollowLocationError != nil {
		return nil, m.FollowLocationError
	}
	return m.FollowLocationResponse, nil
}

func (m *MockClient) UploadFile(filePath string) (*client.APIResponse, error) {
	m.UploadFileCalls = append(m.UploadFileCalls, filePath)
	if m.UploadFileError != nil {
		return nil, m.UploadFileError
	}
	return m.UploadFileResponse, nil
}

func (m *MockClient) DownloadFile(urlPath string, destPath string) error {
	m.DownloadFileCalls = append(m.DownloadFileCalls, MockDownloadCall{URLPath: urlPath, DestPath: destPath})
	if m.DownloadFileError != nil {
		return m.DownloadFileError
	}
	return nil
}

// Helper functions for creating common responses

// WithGetData sets the data returned by Get calls.
func (m *MockClient) WithGetData(data any) *MockClient {
	m.GetResponse.Data = data
	return m
}

// WithPostData sets the data returned by Post calls.
func (m *MockClient) WithPostData(data any) *MockClient {
	m.PostResponse.Data = data
	return m
}

// WithPatchData sets the data returned by Patch calls.
func (m *MockClient) WithPatchData(data any) *MockClient {
	m.PatchResponse.Data = data
	return m
}

// WithListData sets the data returned by GetWithPagination calls.
func (m *MockClient) WithListData(data []any) *MockClient {
	m.GetWithPaginationResponse.Data = data
	return m
}

// WithFollowLocationData sets the data returned by FollowLocation calls.
func (m *MockClient) WithFollowLocationData(data any) *MockClient {
	m.FollowLocationResponse.Data = data
	return m
}

// WithNotFoundError sets a 404 error for Get calls.
func (m *MockClient) WithNotFoundError() *MockClient {
	m.GetError = errors.NewNotFoundError("Not found")
	return m
}

// WithAuthError sets a 401 error for Get calls.
func (m *MockClient) WithAuthError() *MockClient {
	m.GetError = errors.NewAuthError("Unauthorized")
	return m
}

// WithValidationError sets a 422 error for Post calls.
func (m *MockClient) WithValidationError(message string) *MockClient {
	m.PostError = errors.NewValidationError(message)
	return m
}

// Ensure MockClient implements client.API
var _ client.API = (*MockClient)(nil)

// assertExitCode checks that an error has the expected exit code.
// For exit code 0, it asserts that err is nil.
func assertExitCode(t *testing.T, err error, expected int) {
	t.Helper()
	if expected == 0 {
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		return
	}
	if err == nil {
		t.Errorf("expected error with exit code %d, got nil", expected)
		return
	}
	var cliErr *errors.CLIError
	if !stderrors.As(err, &cliErr) {
		t.Errorf("expected CLIError with exit code %d, got non-CLIError: %v", expected, err)
		return
	}
	if cliErr.ExitCode() != expected {
		t.Errorf("expected exit code %d, got %d (error: %v)", expected, cliErr.ExitCode(), err)
	}
}
