package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
)

const samplePromptUsersHTML = `
<lexxy-prompt-item search="Wayne Smith WS me" sgid="wayne-sgid-123">
  <template type="menu">
    <img aria-hidden="true" title="Wayne Smith" src="/123/users/u1/avatar" width="48" height="48" />
    Wayne Smith
  </template>
  <template type="editor">
    <img aria-hidden="true" title="Wayne Smith" src="/123/users/u1/avatar" width="48" height="48" />
    Wayne
  </template>
</lexxy-prompt-item>
<lexxy-prompt-item search="Bushra Gul BG" sgid="bushra-sgid-456">
  <template type="menu">
    <img aria-hidden="true" title="Bushra Gul" src="/123/users/u2/avatar" width="48" height="48" />
    Bushra Gul
  </template>
  <template type="editor">
    <img aria-hidden="true" title="Bushra Gul" src="/123/users/u2/avatar" width="48" height="48" />
    Bushra
  </template>
</lexxy-prompt-item>
<lexxy-prompt-item search="Kennedy K" sgid="kennedy-sgid-789">
  <template type="menu">
    <img aria-hidden="true" title="Kennedy" src="/123/users/u3/avatar" width="48" height="48" />
    Kennedy
  </template>
  <template type="editor">
    <img aria-hidden="true" title="Kennedy" src="/123/users/u3/avatar" width="48" height="48" />
    Kennedy
  </template>
</lexxy-prompt-item>
`

// ambiguousHTML has two users with the same first name.
const ambiguousHTML = `
<lexxy-prompt-item search="Alex Smith AS" sgid="alex1-sgid">
  <template type="editor">
    <img title="Alex Smith" src="/123/users/a1/avatar" width="48" height="48" />
    Alex
  </template>
</lexxy-prompt-item>
<lexxy-prompt-item search="Alex Jones AJ" sgid="alex2-sgid">
  <template type="editor">
    <img title="Alex Jones" src="/123/users/a2/avatar" width="48" height="48" />
    Alex
  </template>
</lexxy-prompt-item>
`

func newMentionMockClient() *MockClient {
	m := NewMockClient()
	m.GetHTMLResponse = &client.APIResponse{
		StatusCode: 200,
		Body:       []byte(samplePromptUsersHTML),
	}
	return m
}

func TestParseMentionUsers(t *testing.T) {
	users := parseMentionUsers([]byte(samplePromptUsersHTML))

	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}

	tests := []struct {
		idx       int
		firstName string
		fullName  string
		sgid      string
	}{
		{0, "Wayne", "Wayne Smith", "wayne-sgid-123"},
		{1, "Bushra", "Bushra Gul", "bushra-sgid-456"},
		{2, "Kennedy", "Kennedy", "kennedy-sgid-789"},
	}

	for _, tt := range tests {
		u := users[tt.idx]
		if u.FirstName != tt.firstName {
			t.Errorf("user[%d] FirstName = %q, want %q", tt.idx, u.FirstName, tt.firstName)
		}
		if u.FullName != tt.fullName {
			t.Errorf("user[%d] FullName = %q, want %q", tt.idx, u.FullName, tt.fullName)
		}
		if u.SGID != tt.sgid {
			t.Errorf("user[%d] SGID = %q, want %q", tt.idx, u.SGID, tt.sgid)
		}
		if u.AvatarSrc == "" {
			t.Errorf("user[%d] AvatarSrc is empty", tt.idx)
		}
	}
}

func TestParseMentionUsersEmpty(t *testing.T) {
	users := parseMentionUsers([]byte(""))
	if len(users) != 0 {
		t.Errorf("expected 0 users from empty HTML, got %d", len(users))
	}
}

func TestParseMentionUsersAttributeOrder(t *testing.T) {
	// sgid before search — should still parse correctly
	html := `<lexxy-prompt-item sgid="reversed-sgid" search="Test User TU">
  <template type="editor">
    <img title="Test User" src="/123/users/t1/avatar" width="48" height="48" />
    Test
  </template>
</lexxy-prompt-item>`
	users := parseMentionUsers([]byte(html))
	if len(users) != 1 {
		t.Fatalf("expected 1 user from reversed attributes, got %d", len(users))
	}
	if users[0].FirstName != "Test" {
		t.Errorf("FirstName = %q, want %q", users[0].FirstName, "Test")
	}
	if users[0].SGID != "reversed-sgid" {
		t.Errorf("SGID = %q, want %q", users[0].SGID, "reversed-sgid")
	}
}

func TestParseMentionUsersAvatarScoping(t *testing.T) {
	// Each user's avatar should come from their own block, not a later one
	users := parseMentionUsers([]byte(samplePromptUsersHTML))
	if len(users) < 2 {
		t.Fatal("expected at least 2 users")
	}
	if !strings.Contains(users[0].AvatarSrc, "u1") {
		t.Errorf("user[0] avatar should contain u1, got %q", users[0].AvatarSrc)
	}
	if !strings.Contains(users[1].AvatarSrc, "u2") {
		t.Errorf("user[1] avatar should contain u2, got %q", users[1].AvatarSrc)
	}
}

func TestResolveMentions(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		shouldContain  []string
		shouldNotMatch []string // substrings that should NOT appear
	}{
		{
			name:           "no @ passthrough",
			input:          "Hello world",
			shouldContain:  []string{"Hello world"},
			shouldNotMatch: []string{"action-text-attachment"},
		},
		{
			name:          "single mention",
			input:         "Hey @Wayne check this",
			shouldContain: []string{`sgid="wayne-sgid-123"`, `content-type="application/vnd.actiontext.mention"`, `title="Wayne Smith"`, ">Wayne<"},
		},
		{
			name:          "case insensitive",
			input:         "Hey @wayne check this",
			shouldContain: []string{`sgid="wayne-sgid-123"`},
		},
		{
			name:          "multiple mentions",
			input:         "@Wayne and @Bushra please review",
			shouldContain: []string{`sgid="wayne-sgid-123"`, `sgid="bushra-sgid-456"`},
		},
		{
			name:           "email not treated as mention",
			input:          "Contact user@example.com",
			shouldContain:  []string{"user@example.com"},
			shouldNotMatch: []string{"action-text-attachment"},
		},
		{
			name:           "unresolved mention stays as text",
			input:          "Hey @Unknown person",
			shouldContain:  []string{"@Unknown"},
			shouldNotMatch: []string{"action-text-attachment"},
		},
		{
			name:          "mention at start of text",
			input:         "@Kennedy can you look?",
			shouldContain: []string{`sgid="kennedy-sgid-789"`},
		},
		{
			name:          "mention after newline",
			input:         "First line\n@Wayne second line",
			shouldContain: []string{`sgid="wayne-sgid-123"`},
		},
		{
			name:          "single name user",
			input:         "Hey @Kennedy",
			shouldContain: []string{`sgid="kennedy-sgid-789"`, `title="Kennedy"`},
		},
		{
			name:           "mention inside inline code not resolved",
			input:          "Use `@Wayne` to mention someone",
			shouldContain:  []string{"`@Wayne`"},
			shouldNotMatch: []string{"action-text-attachment"},
		},
		{
			name:           "mention inside fenced code block not resolved",
			input:          "Example:\n```\n@Wayne check this\n```",
			shouldContain:  []string{"@Wayne"},
			shouldNotMatch: []string{"action-text-attachment"},
		},
		{
			name:          "mention outside code resolved while code preserved",
			input:         "@Wayne see `@Bushra` example",
			shouldContain: []string{`sgid="wayne-sgid-123"`, "`@Bushra`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMentionCache()
			mock := newMentionMockClient()
			result := resolveMentions(tt.input, mock)

			for _, s := range tt.shouldContain {
				if !strings.Contains(result, s) {
					t.Errorf("result should contain %q\ngot: %s", s, result)
				}
			}
			for _, s := range tt.shouldNotMatch {
				if strings.Contains(result, s) {
					t.Errorf("result should NOT contain %q\ngot: %s", s, result)
				}
			}
		})
	}
}

func TestResolveMentionsNoFetchWithoutAt(t *testing.T) {
	resetMentionCache()
	mock := newMentionMockClient()

	resolveMentions("Hello world, no mentions here", mock)

	if len(mock.GetHTMLCalls) != 0 {
		t.Errorf("expected 0 GetHTML calls for text without @, got %d", len(mock.GetHTMLCalls))
	}
}

func TestResolveMentionsAmbiguous(t *testing.T) {
	resetMentionCache()
	mock := NewMockClient()
	mock.GetHTMLResponse = &client.APIResponse{
		StatusCode: 200,
		Body:       []byte(ambiguousHTML),
	}

	result := resolveMentions("Hey @Alex check this", mock)

	// Should NOT resolve — ambiguous
	if strings.Contains(result, "action-text-attachment") {
		t.Errorf("ambiguous mention should not resolve, got: %s", result)
	}
	if !strings.Contains(result, "@Alex") {
		t.Errorf("ambiguous mention should stay as @Alex, got: %s", result)
	}
}

func TestResolveMentionsAPIError(t *testing.T) {
	resetMentionCache()
	mock := NewMockClient()
	mock.GetHTMLError = fmt.Errorf("server error")

	// Should return text unchanged on error
	input := "Hey @Wayne"
	result := resolveMentions(input, mock)
	if result != input {
		t.Errorf("expected unchanged text on error, got: %s", result)
	}
}

func TestResolveMentionsCaching(t *testing.T) {
	resetMentionCache()
	mock := newMentionMockClient()

	// First call fetches
	resolveMentions("@Wayne", mock)
	if len(mock.GetHTMLCalls) != 1 {
		t.Errorf("expected 1 GetHTML call, got %d", len(mock.GetHTMLCalls))
	}

	// Second call uses cache
	resolveMentions("@Bushra", mock)
	if len(mock.GetHTMLCalls) != 1 {
		t.Errorf("expected still 1 GetHTML call (cached), got %d", len(mock.GetHTMLCalls))
	}
}

func TestBuildMentionHTMLEscaping(t *testing.T) {
	u := mentionUser{
		FirstName: `O'Brien`,
		FullName:  `O'Brien <script>`,
		SGID:      `sgid"test`,
		AvatarSrc: `/users/1/avatar?a=1&b=2`,
	}
	result := buildMentionHTML(u)

	if strings.Contains(result, `<script>`) {
		t.Error("HTML should be escaped, found raw <script>")
	}
	if !strings.Contains(result, `&lt;script&gt;`) {
		t.Error("expected escaped <script> tag")
	}
	if strings.Contains(result, `sgid"test`) {
		t.Error("SGID with quote should be escaped")
	}
}
