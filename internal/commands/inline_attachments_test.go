package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
)

func TestAppendAttachmentTags(t *testing.T) {
	got := appendAttachmentTags("See attached", []string{"sgid-1", "sgid-2"})
	want := strings.Join([]string{
		"See attached",
		`<action-text-attachment sgid="sgid-1"></action-text-attachment>`,
		`<action-text-attachment sgid="sgid-2"></action-text-attachment>`,
	}, "\n")

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestAppendInlineAttachmentsToContentPreservesOrderAndUploadsEachPath(t *testing.T) {
	tempDir := t.TempDir()
	pathA := writeTestAttachmentFile(t, tempDir, "a.txt", "a")
	pathB := writeTestAttachmentFile(t, tempDir, "b.txt", "b")

	mock := NewMockClient()
	mock.UploadFileResponses = []*client.APIResponse{
		{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-a-1"}},
		{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-b"}},
		{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-a-2"}},
	}
	SetTestMode(mock)
	defer ResetTestMode()

	got, err := appendInlineAttachmentsToContent("Body", []string{pathA, pathB, pathA})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := strings.Join([]string{
		"Body",
		`<action-text-attachment sgid="sgid-a-1"></action-text-attachment>`,
		`<action-text-attachment sgid="sgid-b"></action-text-attachment>`,
		`<action-text-attachment sgid="sgid-a-2"></action-text-attachment>`,
	}, "\n")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	if len(mock.UploadFileCalls) != 3 {
		t.Fatalf("expected 3 uploads, got %d", len(mock.UploadFileCalls))
	}
	if mock.UploadFileCalls[0] != pathA || mock.UploadFileCalls[1] != pathB || mock.UploadFileCalls[2] != pathA {
		t.Fatalf("unexpected upload order: %#v", mock.UploadFileCalls)
	}
}

func TestAppendInlineAttachmentsToContentAllowsAttachmentOnlyBody(t *testing.T) {
	tempDir := t.TempDir()
	path := writeTestAttachmentFile(t, tempDir, "only.txt", "content")

	mock := NewMockClient()
	mock.UploadFileResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-only"}}
	SetTestMode(mock)
	defer ResetTestMode()

	got, err := appendInlineAttachmentsToContent("", []string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := `<action-text-attachment sgid="sgid-only"></action-text-attachment>`
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestAppendInlineAttachmentsToContentErrorsForMissingFile(t *testing.T) {
	mock := NewMockClient()
	SetTestMode(mock)
	defer ResetTestMode()

	_, err := appendInlineAttachmentsToContent("Body", []string{filepath.Join(t.TempDir(), "missing.txt")})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "File not found") {
		t.Fatalf("expected file not found error, got %v", err)
	}
	if len(mock.UploadFileCalls) != 0 {
		t.Fatalf("expected no upload attempts, got %d", len(mock.UploadFileCalls))
	}
}

func TestUploadAttachableSGIDsRequiresAttachableSGID(t *testing.T) {
	tempDir := t.TempDir()
	path := writeTestAttachmentFile(t, tempDir, "missing-sgid.txt", "content")

	mock := NewMockClient()
	mock.UploadFileResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"signed_id": "signed-only"}}
	SetTestMode(mock)
	defer ResetTestMode()

	_, err := uploadAttachableSGIDs([]string{path})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing attachable_sgid") {
		t.Fatalf("expected missing attachable_sgid error, got %v", err)
	}
}

func writeTestAttachmentFile(t *testing.T, dir string, name string, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test attachment file: %v", err)
	}
	return path
}
