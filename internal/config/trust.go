package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// trustFileName is the trust store's file, kept beside the global config.
const trustFileName = "trusted-configs.json"

// TrustEntry records one trusted local config: its path, when it was trusted,
// and a digest of the trust-gated settings it held then.
type TrustEntry struct {
	Path      string `json:"path"`
	TrustedAt string `json:"trusted_at"`
	Digest    string `json:"digest"`
}

type trustFile struct {
	Trusted []TrustEntry `json:"trusted"`
}

// LocalConfig is the nearest .fizzy.yaml and what applies from it. A local
// config sets the account and the board whoever wrote it. Its api_url and
// token decide where a credential is sent and which one, so they apply only
// while the file is trusted: a checkout or pull that changes either leaves the
// file untrusted until it is trusted again.
type LocalConfig struct {
	Path string
	// Config holds the settings that apply: without trust, APIURL and Token
	// are empty.
	Config Config
	// Gated reports whether the file sets api_url or token.
	Gated bool
	// Trusted reports whether those settings are trusted as they are now.
	Trusted bool
	// Ignored names the gated keys the file sets that were not applied.
	Ignored []string
}

// LoadLocal reads the nearest local config, or returns nil when there is none
// or it cannot be read as one.
func LoadLocal() *LocalConfig {
	path := findLocalConfig()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var file Config
	if yaml.Unmarshal(data, &file) != nil {
		return nil
	}

	local := &LocalConfig{Path: path, Config: file}
	local.Gated = file.APIURL != "" || file.Token != ""
	if !local.Gated {
		return local
	}
	local.Trusted = isTrusted(path, file)
	if !local.Trusted {
		if file.APIURL != "" {
			local.Ignored = append(local.Ignored, "api_url")
		}
		if file.Token != "" {
			local.Ignored = append(local.Ignored, "token")
		}
		local.Config.APIURL = ""
		local.Config.Token = ""
	}
	return local
}

// TrustLocal trusts the local config at path, or the nearest one when path is
// empty, with the api_url and token it holds now.
func TrustLocal(path string) (*TrustEntry, error) {
	if path == "" {
		if path = findLocalConfig(); path == "" {
			return nil, fmt.Errorf("no %s found here or in a parent directory", LocalConfigFile)
		}
	}
	canon, err := canonicalPath(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(canon)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}
	var file Config
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("cannot trust %s: %w", path, err)
	}

	entry := TrustEntry{
		Path:      canon,
		TrustedAt: time.Now().UTC().Format(time.RFC3339),
		Digest:    trustDigest(file),
	}
	tf := loadTrustFile()
	found := false
	for i := range tf.Trusted {
		if tf.Trusted[i].Path == canon {
			tf.Trusted[i] = entry
			found = true
			break
		}
	}
	if !found {
		tf.Trusted = append(tf.Trusted, entry)
	}
	if err := saveTrustFile(tf); err != nil {
		return nil, err
	}
	return &entry, nil
}

// UntrustLocal removes trust from the local config at path, or the nearest one
// when path is empty. It reports whether the file was trusted.
func UntrustLocal(path string) (string, bool, error) {
	if path == "" {
		if path = findLocalConfig(); path == "" {
			return "", false, fmt.Errorf("no %s found here or in a parent directory", LocalConfigFile)
		}
	}
	canon, err := canonicalPath(path)
	if err != nil {
		// A file that is gone can still be untrusted by the path it had.
		if canon, err = filepath.Abs(path); err != nil {
			return "", false, err
		}
	}
	tf := loadTrustFile()
	kept := tf.Trusted[:0]
	removed := false
	for _, e := range tf.Trusted {
		if e.Path == canon {
			removed = true
			continue
		}
		kept = append(kept, e)
	}
	if !removed {
		return canon, false, nil
	}
	tf.Trusted = kept
	return canon, true, saveTrustFile(tf)
}

// TrustedLocals lists the trusted local configs.
func TrustedLocals() []TrustEntry {
	return loadTrustFile().Trusted
}

func isTrusted(path string, file Config) bool {
	canon, err := canonicalPath(path)
	if err != nil {
		return false
	}
	digest := trustDigest(file)
	for _, e := range loadTrustFile().Trusted {
		if e.Path == canon {
			return e.Digest == digest
		}
	}
	return false
}

// trustDigest is the SHA-256 of a local config's trust-gated settings. A
// setting that trust gates must be part of it, or changing that setting would
// keep the trust.
func trustDigest(file Config) string {
	gated, _ := json.Marshal(struct {
		APIURL string `json:"api_url"`
		Token  string `json:"token"`
	}{file.APIURL, file.Token})
	sum := sha256.Sum256(gated)
	return hex.EncodeToString(sum[:])
}

func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %s: %w", path, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path %s: %w", path, err)
	}
	return filepath.Clean(resolved), nil
}

// trustFilePath is the trust store beside the global config in use, or beside
// the preferred one when there is none yet.
func trustFilePath() (string, error) {
	paths := globalConfigPaths()
	if len(paths) == 0 {
		return "", errors.New("unable to determine config path")
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return filepath.Join(filepath.Dir(path), trustFileName), nil
		}
	}
	return filepath.Join(filepath.Dir(paths[0]), trustFileName), nil
}

func loadTrustFile() trustFile {
	var tf trustFile
	path, err := trustFilePath()
	if err != nil {
		return tf
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return tf
	}
	if json.Unmarshal(data, &tf) != nil {
		return trustFile{}
	}
	return tf
}

func saveTrustFile(tf trustFile) error {
	path, err := trustFilePath()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
