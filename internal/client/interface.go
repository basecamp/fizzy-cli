package client

// API defines the interface for API operations.
// This allows for mocking in tests.
type API interface {
	Get(path string) (*APIResponse, error)
	Post(path string, body any) (*APIResponse, error)
	Patch(path string, body any) (*APIResponse, error)
	PatchMultipart(path, fileField, filePath string, fields map[string]string) (*APIResponse, error)
	Put(path string, body any) (*APIResponse, error)
	Delete(path string) (*APIResponse, error)
	GetWithPagination(path string, fetchAll bool) (*APIResponse, error)
	FollowLocation(location string) (*APIResponse, error)
	UploadFile(filePath string) (*APIResponse, error)
	DownloadFile(urlPath string, destPath string) error
	GetHTML(path string) (*APIResponse, error)
}

// Ensure Client implements API interface
var _ API = (*Client)(nil)
