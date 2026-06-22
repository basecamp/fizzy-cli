package commands

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/harness"
	fizzy "github.com/basecamp/fizzy-sdk/go/pkg/fizzy"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// DoctorCheck represents a single diagnostic check result.
type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pass, warn, fail, skip
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// DoctorProfileResult holds diagnostic results for a single profile.
type DoctorProfileResult struct {
	Name    string        `json:"name"`
	Default bool          `json:"default,omitempty"`
	Checks  []DoctorCheck `json:"checks"`
	Passed  int           `json:"passed"`
	Failed  int           `json:"failed"`
	Warned  int           `json:"warned"`
	Skipped int           `json:"skipped"`
}

// DoctorResult holds the complete diagnostic results.
type DoctorResult struct {
	Checks   []DoctorCheck         `json:"checks"`
	Profiles []DoctorProfileResult `json:"profiles,omitempty"`
	Passed   int                   `json:"passed"`
	Failed   int                   `json:"failed"`
	Warned   int                   `json:"warned"`
	Skipped  int                   `json:"skipped"`
}

func (r *DoctorResult) Summary() string {
	if r.Failed == 0 && r.Warned == 0 && r.Passed > 0 {
		if r.Skipped > 0 {
			return fmt.Sprintf("All %d checks passed, %d skipped", r.Passed, r.Skipped)
		}
		return fmt.Sprintf("All %d checks passed", r.Passed)
	}
	parts := []string{}
	if r.Passed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", r.Passed))
	}
	if r.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", r.Failed))
	}
	if r.Warned > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", r.Warned, pluralize(r.Warned, "warning", "warnings")))
	}
	if r.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", r.Skipped))
	}
	return strings.Join(parts, ", ")
}

type doctorEffectiveConfig struct {
	ProfileName    string
	Default        bool
	ProfileSource  string
	APIURL         string
	APIURLSource   string
	Board          string
	BoardSource    string
	Token          string
	TokenSource    string
	TokenSourceRaw string
}

var doctorVersionChecker = fetchLatestDoctorVersion

func NewDoctorCmd() *cobra.Command {
	var verbose bool
	var allProfiles bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check CLI health and diagnose issues",
		Long: `Run diagnostic checks on installation health, configuration, authentication, API connectivity,
board access, shell integration, and coding-agent setup.

Use --profile NAME to check a specific saved profile, or --all-profiles to sweep every saved
profile in addition to the global install checks.

Examples:
  fizzy doctor
  fizzy doctor --profile acme
  fizzy doctor --all-profiles
  fizzy doctor --verbose
  fizzy doctor --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allProfiles && cfgProfile != "" {
				return errors.NewInvalidArgsError("--all-profiles cannot be used with --profile")
			}
			if allProfiles && cfgToken != "" {
				return errors.NewInvalidArgsError("--all-profiles cannot be used with --token")
			}
			if allProfiles && cfgAPIURL != "" {
				return errors.NewInvalidArgsError("--all-profiles cannot be used with --api-url")
			}

			result := runDoctor(cmd.Context(), verbose, allProfiles)
			breadcrumbs := buildDoctorBreadcrumbs(flattenDoctorChecks(result))

			switch out.EffectiveFormat() {
			case output.FormatStyled:
				renderDoctorStyled(outWriter, result, breadcrumbs)
				captureResponse()
				return nil
			case output.FormatMarkdown:
				renderDoctorMarkdown(outWriter, result, breadcrumbs)
				captureResponse()
				return nil
			default:
				opts := []output.ResponseOption{output.WithSummary(result.Summary())}
				if len(breadcrumbs) > 0 {
					opts = append(opts, output.WithBreadcrumbs(breadcrumbs...))
				}
				recordOutputError(out.OK(result, opts...))
				captureResponse()
				return nil
			}
		},
	}

	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show additional diagnostic detail")
	cmd.Flags().BoolVar(&allProfiles, "all-profiles", false, "Run profile health checks for every saved profile")
	return cmd
}

func init() {
	rootCmd.AddCommand(NewDoctorCmd())
}

func runDoctor(ctx context.Context, verbose, allProfiles bool) *DoctorResult {
	result := &DoctorResult{Checks: runDoctorGlobalChecks(verbose)}
	if allProfiles {
		for _, target := range doctorTargetsFromProfileStore() {
			profileResult := DoctorProfileResult{
				Name:    target.ProfileName,
				Default: target.Default,
				Checks:  runDoctorTargetChecks(ctx, target, verbose),
			}
			summarizeDoctorProfile(&profileResult)
			result.Profiles = append(result.Profiles, profileResult)
		}
		if len(result.Profiles) == 0 {
			result.Checks = append(result.Checks, DoctorCheck{
				Name:    "Profile Sweep",
				Status:  "skip",
				Message: "No saved profiles to check",
				Hint:    "Run: fizzy auth login <token>",
			})
		}
	} else {
		result.Checks = append(result.Checks, runDoctorTargetChecks(ctx, resolveDoctorEffectiveConfig(), verbose)...)
	}
	summarizeDoctorResult(result)
	return result
}

func runDoctorGlobalChecks(verbose bool) []DoctorCheck {
	checks := []DoctorCheck{checkDoctorVersion(verbose)}
	if verbose {
		checks = append(checks, checkDoctorRuntime())
	}
	checks = append(checks,
		checkDoctorGlobalConfig(verbose),
		checkDoctorLocalConfig(verbose),
		checkDoctorProfileStore(verbose),
		checkDoctorSavedProfiles(),
		checkDoctorFilesystem(verbose),
		checkDoctorShellCompletion(verbose),
		checkDoctorSkillInstallation(),
	)
	if baselineSkillInstalled() {
		checks = append(checks, checkDoctorSkillVersion())
	}
	for _, agent := range harness.DetectedAgents() {
		if agent.Checks == nil {
			continue
		}
		for _, c := range agent.Checks() {
			status := c.Status
			if status == "fail" && isOptionalAgentCheck(c.Name) {
				status = "warn"
			}
			checks = append(checks, DoctorCheck{Name: c.Name, Status: status, Message: c.Message, Hint: c.Hint})
		}
	}
	return checks
}

func runDoctorTargetChecks(ctx context.Context, eff doctorEffectiveConfig, verbose bool) []DoctorCheck {
	checks := []DoctorCheck{checkDoctorEffectiveConfig(eff, verbose)}
	credCheck := checkDoctorCredentials(eff, verbose)
	checks = append(checks,
		credCheck,
		checkDoctorCredentialStorage(eff, verbose),
		checkDoctorLegacyState(eff),
		checkDoctorAPIURL(eff, verbose),
	)

	reachabilityCheck := checkDoctorAPIReachability(ctx, eff, verbose)
	checks = append(checks, reachabilityCheck)

	canAuth := credCheck.Status != "fail" && reachabilityCheck.Status == "pass"
	if canAuth {
		authCheck := checkDoctorAuthentication(ctx, eff, verbose)
		checks = append(checks, authCheck)
		if authCheck.Status == "pass" {
			checks = append(checks, checkDoctorAccountAccess(ctx, eff, verbose))
			checks = append(checks, checkDoctorBoardAccess(ctx, eff, verbose))
		} else {
			checks = append(checks,
				DoctorCheck{Name: "Account Access", Status: "skip", Message: "Skipped (authentication failed)"},
				DoctorCheck{Name: "Default Board", Status: "skip", Message: "Skipped (authentication failed)"},
			)
		}
	} else {
		authMsg := "Skipped (missing credentials or API unreachable)"
		checks = append(checks,
			DoctorCheck{Name: "Authentication", Status: "skip", Message: authMsg},
			DoctorCheck{Name: "Account Access", Status: "skip", Message: authMsg},
			DoctorCheck{Name: "Default Board", Status: "skip", Message: authMsg},
		)
	}
	return checks
}

func summarizeDoctorProfile(result *DoctorProfileResult) {
	for _, c := range result.Checks {
		switch c.Status {
		case "pass":
			result.Passed++
		case "fail":
			result.Failed++
		case "warn":
			result.Warned++
		case "skip":
			result.Skipped++
		}
	}
}

func summarizeDoctorResult(result *DoctorResult) {
	result.Passed, result.Failed, result.Warned, result.Skipped = 0, 0, 0, 0
	for _, c := range result.Checks {
		switch c.Status {
		case "pass":
			result.Passed++
		case "fail":
			result.Failed++
		case "warn":
			result.Warned++
		case "skip":
			result.Skipped++
		}
	}
	for i := range result.Profiles {
		for _, c := range result.Profiles[i].Checks {
			switch c.Status {
			case "pass":
				result.Passed++
			case "fail":
				result.Failed++
			case "warn":
				result.Warned++
			case "skip":
				result.Skipped++
			}
		}
	}
}

func flattenDoctorChecks(result *DoctorResult) []DoctorCheck {
	checks := append([]DoctorCheck{}, result.Checks...)
	for _, profileResult := range result.Profiles {
		checks = append(checks, profileResult.Checks...)
	}
	return checks
}

func checkDoctorVersion(verbose bool) DoctorCheck {
	check := DoctorCheck{Name: "CLI Version", Status: "pass", Message: currentVersion()}
	if verbose {
		if exe, err := os.Executable(); err == nil {
			check.Message = fmt.Sprintf("%s (%s)", currentVersion(), exe)
		}
	}

	current := currentVersion()
	if current == "dev" || strings.Contains(current, "-g") || strings.Contains(current, "dirty") || !strings.HasPrefix(current, "v") {
		return check
	}

	latest, err := doctorVersionChecker()
	if err == nil && latest != "" && latest != current {
		check.Status = "warn"
		check.Message = fmt.Sprintf("%s (latest: %s)", current, latest)
		check.Hint = "Download the latest release from https://github.com/basecamp/fizzy-cli/releases/latest"
	}
	return check
}

func fetchLatestDoctorVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/basecamp/fizzy-cli/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimSpace(release.TagName), nil
}

func checkDoctorRuntime() DoctorCheck {
	return DoctorCheck{
		Name:    "Runtime",
		Status:  "pass",
		Message: fmt.Sprintf("Go %s (%s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}
}

func checkDoctorGlobalConfig(verbose bool) DoctorCheck {
	paths := config.GlobalConfigPaths()
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return validateDoctorConfigFile(path, "Global Config", verbose)
		}
	}
	pathHint := "~/.config/fizzy/config.yaml"
	if len(paths) > 0 {
		pathHint = paths[0]
	}
	return DoctorCheck{
		Name:    "Global Config",
		Status:  "warn",
		Message: "Not found (using defaults)",
		Hint:    fmt.Sprintf("Create %s or run: fizzy setup", pathHint),
	}
}

func checkDoctorLocalConfig(verbose bool) DoctorCheck {
	path := config.LocalConfigPath()
	if path == "" {
		return DoctorCheck{Name: "Local Config", Status: "skip", Message: "Not found"}
	}
	return validateDoctorConfigFile(path, "Local Config", verbose)
}

func validateDoctorConfigFile(path, name string, verbose bool) DoctorCheck {
	data, err := os.ReadFile(path)
	if err != nil {
		return DoctorCheck{
			Name:    name,
			Status:  "fail",
			Message: fmt.Sprintf("Cannot read %s", path),
			Hint:    err.Error(),
		}
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return DoctorCheck{
			Name:    name,
			Status:  "fail",
			Message: fmt.Sprintf("Invalid YAML: %s", path),
			Hint:    err.Error(),
		}
	}
	msg := path
	if verbose {
		msg = fmt.Sprintf("%s (%d keys)", path, len(parsed))
	}
	return DoctorCheck{Name: name, Status: "pass", Message: msg}
}

func checkDoctorProfileStore(verbose bool) DoctorCheck {
	if profiles == nil {
		return DoctorCheck{
			Name:    "Profile Store",
			Status:  "warn",
			Message: "Profile store unavailable",
			Hint:    "Run: fizzy setup or fizzy auth login <token>",
		}
	}
	allProfiles, _, err := profiles.List()
	if err != nil {
		return DoctorCheck{
			Name:    "Profile Store",
			Status:  "fail",
			Message: "Cannot read profile store",
			Hint:    err.Error(),
		}
	}
	if len(allProfiles) == 0 {
		return DoctorCheck{
			Name:    "Profile Store",
			Status:  "warn",
			Message: "No named profiles configured",
			Hint:    "Run: fizzy setup or fizzy auth login <token>",
		}
	}
	msg := fmt.Sprintf("%d profile(s) configured", len(allProfiles))
	if verbose {
		msg += " [config.json]"
	}
	return DoctorCheck{
		Name:    "Profile Store",
		Status:  "pass",
		Message: msg,
		Hint:    "Use: fizzy doctor --profile NAME or fizzy doctor --all-profiles",
	}
}

func resolveDoctorEffectiveConfig() doctorEffectiveConfig {
	eff := doctorEffectiveConfig{
		ProfileName: cfg.Account,
		APIURL:      cfg.APIURL,
		Board:       cfg.Board,
		Token:       cfg.Token,
	}

	globalCfg, _ := loadDoctorConfigFile(globalConfigPathForDoctor())
	localCfg, _ := loadDoctorConfigFile(config.LocalConfigPath())
	resolvedProfile, profileCfg := resolveDoctorProfileContext()

	switch {
	case cfgProfile != "":
		eff.ProfileSource = "flag --profile"
	case os.Getenv("FIZZY_PROFILE") != "":
		eff.ProfileSource = "env FIZZY_PROFILE"
	case os.Getenv("FIZZY_ACCOUNT") != "":
		eff.ProfileSource = "env FIZZY_ACCOUNT"
	case resolvedProfile != "":
		eff.ProfileSource = "profile store"
	case localCfg != nil && localCfg.Account != "":
		eff.ProfileSource = "local config"
	case globalCfg != nil && globalCfg.Account != "":
		eff.ProfileSource = "global config"
	default:
		eff.ProfileSource = "unset"
	}

	switch {
	case cfgAPIURL != "":
		eff.APIURLSource = "flag --api-url"
	case os.Getenv("FIZZY_API_URL") != "":
		eff.APIURLSource = "env FIZZY_API_URL"
	case profileCfg != nil && profileCfg.BaseURL != "":
		eff.APIURLSource = "profile store"
	case localCfg != nil && localCfg.APIURL != "":
		eff.APIURLSource = "local config"
	case globalCfg != nil && globalCfg.APIURL != "":
		eff.APIURLSource = "global config"
	default:
		eff.APIURLSource = "default"
	}

	switch {
	case os.Getenv("FIZZY_BOARD") != "":
		eff.BoardSource = "env FIZZY_BOARD"
	case doctorProfileBoard(profileCfg) != "":
		eff.BoardSource = "profile store"
	case localCfg != nil && localCfg.Board != "":
		eff.BoardSource = "local config"
	case globalCfg != nil && globalCfg.Board != "":
		eff.BoardSource = "global config"
	default:
		eff.BoardSource = "unset"
	}

	eff.TokenSourceRaw, eff.TokenSource, eff.Token = doctorTokenSourceWithValue(cfg.Account, localCfg, globalCfg)
	return eff
}

func checkDoctorEffectiveConfig(eff doctorEffectiveConfig, verbose bool) DoctorCheck {
	parts := []string{}
	if eff.ProfileName != "" {
		if verbose {
			parts = append(parts, fmt.Sprintf("profile=%s [%s]", eff.ProfileName, eff.ProfileSource))
		} else {
			parts = append(parts, fmt.Sprintf("profile=%s", eff.ProfileName))
		}
	} else if verbose {
		parts = append(parts, "profile=<unset>")
	}
	if eff.APIURL != "" {
		if verbose {
			parts = append(parts, fmt.Sprintf("api_url=%s [%s]", eff.APIURL, eff.APIURLSource))
		} else {
			parts = append(parts, fmt.Sprintf("api_url=%s", eff.APIURL))
		}
	}
	if eff.Board != "" {
		if verbose {
			parts = append(parts, fmt.Sprintf("board=%s [%s]", eff.Board, eff.BoardSource))
		} else {
			parts = append(parts, fmt.Sprintf("board=%s", eff.Board))
		}
	} else if verbose {
		parts = append(parts, "board=<unset>")
	}
	if eff.TokenSource != "" {
		if verbose {
			parts = append(parts, fmt.Sprintf("token=%s [%s]", eff.TokenSource, eff.TokenSourceRaw))
		} else {
			parts = append(parts, fmt.Sprintf("token=%s", eff.TokenSource))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "No effective configuration")
	}
	return DoctorCheck{Name: "Effective Config", Status: "pass", Message: strings.Join(parts, ", ")}
}

func checkDoctorCredentials(eff doctorEffectiveConfig, _ bool) DoctorCheck {
	if strings.TrimSpace(eff.Token) == "" || eff.TokenSourceRaw == "" || eff.TokenSourceRaw == "none" {
		return DoctorCheck{
			Name:    "Credentials",
			Status:  "fail",
			Message: "No credentials found",
			Hint:    doctorLoginHint(eff.ProfileName),
		}
	}
	return DoctorCheck{
		Name:    "Credentials",
		Status:  "pass",
		Message: eff.TokenSource,
	}
}

func checkDoctorCredentialStorage(eff doctorEffectiveConfig, _ bool) DoctorCheck {
	switch eff.TokenSourceRaw {
	case "local-config":
		return DoctorCheck{
			Name:    "Credential Storage",
			Status:  "warn",
			Message: "Token is stored in local project config",
			Hint:    "Move credentials to the keyring with: fizzy auth login <token>",
		}
	case "global-config":
		return DoctorCheck{
			Name:    "Credential Storage",
			Status:  "warn",
			Message: "Token is stored in global config",
			Hint:    "Prefer the system keyring: fizzy auth login <token>",
		}
	case "legacy-keyring", "legacy-fallback":
		hint := "Refresh credentials with: fizzy auth login <token>"
		if eff.TokenSourceRaw == "legacy-fallback" && creds != nil {
			if warning := creds.FallbackWarning(); warning != "" {
				hint = warning + "; then refresh credentials with: fizzy auth login <token>"
			}
		}
		return DoctorCheck{
			Name:    "Credential Storage",
			Status:  "warn",
			Message: "Using legacy credential key",
			Hint:    hint,
		}
	case "fallback-file":
		hint := "System keyring unavailable; credentials are stored in a fallback file"
		if creds != nil {
			if warning := creds.FallbackWarning(); warning != "" {
				hint = warning
			}
		}
		return DoctorCheck{
			Name:    "Credential Storage",
			Status:  "warn",
			Message: "Using fallback credential file",
			Hint:    hint,
		}
	case "none":
		return DoctorCheck{Name: "Credential Storage", Status: "skip", Message: "Skipped (no credentials)"}
	default:
		return DoctorCheck{Name: "Credential Storage", Status: "pass", Message: "No storage issues detected"}
	}
}

func checkDoctorLegacyState(eff doctorEffectiveConfig) DoctorCheck {
	if os.Getenv("FIZZY_ACCOUNT") != "" {
		return DoctorCheck{
			Name:    "Legacy Environment",
			Status:  "warn",
			Message: "Using deprecated FIZZY_ACCOUNT environment variable",
			Hint:    "Use FIZZY_PROFILE instead",
		}
	}
	if eff.TokenSourceRaw == "legacy-keyring" || eff.TokenSourceRaw == "legacy-fallback" {
		return DoctorCheck{
			Name:    "Legacy Environment",
			Status:  "warn",
			Message: "Using legacy credential format",
			Hint:    "Run: fizzy auth login <token>",
		}
	}
	return DoctorCheck{Name: "Legacy Environment", Status: "pass", Message: "No legacy compatibility issues detected"}
}

func checkDoctorAPIURL(eff doctorEffectiveConfig, _ bool) DoctorCheck {
	if err := validateAPIURL(eff.APIURL); err != nil {
		return DoctorCheck{
			Name:    "API URL",
			Status:  "fail",
			Message: fmt.Sprintf("Invalid API URL: %s", eff.APIURL),
			Hint:    fmt.Sprintf("%v. Fix --api-url, FIZZY_API_URL, profile base_url, or run: fizzy setup", err),
		}
	}
	if strings.HasPrefix(eff.APIURL, "http://") {
		return DoctorCheck{
			Name:    "API URL",
			Status:  "warn",
			Message: fmt.Sprintf("Using insecure HTTP URL: %s", eff.APIURL),
			Hint:    "Use HTTPS outside localhost development",
		}
	}
	return DoctorCheck{Name: "API URL", Status: "pass", Message: eff.APIURL}
}

func checkDoctorAPIReachability(ctx context.Context, eff doctorEffectiveConfig, verbose bool) DoctorCheck {
	if eff.APIURL == "" {
		return DoctorCheck{Name: "API Reachability", Status: "fail", Message: "No API URL configured", Hint: "Run: fizzy setup"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eff.APIURL, nil)
	if err != nil {
		return DoctorCheck{Name: "API Reachability", Status: "fail", Message: "Cannot build API request", Hint: err.Error()}
	}
	start := time.Now()
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return DoctorCheck{Name: "API Reachability", Status: "fail", Message: "Cannot reach API host", Hint: err.Error()}
	}
	defer resp.Body.Close()
	latency := time.Since(start)
	if resp.StatusCode >= 500 {
		return DoctorCheck{
			Name:    "API Reachability",
			Status:  "warn",
			Message: fmt.Sprintf("Host reachable but returned %d", resp.StatusCode),
			Hint:    "Check service status and try again",
		}
	}
	msg := "API host reachable"
	if verbose {
		msg = fmt.Sprintf("API host reachable (%d, %dms)", resp.StatusCode, latency.Milliseconds())
	}
	return DoctorCheck{Name: "API Reachability", Status: "pass", Message: msg}
}

func checkDoctorAuthentication(ctx context.Context, eff doctorEffectiveConfig, verbose bool) DoctorCheck {
	client, _, err := newDoctorClients(eff)
	if err != nil {
		return DoctorCheck{Name: "Authentication", Status: "fail", Message: "SDK initialization failed", Hint: err.Error()}
	}
	start := time.Now()
	identity, _, err := client.Identity().GetMyIdentity(ctx)
	if err != nil {
		conv := convertSDKError(err)
		var outErr *output.Error
		if stderrors.As(conv, &outErr) {
			return DoctorCheck{Name: "Authentication", Status: "fail", Message: outErr.Message, Hint: firstNonEmpty(outErr.Hint, doctorLoginHint(eff.ProfileName))}
		}
		return DoctorCheck{Name: "Authentication", Status: "fail", Message: err.Error(), Hint: doctorLoginHint(eff.ProfileName)}
	}
	msg := "Token accepted"
	if verbose {
		msg = fmt.Sprintf("Token accepted (%dms)", time.Since(start).Milliseconds())
		if identity != nil && identity.EmailAddress != "" {
			msg += fmt.Sprintf(" for %s", identity.EmailAddress)
		}
	}
	return DoctorCheck{Name: "Authentication", Status: "pass", Message: msg}
}

func checkDoctorAccountAccess(ctx context.Context, eff doctorEffectiveConfig, verbose bool) DoctorCheck {
	if eff.ProfileName == "" {
		return DoctorCheck{
			Name:    "Account Access",
			Status:  "warn",
			Message: "No profile/account configured",
			Hint:    "Set --profile, FIZZY_PROFILE, or run: fizzy setup",
		}
	}
	_, accountClient, err := newDoctorClients(eff)
	if err != nil {
		return DoctorCheck{Name: "Account Access", Status: "fail", Message: "SDK initialization failed", Hint: err.Error()}
	}
	start := time.Now()
	items, _, err := accountClient.Boards().List(ctx, "/boards.json")
	if err != nil {
		conv := convertSDKError(err)
		var outErr *output.Error
		if stderrors.As(conv, &outErr) {
			return DoctorCheck{Name: "Account Access", Status: "fail", Message: fmt.Sprintf("Cannot access account %s", eff.ProfileName), Hint: outErr.Message}
		}
		return DoctorCheck{Name: "Account Access", Status: "fail", Message: fmt.Sprintf("Cannot access account %s", eff.ProfileName), Hint: err.Error()}
	}
	count := dataCount(normalizeAny(items))
	msg := fmt.Sprintf("Account %s accessible", eff.ProfileName)
	if verbose {
		msg = fmt.Sprintf("Account %s accessible (%d boards, %dms)", eff.ProfileName, count, time.Since(start).Milliseconds())
	}
	return DoctorCheck{Name: "Account Access", Status: "pass", Message: msg}
}

func checkDoctorBoardAccess(ctx context.Context, eff doctorEffectiveConfig, verbose bool) DoctorCheck {
	if eff.Board == "" {
		return DoctorCheck{
			Name:    "Default Board",
			Status:  "warn",
			Message: "No default board configured",
			Hint:    "Set FIZZY_BOARD, add board to your profile, or run: fizzy setup",
		}
	}
	_, accountClient, err := newDoctorClients(eff)
	if err != nil {
		return DoctorCheck{Name: "Default Board", Status: "fail", Message: "SDK initialization failed", Hint: err.Error()}
	}
	start := time.Now()
	resp, err := accountClient.Get(ctx, "/boards/"+eff.Board+".json")
	if err != nil {
		conv := convertSDKError(err)
		var outErr *output.Error
		if stderrors.As(conv, &outErr) {
			return DoctorCheck{Name: "Default Board", Status: "fail", Message: fmt.Sprintf("Cannot access board %s", eff.Board), Hint: outErr.Message}
		}
		return DoctorCheck{Name: "Default Board", Status: "fail", Message: fmt.Sprintf("Cannot access board %s", eff.Board), Hint: err.Error()}
	}
	name := eff.Board
	if board, ok := normalizeAny(resp.Data).(map[string]any); ok {
		if boardName, ok := board["name"].(string); ok && boardName != "" {
			name = boardName
		}
	}
	msg := fmt.Sprintf("Board %s accessible", eff.Board)
	if verbose {
		msg = fmt.Sprintf("Board %s accessible as %q (%dms)", eff.Board, name, time.Since(start).Milliseconds())
	}
	return DoctorCheck{Name: "Default Board", Status: "pass", Message: msg}
}

func checkDoctorFilesystem(verbose bool) DoctorCheck {
	var warnings []string
	var badPaths []string

	globalPath := globalConfigPathForDoctor()
	if globalPath != "" {
		badPaths = appendIfBadParent(badPaths, filepath.Dir(globalPath))
		warnings = appendPermissionWarning(warnings, globalPath)
	}
	if localPath := config.LocalConfigPath(); localPath != "" {
		badPaths = appendIfBadParent(badPaths, filepath.Dir(localPath))
		warnings = appendPermissionWarning(warnings, localPath)
	}
	if home, err := os.UserHomeDir(); err == nil {
		badPaths = appendIfBadParent(badPaths, filepath.Join(home, ".agents", "skills"))
	}

	if len(badPaths) > 0 {
		return DoctorCheck{
			Name:    "Filesystem",
			Status:  "fail",
			Message: "Path collision detected",
			Hint:    strings.Join(badPaths, "; "),
		}
	}
	if len(warnings) > 0 {
		return DoctorCheck{
			Name:    "Filesystem",
			Status:  "warn",
			Message: "Sensitive files have broad permissions",
			Hint:    strings.Join(warnings, "; "),
		}
	}
	msg := "Expected paths look sane"
	if verbose && globalPath != "" {
		msg = fmt.Sprintf("Expected paths look sane (%s)", filepath.Dir(globalPath))
	}
	return DoctorCheck{Name: "Filesystem", Status: "pass", Message: msg}
}

func checkDoctorShellCompletion(verbose bool) DoctorCheck {
	shell := detectDoctorShell()
	if shell == "" {
		return DoctorCheck{Name: "Shell Completion", Status: "skip", Message: "Could not detect shell"}
	}
	installed, path := doctorCompletionPath(shell)
	if installed {
		msg := fmt.Sprintf("%s (installed)", shell)
		if verbose && path != "" {
			msg = fmt.Sprintf("%s (%s)", shell, path)
		}
		return DoctorCheck{Name: "Shell Completion", Status: "pass", Message: msg}
	}
	return DoctorCheck{
		Name:    "Shell Completion",
		Status:  "warn",
		Message: fmt.Sprintf("%s completion not installed", shell),
		Hint:    fmt.Sprintf("Run: fizzy completion %s --help", shell),
	}
}

func checkDoctorSkillInstallation() DoctorCheck {
	if baselineSkillInstalled() {
		return DoctorCheck{Name: "Agent Skill", Status: "pass", Message: "Installed"}
	}
	return DoctorCheck{
		Name:    "Agent Skill",
		Status:  "warn",
		Message: "Baseline skill not installed",
		Hint:    "Run: fizzy skill install",
	}
}

func checkDoctorSkillVersion() DoctorCheck {
	installed := installedSkillVersion()
	if installed == "" {
		return DoctorCheck{Name: "Skill Version", Status: "pass", Message: "Installed (version not tracked)"}
	}
	if currentVersion() == "dev" {
		return DoctorCheck{Name: "Skill Version", Status: "pass", Message: fmt.Sprintf("Installed (%s, dev build)", installed)}
	}
	if installed == currentVersion() {
		return DoctorCheck{Name: "Skill Version", Status: "pass", Message: fmt.Sprintf("Up to date (%s)", installed)}
	}
	return DoctorCheck{
		Name:    "Skill Version",
		Status:  "warn",
		Message: fmt.Sprintf("Stale (installed: %s, current: %s)", installed, currentVersion()),
		Hint:    "Run: fizzy skill install",
	}
}

func buildDoctorBreadcrumbs(checks []DoctorCheck) []Breadcrumb {
	var breadcrumbs []Breadcrumb
	for _, c := range checks {
		if c.Status != "fail" && c.Status != "warn" {
			continue
		}
		switch c.Name {
		case "Global Config", "Local Config", "API URL", "Filesystem", "Effective Config":
			breadcrumbs = append(breadcrumbs, breadcrumb("setup", "fizzy setup", "Review and repair configuration"))
		case "Profile Store":
			breadcrumbs = append(breadcrumbs,
				breadcrumb("profiles", "fizzy auth list", "List saved profiles"),
				breadcrumb("doctor_all", "fizzy doctor --all-profiles", "Check every saved profile"),
			)
		case "Credentials", "Credential Storage", "Authentication", "Legacy Environment":
			breadcrumbs = append(breadcrumbs,
				breadcrumb("login", "fizzy auth login <token>", "Refresh credentials"),
				breadcrumb("status", "fizzy auth status", "Check authentication status"),
			)
		case "Account Access":
			breadcrumbs = append(breadcrumbs,
				breadcrumb("identity", "fizzy identity show", "Verify identity and accounts"),
				breadcrumb("boards", "fizzy board list", "List boards for the active account"),
			)
		case "Default Board":
			breadcrumbs = append(breadcrumbs, breadcrumb("boards", "fizzy board list", "Pick a working default board"))
		case "Shell Completion":
			if shell := detectDoctorShell(); shell != "" {
				breadcrumbs = append(breadcrumbs, breadcrumb("completion", fmt.Sprintf("fizzy completion %s --help", shell), "Install shell completion"))
			}
		case "Agent Skill", "Skill Version":
			breadcrumbs = append(breadcrumbs, breadcrumb("skill", "fizzy skill install", "Install or refresh the agent skill"))
		case "Claude Code Plugin", "Claude Code Skill":
			breadcrumbs = append(breadcrumbs, breadcrumb("claude", "fizzy setup claude", "Repair Claude Code integration"))
		}
	}
	seen := map[string]bool{}
	unique := make([]Breadcrumb, 0, len(breadcrumbs))
	for _, crumb := range breadcrumbs {
		if seen[crumb.Cmd] {
			continue
		}
		seen[crumb.Cmd] = true
		unique = append(unique, crumb)
	}
	return unique
}

func renderDoctorStyled(w io.Writer, result *DoctorResult, breadcrumbs []Breadcrumb) {
	if w == nil {
		return
	}
	title := lipgloss.NewStyle().Bold(true)
	passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	statusIcon := map[string]string{"pass": "✓", "fail": "✗", "warn": "!", "skip": "○"}
	statusStyle := map[string]lipgloss.Style{"pass": passStyle, "fail": failStyle, "warn": warnStyle, "skip": skipStyle}
	renderChecks := func(indent string, checks []DoctorCheck) {
		for _, check := range checks {
			style := statusStyle[check.Status]
			fmt.Fprintf(w, "%s%s %s %s\n", indent, style.Render(statusIcon[check.Status]), title.Render(check.Name), style.Render(check.Message))
			if check.Hint != "" && (check.Status == "warn" || check.Status == "fail") {
				fmt.Fprintf(w, "%s    %s\n", indent, hintStyle.Render("↳ "+check.Hint))
			}
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, title.Render("Fizzy CLI Doctor"))
	fmt.Fprintln(w)
	renderChecks("  ", result.Checks)
	if len(result.Profiles) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, title.Render("Profiles"))
		for _, profileResult := range result.Profiles {
			name := profileResult.Name
			if profileResult.Default {
				name += " (default)"
			}
			fmt.Fprintf(w, "  %s\n", title.Render(name))
			renderChecks("    ", profileResult.Checks)
			fmt.Fprintln(w)
		}
	}
	var summaryParts []string
	if result.Passed > 0 {
		summaryParts = append(summaryParts, passStyle.Render(fmt.Sprintf("%d passed", result.Passed)))
	}
	if result.Failed > 0 {
		summaryParts = append(summaryParts, failStyle.Render(fmt.Sprintf("%d failed", result.Failed)))
	}
	if result.Warned > 0 {
		summaryParts = append(summaryParts, warnStyle.Render(fmt.Sprintf("%d %s", result.Warned, pluralize(result.Warned, "warning", "warnings"))))
	}
	if result.Skipped > 0 {
		summaryParts = append(summaryParts, skipStyle.Render(fmt.Sprintf("%d skipped", result.Skipped)))
	}
	fmt.Fprintf(w, "  %s\n", strings.Join(summaryParts, "  "))
	if len(breadcrumbs) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, title.Render("Next steps"))
		for _, crumb := range breadcrumbs {
			fmt.Fprintf(w, "  %s", crumb.Cmd)
			if crumb.Description != "" {
				fmt.Fprintf(w, "  # %s", crumb.Description)
			}
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintln(w)
}

func renderDoctorMarkdown(w io.Writer, result *DoctorResult, breadcrumbs []Breadcrumb) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, "# Fizzy CLI Doctor")
	fmt.Fprintln(w)
	for _, check := range result.Checks {
		icon := map[string]string{"pass": "✅", "fail": "❌", "warn": "⚠️", "skip": "➖"}[check.Status]
		fmt.Fprintf(w, "- %s **%s:** %s\n", icon, check.Name, check.Message)
		if check.Hint != "" && (check.Status == "warn" || check.Status == "fail") {
			fmt.Fprintf(w, "  - Hint: %s\n", check.Hint)
		}
	}
	if len(result.Profiles) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Profiles")
		for _, profileResult := range result.Profiles {
			name := profileResult.Name
			if profileResult.Default {
				name += " (default)"
			}
			fmt.Fprintf(w, "### %s\n", name)
			for _, check := range profileResult.Checks {
				icon := map[string]string{"pass": "✅", "fail": "❌", "warn": "⚠️", "skip": "➖"}[check.Status]
				fmt.Fprintf(w, "- %s **%s:** %s\n", icon, check.Name, check.Message)
				if check.Hint != "" && (check.Status == "warn" || check.Status == "fail") {
					fmt.Fprintf(w, "  - Hint: %s\n", check.Hint)
				}
			}
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintf(w, "**Summary:** %s\n", result.Summary())
	if len(breadcrumbs) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "## Next steps")
		for _, crumb := range breadcrumbs {
			fmt.Fprintf(w, "- `%s`", crumb.Cmd)
			if crumb.Description != "" {
				fmt.Fprintf(w, " — %s", crumb.Description)
			}
			fmt.Fprintln(w)
		}
	}
}

func detectDoctorShell() string {
	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" {
		return ""
	}
	switch filepath.Base(shell) {
	case "bash", "zsh", "fish":
		return filepath.Base(shell)
	default:
		return ""
	}
}

func doctorCompletionPath(shell string) (bool, string) {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home != "" {
		home = filepath.Clean(home)
	}
	switch shell {
	case "bash":
		paths := []string{"/opt/homebrew/etc/bash_completion.d/fizzy", "/usr/local/etc/bash_completion.d/fizzy", "/etc/bash_completion.d/fizzy"}
		if home != "" {
			paths = append(paths, filepath.Join(home, ".local", "share", "bash-completion", "completions", "fizzy"))
		}
		for _, path := range paths {
			if doctorPathExists(path) {
				return true, path
			}
		}
	case "zsh":
		paths := []string{"/opt/homebrew/share/zsh/site-functions/_fizzy", "/usr/local/share/zsh/site-functions/_fizzy"}
		if home != "" {
			paths = append(paths, filepath.Join(home, ".zsh", "completions", "_fizzy"))
		}
		for _, path := range paths {
			if doctorPathExists(path) {
				return true, path
			}
		}
		if doctorZshrcHasCompletionEval() {
			return true, "~/.zshrc (via eval)"
		}
	case "fish":
		if home != "" {
			path := filepath.Join(home, ".config", "fish", "completions", "fizzy.fish")
			if doctorPathExists(path) {
				return true, path
			}
		}
	}
	return false, ""
}

func doctorZshrcHasCompletionEval() bool {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(filepath.Clean(home), ".zshrc")) //nolint:gosec // trusted home-relative path for best-effort shell detection
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "fizzy completion zsh")
}

func globalConfigPathForDoctor() string {
	paths := config.GlobalConfigPaths()
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

func loadDoctorConfigFile(path string) (*config.Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfgFile config.Config
	if err := yaml.Unmarshal(data, &cfgFile); err != nil {
		return nil, err
	}
	return &cfgFile, nil
}

func resolveDoctorProfileContext() (string, *profile.Profile) {
	if profiles == nil {
		return "", nil
	}
	allProfiles, defaultName, err := profiles.List()
	if err != nil || len(allProfiles) == 0 {
		return "", nil
	}
	resolved, err := profile.Resolve(profile.ResolveOptions{
		FlagValue:      cfgProfile,
		EnvVar:         profileEnvVar(),
		DefaultProfile: defaultName,
		Profiles:       allProfiles,
	})
	if err != nil || resolved == "" {
		return "", nil
	}
	return resolved, allProfiles[resolved]
}

func doctorProfileBoard(p *profile.Profile) string {
	if p == nil {
		return ""
	}
	boardRaw, ok := p.Extra["board"]
	if !ok {
		return ""
	}
	var board string
	if json.Unmarshal(boardRaw, &board) != nil {
		return ""
	}
	return board
}

func doctorTokenSourceWithValue(account string, localCfg, globalCfg *config.Config) (string, string, string) {
	if cfgToken != "" {
		return "flag", "CLI flag", cfgToken
	}
	if envToken := os.Getenv("FIZZY_TOKEN"); envToken != "" {
		return "env", "environment variable", envToken
	}
	if creds != nil {
		if account != "" {
			if token, err := credsLoadProfileToken(account); err == nil && token != "" {
				if creds.UsingKeyring() {
					return "keyring", "system keyring", token
				}
				return "fallback-file", "fallback credential file", token
			}
			if token, err := credsLoadLegacyToken(account); err == nil && token != "" {
				if creds.UsingKeyring() {
					return "legacy-keyring", "legacy system keyring entry", token
				}
				return "legacy-fallback", "legacy fallback credential file", token
			}
		} else if token, err := credsLoadLegacyToken(""); err == nil && token != "" {
			if creds.UsingKeyring() {
				return "legacy-keyring", "legacy system keyring entry", token
			}
			return "legacy-fallback", "legacy fallback credential file", token
		}
	}
	if localCfg != nil && localCfg.Token != "" {
		return "local-config", "local config file", localCfg.Token
	}
	if globalCfg != nil && globalCfg.Token != "" {
		return "global-config", "global config file", globalCfg.Token
	}
	return "none", "not configured", ""
}

func checkDoctorSavedProfiles() DoctorCheck {
	if profiles == nil {
		return DoctorCheck{Name: "Saved Profiles", Status: "skip", Message: "Profile store unavailable"}
	}
	allProfiles, defaultName, err := profiles.List()
	if err != nil {
		return DoctorCheck{Name: "Saved Profiles", Status: "skip", Message: "Unavailable", Hint: err.Error()}
	}
	if len(allProfiles) == 0 {
		return DoctorCheck{Name: "Saved Profiles", Status: "skip", Message: "No saved profiles"}
	}
	names := make([]string, 0, len(allProfiles))
	for name := range allProfiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		if name == defaultName {
			names[i] = name + " (default)"
		}
	}
	return DoctorCheck{
		Name:    "Saved Profiles",
		Status:  "pass",
		Message: strings.Join(names, ", "),
		Hint:    "Use: fizzy doctor --all-profiles to check each saved profile",
	}
}

func doctorTargetsFromProfileStore() []doctorEffectiveConfig {
	if profiles == nil {
		return nil
	}
	allProfiles, defaultName, err := profiles.List()
	if err != nil || len(allProfiles) == 0 {
		return nil
	}
	globalCfg, _ := loadDoctorConfigFile(globalConfigPathForDoctor())
	localCfg, _ := loadDoctorConfigFile(config.LocalConfigPath())

	names := make([]string, 0, len(allProfiles))
	for name := range allProfiles {
		names = append(names, name)
	}
	sort.Strings(names)

	targets := make([]doctorEffectiveConfig, 0, len(names))
	for _, name := range names {
		p := allProfiles[name]
		board := doctorProfileBoard(p)
		tokenRaw, tokenSource, token := doctorStoredTokenSourceForProfile(name, localCfg, globalCfg)
		apiURL := config.DefaultAPIURL
		apiURLSource := "default"
		switch {
		case p != nil && strings.TrimSpace(p.BaseURL) != "":
			apiURL = p.BaseURL
			apiURLSource = "profile store"
		case localCfg != nil && strings.TrimSpace(localCfg.APIURL) != "":
			apiURL = localCfg.APIURL
			apiURLSource = "local config"
		case globalCfg != nil && strings.TrimSpace(globalCfg.APIURL) != "":
			apiURL = globalCfg.APIURL
			apiURLSource = "global config"
		}
		boardSource := "unset"
		switch {
		case board != "":
			boardSource = "profile store"
		case localCfg != nil && strings.TrimSpace(localCfg.Board) != "":
			board = localCfg.Board
			boardSource = "local config"
		case globalCfg != nil && strings.TrimSpace(globalCfg.Board) != "":
			board = globalCfg.Board
			boardSource = "global config"
		}
		targets = append(targets, doctorEffectiveConfig{
			ProfileName:    name,
			Default:        name == defaultName,
			ProfileSource:  "profile store",
			APIURL:         apiURL,
			APIURLSource:   apiURLSource,
			Board:          board,
			BoardSource:    boardSource,
			Token:          token,
			TokenSourceRaw: tokenRaw,
			TokenSource:    tokenSource,
		})
	}
	return targets
}

func doctorStoredTokenSourceForProfile(account string, localCfg, globalCfg *config.Config) (string, string, string) {
	if creds != nil {
		if token, err := credsLoadProfileToken(account); err == nil && token != "" {
			if creds.UsingKeyring() {
				return "keyring", "system keyring", token
			}
			return "fallback-file", "fallback credential file", token
		}
		if token, err := credsLoadLegacyToken(account); err == nil && token != "" {
			if creds.UsingKeyring() {
				return "legacy-keyring", "legacy system keyring entry", token
			}
			return "legacy-fallback", "legacy fallback credential file", token
		}
	}
	if localCfg != nil && localCfg.Token != "" {
		return "local-config", "local config file", localCfg.Token
	}
	if globalCfg != nil && globalCfg.Token != "" {
		return "global-config", "global config file", globalCfg.Token
	}
	return "none", "not configured", ""
}

func newDoctorClients(eff doctorEffectiveConfig) (client *fizzy.Client, accountClient *fizzy.AccountClient, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cannot initialize SDK: %v", r)
			client = nil
			accountClient = nil
		}
	}()
	sdkCfg := &fizzy.Config{BaseURL: eff.APIURL}
	client = fizzy.NewClient(sdkCfg, &fizzy.StaticTokenProvider{Token: eff.Token}, fizzy.WithUserAgent("fizzy-cli/"+currentVersion()))
	accountClient = client.ForAccount(eff.ProfileName)
	return client, accountClient, nil
}

func doctorLoginHint(profileName string) string {
	if strings.TrimSpace(profileName) == "" {
		return "Run: fizzy auth login <token>"
	}
	return fmt.Sprintf("Run: fizzy auth login <token> --profile %s", profileName)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func doctorPathExists(path string) bool {
	clean := filepath.Clean(path)
	_, err := os.Stat(clean) //nolint:gosec // trusted path built from constants and cleaned home-relative locations
	return err == nil
}

func isOptionalAgentCheck(name string) bool {
	switch name {
	case "Claude Code Plugin", "Claude Code Skill":
		return true
	default:
		return false
	}
}

func appendIfBadParent(paths []string, dir string) []string {
	if strings.TrimSpace(dir) == "" {
		return paths
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return append(paths, fmt.Sprintf("%s exists but is not a directory", dir))
	}
	return paths
}

func appendPermissionWarning(warnings []string, path string) []string {
	if runtime.GOOS == "windows" || strings.TrimSpace(path) == "" {
		return warnings
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return warnings
	}
	if info.Mode().Perm()&0o077 != 0 {
		return append(warnings, fmt.Sprintf("tighten permissions on %s (recommended: chmod 600)", path))
	}
	return warnings
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
