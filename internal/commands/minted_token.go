package commands

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
)

// A token `fizzy signup` created is the CLI's own: nobody pasted it, so it
// lives nowhere else, and a logout that only forgot it would leave a live
// write token behind on the server. Logout revokes those. A token handed to
// `fizzy auth login` is not the CLI's to revoke — the same token may be in a
// password manager, CI or another tool — so logout only forgets it.
//
// The profile records which token the CLI created by its SHA-256, never the
// token itself. A mark that outlives its token (a later login on the same
// profile through a path that doesn't reset it) matches nothing, so it can
// never revoke a pasted token.
const mintedTokenExtraKey = "minted_token_sha256" //nolint:gosec // G101: a metadata key, not a credential

// mintedTokenRevocationTimeout bounds the revocation request: a logout must
// not hang on a slow server, and it forgets the token locally regardless.
var mintedTokenRevocationTimeout = 10 * time.Second

// mintedTokenRevocationClient sends the revocation. It follows no redirect:
// the token goes to the Fizzy that minted it and nowhere else, and a redirect
// is reported as a failed revocation. Replaceable in tests.
var mintedTokenRevocationClient = func() *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func mintedTokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// setMintedTokenMark records token as CLI-created in a profile's extra
// metadata when minted, and drops any earlier mark when not.
func setMintedTokenMark(extra map[string]json.RawMessage, token string, minted bool) {
	if !minted || token == "" {
		delete(extra, mintedTokenExtraKey)
		return
	}
	extra[mintedTokenExtraKey], _ = json.Marshal(mintedTokenDigest(token))
}

// mintedTokenFor returns the profile's CLI-created token: the first of the
// stored candidates whose digest the profile recorded. Empty when the profile
// recorded none, or none of the candidates is it.
func mintedTokenFor(p *profile.Profile, candidates ...string) string {
	if p == nil || p.Extra == nil {
		return ""
	}
	var mark string
	if raw, ok := p.Extra[mintedTokenExtraKey]; !ok || json.Unmarshal(raw, &mark) != nil || mark == "" {
		return ""
	}
	for _, token := range candidates {
		if token != "" && subtle.ConstantTimeCompare([]byte(mintedTokenDigest(token)), []byte(mark)) == 1 {
			return token
		}
	}
	return ""
}

// storedTokenCandidates are the places a profile's token can be stored: its
// credential-store entry, and the global config file signup falls back to
// when the credential store is unavailable.
func storedTokenCandidates(profileName string) []string {
	var candidates []string
	if creds != nil {
		if token, err := credsLoadProfileToken(profileName); err == nil {
			candidates = append(candidates, token)
		}
	}
	return append(candidates, config.LoadGlobal().Token)
}

// mintedTokenRevocation is what became of a profile's CLI-created token.
type mintedTokenRevocation struct {
	// Attempted: the profile held a token the CLI created.
	Attempted bool
	// Err is why an attempted revocation did not go through.
	Err error
}

// revokeProfileMintedToken revokes the token the CLI created for p, if it
// still holds one, at the profile's Fizzy. Best effort: the caller forgets
// the token locally whatever happens here.
func revokeProfileMintedToken(name string, p *profile.Profile) mintedTokenRevocation {
	token := mintedTokenFor(p, storedTokenCandidates(name)...)
	if token == "" {
		return mintedTokenRevocation{}
	}
	baseURL := config.DefaultAPIURL
	if p.BaseURL != "" {
		baseURL = p.BaseURL
	}
	return mintedTokenRevocation{Attempted: true, Err: revokeToken(baseURL, token)}
}

// revokeToken asks Fizzy to revoke token (RFC 7009). Fizzy's revocation
// endpoint takes a personal access token on possession alone and answers 200
// whatever the token's state; any other answer is reported, never the token.
func revokeToken(baseURL, token string) error {
	endpoint := strings.TrimSuffix(baseURL, "/") + "/oauth/revocation"
	if err := validateSignupURL(endpoint); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), mintedTokenRevocationTimeout)
	defer cancel()
	form := url.Values{"token": {token}, "token_type_hint": {"access_token"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating revocation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("fizzy-cli/%s", cliVersion))

	resp, err := mintedTokenRevocationClient().Do(req)
	if err != nil {
		return fmt.Errorf("revocation request failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, signupMaxResponseBytes))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the revocation was answered with HTTP %d", resp.StatusCode)
	}
	return nil
}
