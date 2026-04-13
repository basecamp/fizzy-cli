package commands

import (
	"fmt"
	"html"
	"os"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/errors"
)

func resolveRichTextContent(content string, filePath string) (string, error) {
	if filePath != "" {
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return markdownToHTML(string(fileContent)), nil
	}
	if content == "" {
		return "", nil
	}
	return markdownToHTML(content), nil
}

func appendInlineAttachmentsToContent(content string, paths []string) (string, error) {
	if len(paths) == 0 {
		return content, nil
	}

	sgids, err := uploadAttachableSGIDs(paths)
	if err != nil {
		return "", err
	}

	return appendAttachmentTags(content, sgids), nil
}

func uploadAttachableSGIDs(paths []string) ([]string, error) {
	client := getClient()
	sgids := make([]string, 0, len(paths))

	for _, path := range paths {
		if err := validateAttachmentPath(path); err != nil {
			return nil, err
		}

		resp, err := client.UploadFile(path)
		if err != nil {
			return nil, err
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			return nil, errors.NewError(fmt.Sprintf("Invalid attachment upload response for %s", path))
		}

		sgid, _ := data["attachable_sgid"].(string)
		if sgid == "" {
			return nil, errors.NewError(fmt.Sprintf("Upload response missing attachable_sgid for %s", path))
		}

		sgids = append(sgids, sgid)
	}

	return sgids, nil
}

func validateAttachmentPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.NewInvalidArgsError("attachment path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.NewError("File not found: " + path)
		}
		return errors.NewError(fmt.Sprintf("Failed to stat attachment %s: %v", path, err))
	}
	if info.IsDir() {
		return errors.NewError("Attachment path is a directory: " + path)
	}

	file, err := os.Open(path)
	if err != nil {
		return errors.NewError(fmt.Sprintf("Failed to open attachment %s: %v", path, err))
	}
	if err := file.Close(); err != nil {
		return errors.NewError(fmt.Sprintf("Failed to close attachment %s: %v", path, err))
	}
	return nil
}

func appendAttachmentTags(content string, sgids []string) string {
	if len(sgids) == 0 {
		return content
	}

	tags := make([]string, 0, len(sgids))
	for _, sgid := range sgids {
		tags = append(tags, actionTextAttachmentTag(sgid))
	}

	attachments := strings.Join(tags, "\n")
	if content == "" {
		return attachments
	}
	if strings.HasSuffix(content, "\n") {
		return content + attachments
	}
	return content + "\n" + attachments
}

func actionTextAttachmentTag(sgid string) string {
	return `<action-text-attachment sgid="` + html.EscapeString(sgid) + `"></action-text-attachment>`
}
