package commands

import (
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/basecamp/fizzy-cli/internal/client"
)

// mentionUser represents a mentionable user parsed from the /prompts/users endpoint.
type mentionUser struct {
	FirstName string // e.g. "Wayne"
	FullName  string // e.g. "Wayne Smith"
	SGID      string // signed global ID for ActionText
	AvatarSrc string // e.g. "/6103476/users/03f5awg7.../avatar"
}

// Package-level cache: populated once per CLI invocation.
var (
	mentionUsers []mentionUser
	mentionOnce  sync.Once
	mentionErr   error
)

// resetMentionCache resets the cache for testing.
func resetMentionCache() {
	mentionOnce = sync.Once{}
	mentionUsers = nil
	mentionErr = nil
}

// mentionRegex matches @Name patterns not preceded by word characters or dots
// (to avoid matching emails like user@example.com).
// Supports Unicode letters and hyphens in names (e.g. @José, @Mary-Jane).
var mentionRegex = regexp.MustCompile(`(?:^|[^-\p{L}\p{N}_.])@([\p{L}][\p{L}\p{N}_-]*)`)

// promptItemRegex matches opening <lexxy-prompt-item> tags.
// Attributes are extracted separately to handle any order.
var promptItemRegex = regexp.MustCompile(`<lexxy-prompt-item\s[^>]*>`)

// searchAttrRegex extracts the search attribute value.
var searchAttrRegex = regexp.MustCompile(`\ssearch="([^"]+)"`)

// sgidAttrRegex extracts the sgid attribute value.
var sgidAttrRegex = regexp.MustCompile(`\ssgid="([^"]+)"`)

// promptItemEndRegex matches the closing tag for a prompt item block.
var promptItemEndRegex = regexp.MustCompile(`</lexxy-prompt-item>`)

// avatarRegex extracts the src attribute from the first <img> tag.
var avatarRegex = regexp.MustCompile(`<img[^>]+src="([^"]+)"`)

// codeBlockRegex matches fenced code blocks (``` ... ```).
var codeBlockRegex = regexp.MustCompile("(?s)```.*?```")

// codeSpanRegex matches inline code spans (` ... `).
var codeSpanRegex = regexp.MustCompile("`[^`]+`")

// resolveMentions scans text for @FirstName patterns and replaces them with
// ActionText mention HTML. If the text contains no @ characters, it is returned
// unchanged. On any error fetching users, the original text is returned with a
// warning printed to stderr.
//
// Mentions inside markdown code spans (`@name`) and fenced code blocks are not
// resolved, preserving the user's intended literal text.
func resolveMentions(text string, c client.API) string {
	if !strings.Contains(text, "@") {
		return text
	}

	mentionOnce.Do(func() {
		mentionUsers, mentionErr = fetchMentionUsers(c)
	})

	if mentionErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch mentionable users: %v\n", mentionErr)
		return text
	}

	if len(mentionUsers) == 0 {
		return text
	}

	// Protect code blocks and code spans from mention resolution by replacing
	// them with placeholders, resolving mentions, then restoring the originals.
	var codeChunks []string
	placeholder := func(s string) string {
		idx := len(codeChunks)
		codeChunks = append(codeChunks, s)
		return fmt.Sprintf("\x00CODE%d\x00", idx)
	}

	protected := codeBlockRegex.ReplaceAllStringFunc(text, placeholder)
	protected = codeSpanRegex.ReplaceAllStringFunc(protected, placeholder)

	// Find all @Name matches with positions
	type mentionMatch struct {
		start int // start of @Name (the @ character)
		end   int // end of @Name
		name  string
	}

	allMatches := mentionRegex.FindAllStringSubmatchIndex(protected, -1)
	var matches []mentionMatch
	for _, loc := range allMatches {
		// loc[2]:loc[3] is the capture group (the name without @)
		nameStart := loc[2]
		nameEnd := loc[3]
		// The @ is one character before the name
		atStart := nameStart - 1
		name := protected[nameStart:nameEnd]
		matches = append(matches, mentionMatch{start: atStart, end: nameEnd, name: name})
	}

	// Process from end to start so replacements don't shift indices
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]

		// Find matching user by first name (case-insensitive)
		var found []mentionUser
		for _, u := range mentionUsers {
			if strings.EqualFold(u.FirstName, m.name) {
				found = append(found, u)
			}
		}

		switch len(found) {
		case 1:
			mentionHTML := buildMentionHTML(found[0])
			protected = protected[:m.start] + mentionHTML + protected[m.end:]
		case 0:
			fmt.Fprintf(os.Stderr, "Warning: could not resolve mention @%s\n", m.name)
		default:
			names := make([]string, len(found))
			for j, u := range found {
				names[j] = u.FullName
			}
			fmt.Fprintf(os.Stderr, "Warning: ambiguous mention @%s — matches: %s\n", m.name, strings.Join(names, ", "))
		}
	}

	// Restore code blocks and spans
	for i, chunk := range codeChunks {
		protected = strings.Replace(protected, fmt.Sprintf("\x00CODE%d\x00", i), chunk, 1)
	}

	return protected
}

// fetchMentionUsers fetches the list of mentionable users from the API.
func fetchMentionUsers(c client.API) ([]mentionUser, error) {
	resp, err := c.GetHTML("/prompts/users")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d from /prompts/users", resp.StatusCode)
	}
	return parseMentionUsers(resp.Body), nil
}

// parseMentionUsers extracts mentionable users from the /prompts/users HTML.
// Each user is represented as a <lexxy-prompt-item> element with search and sgid
// attributes, containing <img> tags with avatar URLs.
func parseMentionUsers(htmlBytes []byte) []mentionUser {
	htmlStr := string(htmlBytes)
	items := promptItemRegex.FindAllStringIndex(htmlStr, -1)
	if len(items) == 0 {
		return nil
	}

	// Find all closing tags for scoping avatar lookups
	endIndices := promptItemEndRegex.FindAllStringIndex(htmlStr, -1)

	var users []mentionUser
	for itemIdx, loc := range items {
		tag := htmlStr[loc[0]:loc[1]]

		// Extract search and sgid attributes (order-independent)
		searchMatch := searchAttrRegex.FindStringSubmatch(tag)
		sgidMatch := sgidAttrRegex.FindStringSubmatch(tag)
		if searchMatch == nil || sgidMatch == nil {
			continue
		}

		search := strings.TrimSpace(searchMatch[1])
		sgid := sgidMatch[1]

		if search == "" || sgid == "" {
			continue
		}

		// Parse name from search attribute.
		// Format: "Full Name INITIALS [me]"
		// Strip trailing "me" and all-uppercase words (initials like "WS", "FMA").
		words := strings.Fields(search)
		for len(words) > 1 {
			last := words[len(words)-1]
			if last == "me" || isAllUpper(last) {
				words = words[:len(words)-1]
			} else {
				break
			}
		}

		fullName := strings.Join(words, " ")
		firstName := words[0]

		// Extract avatar URL scoped to this prompt-item block only.
		avatarSrc := ""
		blockStart := loc[0]
		blockEnd := len(htmlStr)
		if itemIdx < len(endIndices) {
			blockEnd = endIndices[itemIdx][1]
		}
		block := htmlStr[blockStart:blockEnd]
		if m := avatarRegex.FindStringSubmatch(block); len(m) > 1 {
			avatarSrc = m[1]
		}

		users = append(users, mentionUser{
			FirstName: firstName,
			FullName:  fullName,
			SGID:      sgid,
			AvatarSrc: avatarSrc,
		})
	}

	return users
}

// buildMentionHTML creates the ActionText attachment HTML for a mention.
// Values are HTML-escaped to prevent injection from user-controlled names.
func buildMentionHTML(u mentionUser) string {
	return fmt.Sprintf(
		`<action-text-attachment sgid="%s" content-type="application/vnd.actiontext.mention">`+
			`<img title="%s" src="%s" width="48" height="48">%s`+
			`</action-text-attachment>`,
		html.EscapeString(u.SGID),
		html.EscapeString(u.FullName),
		html.EscapeString(u.AvatarSrc),
		html.EscapeString(u.FirstName),
	)
}

// isAllUpper returns true if s is non-empty and all uppercase letters.
func isAllUpper(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}
