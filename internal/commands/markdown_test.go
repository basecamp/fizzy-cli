package commands

import (
	"strings"
	"testing"
)

func TestMarkdownToHTML(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		shouldContain    []string
		shouldNotContain []string
		exactOutput      string
	}{
		{
			name:        "plain text passes through unchanged",
			input:       "Just a simple comment",
			exactOutput: "Just a simple comment",
		},
		{
			name:          "common markdown emphasis is converted",
			input:         "This is **bold** and _italic_",
			shouldContain: []string{"<strong>bold</strong>", "<em>italic</em>"},
		},
		{
			name:          "markdown list is converted",
			input:         "- first\n- second",
			shouldContain: []string{"<ul>", "<li>first</li>", "<li>second</li>"},
		},
		{
			name:             "backtick-wrapped attachment tag is escaped",
			input:            "Manually construct the `<action-text-attachment sgid=\"...\" content-type=\"application/vnd.actiontext.mention\"></action-text-attachment>`",
			shouldContain:    []string{"<code>", "&lt;action-text-attachment"},
			shouldNotContain: []string{"<action-text-attachment"},
		},
		{
			name:          "raw attachment tag passes through unchanged",
			input:         `<action-text-attachment sgid="REAL_SGID"></action-text-attachment>`,
			shouldContain: []string{`<action-text-attachment sgid="REAL_SGID">`},
		},
		{
			name:          "plain HTML passes through unchanged",
			input:         `<p>Hello <strong>world</strong></p>`,
			shouldContain: []string{"<strong>world</strong>"},
		},
		{
			name:          "attachment with surrounding HTML passes through",
			input:         `<p>See image: <action-text-attachment sgid="REAL_SGID"></action-text-attachment></p>`,
			shouldContain: []string{`<action-text-attachment sgid="REAL_SGID">`},
		},
		{
			name:             "mixed HTML and markdown with backtick attachment",
			input:            `<br>2. Manually construct the ` + "`" + `<action-text-attachment sgid="..." ...=""></action-text-attachment>` + "`",
			shouldContain:    []string{"<code>", "&lt;action-text-attachment"},
			shouldNotContain: []string{"<action-text-attachment sgid"},
		},
		{
			name:             "backtick attachment inside HTML block is escaped",
			input:            "<p>To embed attachments use: `<action-text-attachment sgid=\"...\" content-type=\"...\"></action-text-attachment>`</p>",
			shouldContain:    []string{"<code>", "&lt;action-text-attachment"},
			shouldNotContain: []string{"<action-text-attachment sgid"},
		},
		{
			name:  "real attachment alongside backtick attachment",
			input: "<p>See:</p><action-text-attachment sgid=\"REAL\"></action-text-attachment><p>Example: `<action-text-attachment sgid=\"...\" content-type=\"...\"></action-text-attachment>`</p>",
			shouldContain: []string{
				`<action-text-attachment sgid="REAL">`,
				"<code>",
				"&lt;action-text-attachment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := markdownToHTML(tt.input)

			if tt.exactOutput != "" {
				if result != tt.exactOutput {
					t.Errorf("expected exact output %q, got %q", tt.exactOutput, result)
				}
				return
			}

			for _, s := range tt.shouldContain {
				if !strings.Contains(result, s) {
					t.Errorf("expected output to contain %q\ngot: %s", s, result)
				}
			}

			for _, s := range tt.shouldNotContain {
				if strings.Contains(result, s) {
					t.Errorf("expected output NOT to contain %q\ngot: %s", s, result)
				}
			}
		})
	}
}
