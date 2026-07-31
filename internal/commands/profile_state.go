package commands

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/zalando/go-keyring"
)

var accountIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]*$`)

type profileAccountBinding struct {
	Account  string
	Explicit bool
}

func resolveProfileAccountBinding(name string, p *profile.Profile) (profileAccountBinding, error) {
	if p == nil || p.Extra == nil {
		if err := validateAccountIdentifier(name); err != nil {
			return profileAccountBinding{}, fmt.Errorf("profile %q: %w", name, err)
		}
		return profileAccountBinding{Account: name}, nil
	}

	raw, present := p.Extra["account"]
	if !present {
		if err := validateAccountIdentifier(name); err != nil {
			return profileAccountBinding{}, fmt.Errorf("profile %q: %w", name, err)
		}
		return profileAccountBinding{Account: name}, nil
	}

	var account string
	if err := json.Unmarshal(raw, &account); err != nil || strings.TrimSpace(account) == "" {
		return profileAccountBinding{}, fmt.Errorf("profile %q has invalid account metadata: account must be a non-empty string", name)
	}
	account = strings.TrimSpace(account)
	if err := validateAccountIdentifier(account); err != nil {
		return profileAccountBinding{}, fmt.Errorf("profile %q has invalid account metadata: %w", name, err)
	}
	return profileAccountBinding{Account: account, Explicit: true}, nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func validateAccountIdentifier(account string) error {
	if !accountIdentifierPattern.MatchString(account) {
		return fmt.Errorf("invalid account %q: use a slug or ID containing only letters, numbers, periods, underscores, tildes, or hyphens", account)
	}
	return nil
}

func profileHasExplicitAccount(profileName string) bool {
	if profiles == nil || profileName == "" {
		return false
	}
	allProfiles, _, err := profiles.List()
	if err != nil {
		// A profile-store read failure makes legacy migration ambiguous.
		return true
	}
	p := allProfiles[profileName]
	if p == nil || p.Extra == nil {
		return false
	}
	_, present := p.Extra["account"]
	return present
}

type profileStateSnapshot struct {
	name            string
	previous        *profile.Profile
	previousDefault string
}

func snapshotProfileState(name string) (profileStateSnapshot, error) {
	snapshot := profileStateSnapshot{name: name}
	if profiles == nil {
		return snapshot, nil
	}
	allProfiles, defaultName, err := profiles.List()
	if err != nil {
		return profileStateSnapshot{}, err
	}
	snapshot.previous = allProfiles[name]
	snapshot.previousDefault = defaultName
	return snapshot, nil
}

func restoreProfileState(snapshot profileStateSnapshot) error {
	if profiles == nil {
		return nil
	}
	allProfiles, _, err := profiles.List()
	if err != nil {
		return err
	}
	if _, exists := allProfiles[snapshot.name]; exists {
		if err := profiles.Delete(snapshot.name); err != nil {
			return err
		}
	}
	if snapshot.previous != nil {
		if snapshot.previousDefault == "" && len(allProfiles) == 1 {
			temporaryName := "fizzy-restore"
			for suffix := 2; ; suffix++ {
				if _, exists := allProfiles[temporaryName]; !exists {
					break
				}
				temporaryName = fmt.Sprintf("fizzy-restore-%d", suffix)
			}
			if err := profiles.Create(&profile.Profile{Name: temporaryName, BaseURL: config.DefaultAPIURL}); err != nil {
				return err
			}
			if err := profiles.Create(snapshot.previous); err != nil {
				return err
			}
			return profiles.Delete(temporaryName)
		}
		if err := profiles.Create(snapshot.previous); err != nil {
			return err
		}
	}
	if snapshot.previousDefault != "" {
		return profiles.SetDefault(snapshot.previousDefault)
	}
	return nil
}

type credentialStateSnapshot struct {
	profileName string
	data        []byte
	exists      bool
}

func snapshotProfileCredential(profileName string) (credentialStateSnapshot, error) {
	snapshot := credentialStateSnapshot{profileName: profileName}
	if creds == nil {
		return snapshot, nil
	}
	data, err := creds.Load(profile.CredentialKey(profileName, ""))
	if err != nil {
		if isCredentialNotFound(err) {
			return snapshot, nil
		}
		return credentialStateSnapshot{}, err
	}
	snapshot.data = append([]byte(nil), data...)
	snapshot.exists = true
	return snapshot, nil
}

func restoreProfileCredential(snapshot credentialStateSnapshot) error {
	if creds == nil {
		return nil
	}
	key := profile.CredentialKey(snapshot.profileName, "")
	if snapshot.exists {
		return creds.Save(key, snapshot.data)
	}
	if err := creds.Delete(key); err != nil && !isCredentialNotFound(err) {
		return err
	}
	return nil
}

func isCredentialNotFound(err error) bool {
	if err == nil {
		return false
	}
	return stderrors.Is(err, keyring.ErrNotFound) || strings.Contains(err.Error(), "credentials not found for ")
}

type globalConfigSnapshot struct {
	config *config.Config
	exists bool
}

func snapshotGlobalConfig() globalConfigSnapshot {
	loaded := config.LoadGlobal()
	configCopy := *loaded
	return globalConfigSnapshot{config: &configCopy, exists: config.Exists()}
}

func restoreGlobalConfig(snapshot globalConfigSnapshot) error {
	if !snapshot.exists {
		return config.Delete()
	}
	return snapshot.config.Save()
}

type profileCredentialSaveOptions struct {
	ProfileName            string
	Account                string
	BaseURL                string
	Board                  *string
	Token                  string
	AllowYAMLTokenFallback bool
	UpdateGlobal           func(*config.Config, bool)
}

func saveProfileCredentialState(opts profileCredentialSaveOptions) (error, error) {
	var warning error
	if err := profile.ValidateName(opts.ProfileName); err != nil {
		return nil, err
	}
	if err := validateAccountIdentifier(opts.Account); err != nil {
		return nil, err
	}

	profileSnapshot, err := snapshotProfileState(opts.ProfileName)
	if err != nil {
		return nil, err
	}
	credentialSnapshot, err := snapshotProfileCredential(opts.ProfileName)
	if err != nil {
		return nil, err
	}
	globalSnapshot := snapshotGlobalConfig()

	rollback := func(operationErr error, restoreCredential, restoreGlobal bool) error {
		rollbackErrors := []error{operationErr}
		if restoreCredential {
			if restoreErr := restoreProfileCredential(credentialSnapshot); restoreErr != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore credential: %w", restoreErr))
			}
		}
		if restoreErr := restoreProfileState(profileSnapshot); restoreErr != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restore profile: %w", restoreErr))
		}
		if restoreGlobal {
			if restoreErr := restoreGlobalConfig(globalSnapshot); restoreErr != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore global config: %w", restoreErr))
			}
		}
		return stderrors.Join(rollbackErrors...)
	}

	if err := ensureProfileForAccountWithBoard(opts.ProfileName, opts.Account, opts.BaseURL, opts.Board); err != nil {
		return nil, err
	}
	if profiles != nil {
		if err := profiles.SetDefault(opts.ProfileName); err != nil {
			return nil, rollback(err, false, false)
		}
	}

	credentialStored := false
	if creds != nil {
		if err := credsSaveProfileToken(opts.ProfileName, opts.Token); err != nil {
			if opts.AllowYAMLTokenFallback && !credentialSnapshot.exists {
				// With no prior profile credential, the YAML fallback remains
				// unambiguous and can safely carry the new token.
				warning = err
			} else {
				if restoreErr := restoreProfileCredential(credentialSnapshot); restoreErr != nil {
					return nil, rollback(stderrors.Join(err, fmt.Errorf("restore credential: %w", restoreErr)), false, false)
				}
				return nil, rollback(err, false, false)
			}
		} else {
			credentialStored = true
		}
	}

	globalCfg := config.LoadGlobal()
	if opts.UpdateGlobal != nil {
		opts.UpdateGlobal(globalCfg, credentialStored)
	}
	if err := globalCfg.Save(); err != nil {
		return warning, rollback(err, credentialStored, true)
	}
	return warning, nil
}
