package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestExtractCommentAttachments(t *testing.T) {
	tests := []struct {
		name          string
		comments      []any
		expectedCount int
		expectedFirst *CommentAttachment
	}{
		{
			name: "comment with attachment",
			comments: []any{
				map[string]any{
					"id": "comment-1",
					"body": map[string]any{
						"html": `<action-text-attachment sgid="sgid1" content-type="image/png" filename="screenshot.png" filesize="5000" width="800" height="600">
							<a href="/blobs/blob1/screenshot.png?disposition=attachment">Download</a>
						</action-text-attachment>`,
						"plain_text": "screenshot.png",
					},
				},
			},
			expectedCount: 1,
			expectedFirst: &CommentAttachment{
				Attachment: Attachment{
					Index:       1,
					Filename:    "screenshot.png",
					ContentType: "image/png",
					Filesize:    5000,
					Width:       800,
					Height:      600,
					SGID:        "sgid1",
					DownloadURL: "/blobs/blob1/screenshot.png?disposition=attachment",
				},
				CommentID: "comment-1",
			},
		},
		{
			name: "multiple comments with attachments",
			comments: []any{
				map[string]any{
					"id": "comment-1",
					"body": map[string]any{
						"html": `<action-text-attachment sgid="sgid1" content-type="image/png" filename="img1.png" filesize="1000">
							<a href="/blobs/blob1/img1.png?disposition=attachment">Download</a>
						</action-text-attachment>`,
					},
				},
				map[string]any{
					"id": "comment-2",
					"body": map[string]any{
						"html": `<action-text-attachment sgid="sgid2" content-type="image/jpeg" filename="img2.jpg" filesize="2000">
							<a href="/blobs/blob2/img2.jpg?disposition=attachment">Download</a>
						</action-text-attachment>`,
					},
				},
			},
			expectedCount: 2,
		},
		{
			name: "comment without attachments",
			comments: []any{
				map[string]any{
					"id": "comment-1",
					"body": map[string]any{
						"html":       "<p>Just text</p>",
						"plain_text": "Just text",
					},
				},
			},
			expectedCount: 0,
		},
		{
			name:          "empty comments",
			comments:      []any{},
			expectedCount: 0,
		},
		{
			name: "mixed comments with and without attachments",
			comments: []any{
				map[string]any{
					"id": "comment-1",
					"body": map[string]any{
						"html": "<p>No attachment here</p>",
					},
				},
				map[string]any{
					"id": "comment-2",
					"body": map[string]any{
						"html": `<action-text-attachment sgid="sgid1" content-type="image/png" filename="found.png" filesize="3000">
							<a href="/blobs/blob1/found.png?disposition=attachment">Download</a>
						</action-text-attachment>`,
					},
				},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCommentAttachments(tt.comments)

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d attachments, got %d", tt.expectedCount, len(result))
				return
			}

			if tt.expectedFirst != nil && len(result) > 0 {
				actual := result[0]
				if actual.Index != tt.expectedFirst.Index {
					t.Errorf("expected index %d, got %d", tt.expectedFirst.Index, actual.Index)
				}
				if actual.Filename != tt.expectedFirst.Filename {
					t.Errorf("expected filename %q, got %q", tt.expectedFirst.Filename, actual.Filename)
				}
				if actual.CommentID != tt.expectedFirst.CommentID {
					t.Errorf("expected comment_id %q, got %q", tt.expectedFirst.CommentID, actual.CommentID)
				}
				if actual.DownloadURL != tt.expectedFirst.DownloadURL {
					t.Errorf("expected download_url %q, got %q", tt.expectedFirst.DownloadURL, actual.DownloadURL)
				}
			}

			// Verify global indexing is sequential
			for i, a := range result {
				if a.Index != i+1 {
					t.Errorf("attachment %d: expected index %d, got %d", i, i+1, a.Index)
				}
			}
		})
	}
}

func TestCommentAttachmentsShowCommand(t *testing.T) {
	t.Run("shows attachments from comments", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{
					"id": "comment-1",
					"body": map[string]any{
						"html": `<action-text-attachment sgid="sgid1" content-type="image/png" filename="test.png" filesize="1000">
							<a href="/blobs/blob1/test.png?disposition=attachment">Download</a>
						</action-text-attachment>`,
					},
				},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsShowCard = "172"
		err := commentAttachmentsShowCmd.RunE(commentAttachmentsShowCmd, []string{})
		commentAttachmentsShowCard = ""

		assertExitCode(t, err, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Errorf("expected success, got error")
		}
		if mock.GetWithPaginationCalls[0].Path != "/cards/172/comments.json" {
			t.Errorf("expected path '/cards/172/comments.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsShowCard = ""
		err := commentAttachmentsShowCmd.RunE(commentAttachmentsShowCmd, []string{})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestCommentAttachmentsDownloadCommand(t *testing.T) {
	commentsWithAttachment := []any{
		map[string]any{
			"id": "comment-1",
			"body": map[string]any{
				"html": `<action-text-attachment sgid="sgid1" content-type="image/png" filename="test.png" filesize="1000">
					<a href="/blobs/blob1/test.png?disposition=attachment">Download</a>
				</action-text-attachment>`,
			},
		},
	}

	commentsWithMultipleAttachments := []any{
		map[string]any{
			"id": "comment-1",
			"body": map[string]any{
				"html": `<action-text-attachment sgid="sgid1" content-type="image/png" filename="img1.png" filesize="1000">
					<a href="/blobs/blob1/img1.png?disposition=attachment">Download</a>
				</action-text-attachment>
				<action-text-attachment sgid="sgid2" content-type="application/pdf" filename="doc.pdf" filesize="2000">
					<a href="/blobs/blob2/doc.pdf?disposition=attachment">Download</a>
				</action-text-attachment>`,
			},
		},
	}

	t.Run("downloads all comment attachments", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       commentsWithMultipleAttachments,
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsDownloadCard = "172"
		err := commentAttachmentsDownloadCmd.RunE(commentAttachmentsDownloadCmd, []string{})
		commentAttachmentsDownloadCard = ""

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Errorf("expected success, got error: %v", result.Response)
		}
		if len(mock.DownloadFileCalls) != 2 {
			t.Errorf("expected 2 downloads, got %d", len(mock.DownloadFileCalls))
		}
	})

	t.Run("downloads single attachment by index", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       commentsWithAttachment,
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsDownloadCard = "172"
		err := commentAttachmentsDownloadCmd.RunE(commentAttachmentsDownloadCmd, []string{"1"})
		commentAttachmentsDownloadCard = ""

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Errorf("expected success, got error: %v", result.Response)
		}
		if len(mock.DownloadFileCalls) != 1 {
			t.Errorf("expected 1 download, got %d", len(mock.DownloadFileCalls))
		}
		if mock.DownloadFileCalls[0].URLPath != "/blobs/blob1/test.png?disposition=attachment" {
			t.Errorf("expected download URL '/blobs/blob1/test.png?disposition=attachment', got '%s'", mock.DownloadFileCalls[0].URLPath)
		}
	})

	t.Run("errors on no attachments", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{
					"id": "comment-1",
					"body": map[string]any{
						"html": "<p>No images here</p>",
					},
				},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsDownloadCard = "172"
		err := commentAttachmentsDownloadCmd.RunE(commentAttachmentsDownloadCmd, []string{})
		commentAttachmentsDownloadCard = ""

		// Command returns error for no attachments
		if err == nil {
			if result.Response.OK {
				t.Error("expected error, got success")
			}
		}
	})

	t.Run("errors on invalid index", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       commentsWithAttachment,
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsDownloadCard = "172"
		err := commentAttachmentsDownloadCmd.RunE(commentAttachmentsDownloadCmd, []string{"abc"})
		commentAttachmentsDownloadCard = ""

		if err == nil {
			if result.Response.OK {
				t.Error("expected error for non-numeric index")
			}
		}
	})

	t.Run("errors on out of range index", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       commentsWithAttachment,
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsDownloadCard = "172"
		err := commentAttachmentsDownloadCmd.RunE(commentAttachmentsDownloadCmd, []string{"5"})
		commentAttachmentsDownloadCard = ""

		if err == nil {
			if result.Response.OK {
				t.Error("expected error for out of range index")
			}
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentAttachmentsDownloadCard = ""
		err := commentAttachmentsDownloadCmd.RunE(commentAttachmentsDownloadCmd, []string{})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}
