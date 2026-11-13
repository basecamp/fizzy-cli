package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

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

// ErrNoLocalConfig reports that there is no local config to act on.
var ErrNoLocalConfig = fmt.Errorf("no %s found here or in a parent directory", LocalConfigFile)

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
			return nil, ErrNoLocalConfig
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
	if err = yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("cannot trust %s: %w", path, err)
	}

	entry := TrustEntry{
		Path:      canon,
		TrustedAt: time.Now().UTC().Format(time.RFC3339),
		Digest:    trustDigest(file),
	}
	err = updateTrustFile(func(tf *trustFile) bool {
		for i := range tf.Trusted {
			if tf.Trusted[i].Path == canon {
				tf.Trusted[i] = entry
				return true
			}
		}
		tf.Trusted = append(tf.Trusted, entry)
		return true
	})
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// UntrustLocal removes trust from the local config at path, or the nearest one
// when path is empty. It reports whether the file was trusted.
func UntrustLocal(path string) (string, bool, error) {
	if path == "" {
		if path = findLocalConfig(); path == "" {
			return "", false, ErrNoLocalConfig
		}
	}
	// A file that is gone, or a link whose target is, can still be untrusted
	// by the paths it could have been trusted under.
	candidates := unresolvedPaths(path)
	if canon, err := canonicalPath(path); err == nil {
		candidates = []string{canon}
	}
	if len(candidates) == 0 {
		return "", false, fmt.Errorf("cannot resolve path %s", path)
	}
	shown := candidates[0]
	removed := false
	err := updateTrustFile(func(tf *trustFile) bool {
		kept := tf.Trusted[:0]
		for _, e := range tf.Trusted {
			if containsPath(candidates, e.Path) {
				removed = true
				shown = e.Path
				continue
			}
			kept = append(kept, e)
		}
		tf.Trusted = kept
		return removed
	})
	return shown, removed, err
}

// unresolvedPaths lists the canonical paths a local config that no longer
// resolves could have had: its own, with its directory's links resolved, and,
// for a dangling link, its target's.
func unresolvedPaths(path string) []string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil
	}
	paths := []string{abs}
	add := func(p string) {
		if dir, err := filepath.EvalSymlinks(filepath.Dir(p)); err == nil {
			p = filepath.Join(dir, filepath.Base(p))
		}
		if !containsPath(paths, p) {
			paths = append(paths, p)
		}
	}
	add(abs)
	if target, err := os.Readlink(abs); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}
		add(filepath.Clean(target))
	}
	return paths
}

func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

// DisplayPath is a path, or other text a local config or its location
// supplies, without control characters, for messages: a directory name can
// hold a newline, and a message must not let it forge output.
func DisplayPath(path string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, path)
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

// trustFilePath is the trust store, in the preferred config directory. It does
// not follow the global config file to a legacy directory: trust must not move
// when that file is created or removed.
func trustFilePath() (string, error) {
	paths := globalConfigPaths()
	if len(paths) == 0 {
		return "", errors.New("unable to determine config path")
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

// updateTrustFile applies change to the trust store and saves it when change
// reports a difference. A lock file serializes the read, the change and the
// save across processes, so one command cannot write back entries another
// just removed.
func updateTrustFile(change func(*trustFile) bool) error {
	path, err := trustFilePath()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	unlock, err := lockTrustFile(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()

	tf := loadTrustFile()
	if !change(&tf) {
		return nil
	}
	return saveTrustFile(path, tf)
}

const (
	trustLockWait  = 5 * time.Second
	trustLockStale = 30 * time.Second
)

// lockTrustFile takes the lock by creating lockPath, which only one process
// can do. A lock left by a process that died is taken over once it is stale.
func lockTrustFile(lockPath string) (func(), error) {
	deadline := time.Now().Add(trustLockWait)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("cannot lock the trust store: %w", err)
		}
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > trustLockStale {
			if os.Remove(lockPath) == nil {
				continue
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("the trust store is locked by another fizzy command (%s)", lockPath)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func saveTrustFile(path string, tf trustFile) error {
	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), trustFileName+".*.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err = tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
