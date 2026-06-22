package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/cli/profile"
	cfgpkg "github.com/basecamp/fizzy-cli/internal/config"
	"github.com/spf13/cobra"
)

type configExplainCandidate struct {
	Source   string `json:"source"`
	Value    string `json:"value,omitempty"`
	Selected bool   `json:"selected,omitempty"`
}

type configExplainField struct {
	Value      any                      `json:"value,omitempty"`
	Source     string                   `json:"source"`
	Configured *bool                    `json:"configured,omitempty"`
	Candidates []configExplainCandidate `json:"candidates"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect resolved configuration",
	Long:  "Inspect effective configuration values and explain why specific values win.",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective configuration",
	Long: `Show the currently effective configuration after applying precedence rules
(flags, environment variables, profile settings, local config, and global config).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		data := configShowData(cfgVerbose)
		breadcrumbs := []Breadcrumb{
			breadcrumb("doctor", "fizzy doctor", "Run a full health check"),
			breadcrumb("explain", "fizzy config explain", "Explain configuration precedence"),
			breadcrumb("profiles", "fizzy auth list", "List saved profiles"),
		}

		switch out.EffectiveFormat() {
		case output.FormatStyled:
			writeOutputString(renderConfigShowHuman(data, false))
			captureResponse()
			return nil
		case output.FormatMarkdown:
			writeOutputString(renderConfigShowHuman(data, true))
			captureResponse()
			return nil
		default:
			printDetail(data, "Effective configuration", breadcrumbs)
			return nil
		}
	},
}

var configExplainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain configuration precedence",
	Long: `Explain why each effective configuration value won, including overridden candidates
from flags, environment variables, profile settings, local config, and global config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		data := configExplainData()
		breadcrumbs := []Breadcrumb{
			breadcrumb("doctor", "fizzy doctor", "Run a full health check"),
			breadcrumb("show", "fizzy config show", "Show the effective config only"),
			breadcrumb("profiles", "fizzy auth list", "List saved profiles"),
		}

		switch out.EffectiveFormat() {
		case output.FormatStyled:
			writeOutputString(renderConfigExplainHuman(data, false))
			captureResponse()
			return nil
		case output.FormatMarkdown:
			writeOutputString(renderConfigExplainHuman(data, true))
			captureResponse()
			return nil
		default:
			recordOutputError(out.OK(data,
				output.WithSummary("Configuration precedence"),
				output.WithBreadcrumbs(breadcrumbs...),
			))
			captureResponse()
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configExplainCmd)
}

func configShowData(verbose bool) map[string]any {
	eff := resolveDoctorEffectiveConfig()
	_, defaultProfile := profileStoreInfo()
	data := map[string]any{
		"profile": map[string]any{
			"value":   emptyToNil(eff.ProfileName),
			"source":  displayProfileSource(eff, defaultProfile),
			"default": eff.Default,
		},
		"api_url": map[string]any{
			"value":  emptyToNil(eff.APIURL),
			"source": displayConfigSource(eff.APIURLSource),
		},
		"board": map[string]any{
			"value":  emptyToNil(eff.Board),
			"source": displayConfigSource(eff.BoardSource),
		},
		"token": map[string]any{
			"configured": eff.Token != "",
			"source":     displayTokenSource(eff.TokenSource),
		},
	}

	if !verbose {
		data = map[string]any{
			"profile":  emptyToNil(eff.ProfileName),
			"api_url":  emptyToNil(eff.APIURL),
			"board":    emptyToNil(eff.Board),
			"token":    map[string]any{"configured": eff.Token != "", "source": eff.TokenSource},
			"profiles": savedProfileNames(),
		}
	} else {
		data["profiles"] = savedProfileNames()
	}

	return data
}

func configExplainData() map[string]any {
	eff := resolveDoctorEffectiveConfig()
	globalCfg, _ := loadDoctorConfigFile(globalConfigPathForDoctor())
	localCfg, _ := loadDoctorConfigFile(cfgpkg.LocalConfigPath())
	resolvedProfile, profileCfg := resolveDoctorProfileContext()
	_, defaultProfile := profileStoreInfo()
	_, _, profileToken := doctorStoredTokenSourceForProfile(resolvedProfileOrEffective(resolvedProfile, eff.ProfileName), localCfg, globalCfg)

	profileField := configExplainField{
		Value:  emptyToNil(eff.ProfileName),
		Source: displayProfileSource(eff, defaultProfile),
		Candidates: []configExplainCandidate{
			{Source: "flag --profile", Value: unsetString(cfgProfile), Selected: eff.ProfileSource == "flag --profile"},
			{Source: "env FIZZY_PROFILE", Value: unsetString(strings.TrimSpace(getEnv("FIZZY_PROFILE"))), Selected: eff.ProfileSource == "env FIZZY_PROFILE"},
			{Source: "env FIZZY_ACCOUNT", Value: unsetString(strings.TrimSpace(getEnv("FIZZY_ACCOUNT"))), Selected: eff.ProfileSource == "env FIZZY_ACCOUNT"},
			{Source: "default profile", Value: unsetString(defaultProfile), Selected: eff.ProfileSource == "profile store"},
			{Source: "local config", Value: unsetString(fieldValue(localCfg, func(c *cfgpkg.Config) string { return c.Account })), Selected: eff.ProfileSource == "local config"},
			{Source: "global config", Value: unsetString(fieldValue(globalCfg, func(c *cfgpkg.Config) string { return c.Account })), Selected: eff.ProfileSource == "global config"},
		},
	}

	apiURLField := configExplainField{
		Value:  emptyToNil(eff.APIURL),
		Source: displayFieldSource(eff.APIURLSource, profileSourceLabel(resolvedProfile, eff.ProfileName)),
		Candidates: []configExplainCandidate{
			{Source: "flag --api-url", Value: unsetString(cfgAPIURL), Selected: eff.APIURLSource == "flag --api-url"},
			{Source: "env FIZZY_API_URL", Value: unsetString(strings.TrimSpace(getEnv("FIZZY_API_URL"))), Selected: eff.APIURLSource == "env FIZZY_API_URL"},
			{Source: profileSourceLabel(resolvedProfile, eff.ProfileName), Value: unsetString(profileBaseURL(profileCfg)), Selected: eff.APIURLSource == "profile store"},
			{Source: "local config", Value: unsetString(fieldValue(localCfg, func(c *cfgpkg.Config) string { return c.APIURL })), Selected: eff.APIURLSource == "local config"},
			{Source: "global config", Value: unsetString(fieldValue(globalCfg, func(c *cfgpkg.Config) string { return c.APIURL })), Selected: eff.APIURLSource == "global config"},
			{Source: "default", Value: cfgpkg.DefaultAPIURL, Selected: eff.APIURLSource == "default"},
		},
	}

	boardField := configExplainField{
		Value:  emptyToNil(eff.Board),
		Source: displayFieldSource(eff.BoardSource, profileSourceLabel(resolvedProfile, eff.ProfileName)),
		Candidates: []configExplainCandidate{
			{Source: "env FIZZY_BOARD", Value: unsetString(strings.TrimSpace(getEnv("FIZZY_BOARD"))), Selected: eff.BoardSource == "env FIZZY_BOARD"},
			{Source: profileSourceLabel(resolvedProfile, eff.ProfileName), Value: unsetString(doctorProfileBoard(profileCfg)), Selected: eff.BoardSource == "profile store"},
			{Source: "local config", Value: unsetString(fieldValue(localCfg, func(c *cfgpkg.Config) string { return c.Board })), Selected: eff.BoardSource == "local config"},
			{Source: "global config", Value: unsetString(fieldValue(globalCfg, func(c *cfgpkg.Config) string { return c.Board })), Selected: eff.BoardSource == "global config"},
		},
	}

	configured := eff.Token != ""
	tokenField := configExplainField{
		Source:     displayTokenSource(eff.TokenSource),
		Configured: &configured,
		Candidates: []configExplainCandidate{
			{Source: "flag --token", Value: configuredString(cfgToken != "", "configured via flag"), Selected: eff.TokenSourceRaw == "flag"},
			{Source: "env FIZZY_TOKEN", Value: configuredString(strings.TrimSpace(getEnv("FIZZY_TOKEN")) != "", "configured in environment"), Selected: eff.TokenSourceRaw == "env"},
			{Source: profileTokenSourceLabel(resolvedProfile, eff.ProfileName), Value: configuredString(profileToken != "", profileCredentialValue(eff.TokenSourceRaw, profileToken != "")), Selected: eff.TokenSourceRaw == "keyring" || eff.TokenSourceRaw == "fallback-file" || eff.TokenSourceRaw == "legacy-keyring" || eff.TokenSourceRaw == "legacy-fallback"},
			{Source: "local config", Value: configuredString(localCfg != nil && localCfg.Token != "", "configured in local config"), Selected: eff.TokenSourceRaw == "local-config"},
			{Source: "global config", Value: configuredString(globalCfg != nil && globalCfg.Token != "", "configured in global config"), Selected: eff.TokenSourceRaw == "global-config"},
		},
	}

	return map[string]any{
		"profile":        profileField,
		"api_url":        apiURLField,
		"board":          boardField,
		"token":          tokenField,
		"saved_profiles": savedProfileNames(),
	}
}

func renderConfigShowHuman(data map[string]any, markdown bool) string {
	var sb strings.Builder
	if markdown {
		sb.WriteString("# Fizzy Config\n\n")
	} else {
		sb.WriteString("Fizzy Config\n\n")
	}

	profile := describeConfigShowValue(data["profile"])
	apiURL := describeConfigShowValue(data["api_url"])
	board := describeConfigShowValue(data["board"])
	token := describeConfigShowToken(data["token"])
	profiles := describeConfigShowProfiles(data["profiles"])

	if markdown {
		fmt.Fprintf(&sb, "- **Profile:** `%s`\n", profile)
		fmt.Fprintf(&sb, "- **API URL:** `%s`\n", apiURL)
		fmt.Fprintf(&sb, "- **Board:** `%s`\n", board)
		fmt.Fprintf(&sb, "- **Token:** %s\n", token)
		if profiles != "" {
			fmt.Fprintf(&sb, "- **Saved Profiles:** %s\n", profiles)
		}
		sb.WriteString("\n## Next steps\n")
		sb.WriteString("- `fizzy config explain` — explain precedence\n")
		sb.WriteString("- `fizzy doctor` — run a full health check\n")
		sb.WriteString("- `fizzy auth list` — inspect saved profiles\n")
	} else {
		fmt.Fprintf(&sb, "Profile        %s\n", profile)
		fmt.Fprintf(&sb, "API URL        %s\n", apiURL)
		fmt.Fprintf(&sb, "Board          %s\n", board)
		fmt.Fprintf(&sb, "Token          %s\n", token)
		if profiles != "" {
			fmt.Fprintf(&sb, "Saved Profiles %s\n", profiles)
		}
		sb.WriteString("\nNext steps\n")
		sb.WriteString("  fizzy config explain  # explain precedence\n")
		sb.WriteString("  fizzy doctor          # run a full health check\n")
		sb.WriteString("  fizzy auth list       # inspect saved profiles\n")
	}

	return sb.String()
}

func renderConfigExplainHuman(data map[string]any, markdown bool) string {
	var sb strings.Builder
	if markdown {
		sb.WriteString("# Fizzy Config Explain\n\n")
	} else {
		sb.WriteString("Fizzy Config Explain\n\n")
	}

	order := []struct {
		key   string
		label string
	}{
		{"profile", "Profile"},
		{"api_url", "API URL"},
		{"board", "Board"},
		{"token", "Token"},
	}

	for _, item := range order {
		field, ok := data[item.key].(configExplainField)
		if !ok {
			continue
		}
		if markdown {
			fmt.Fprintf(&sb, "## %s\n", item.label)
		} else {
			fmt.Fprintf(&sb, "%s\n", item.label)
		}

		value := explainFieldValue(field)
		if markdown {
			fmt.Fprintf(&sb, "- Effective: `%s`\n", value)
			fmt.Fprintf(&sb, "- Source: `%s`\n", field.Source)
			sb.WriteString("- Candidates:\n")
			for _, candidate := range field.Candidates {
				marker := ""
				if candidate.Selected {
					marker = " **(selected)**"
				}
				fmt.Fprintf(&sb, "  - `%s`: `%s`%s\n", candidate.Source, candidate.Value, marker)
			}
			sb.WriteString("\n")
		} else {
			fmt.Fprintf(&sb, "  Effective: %s\n", value)
			fmt.Fprintf(&sb, "  Source:    %s\n", field.Source)
			sb.WriteString("  Candidates:\n")
			for _, candidate := range field.Candidates {
				marker := ""
				if candidate.Selected {
					marker = "  ✓"
				}
				fmt.Fprintf(&sb, "    - %-20s %s%s\n", candidate.Source+":", candidate.Value, marker)
			}
			sb.WriteString("\n")
		}
	}

	if profiles, ok := data["saved_profiles"].([]string); ok && len(profiles) > 0 {
		if markdown {
			sb.WriteString("## Saved Profiles\n")
			for _, name := range profiles {
				fmt.Fprintf(&sb, "- `%s`\n", name)
			}
		} else {
			sb.WriteString("Saved Profiles\n")
			for _, name := range profiles {
				fmt.Fprintf(&sb, "  - %s\n", name)
			}
		}
		sb.WriteString("\n")
	}

	if markdown {
		sb.WriteString("## Next steps\n")
		sb.WriteString("- `fizzy config show` — show the effective config only\n")
		sb.WriteString("- `fizzy doctor` — run a full health check\n")
		sb.WriteString("- `fizzy auth list` — inspect saved profiles\n")
	} else {
		sb.WriteString("Next steps\n")
		sb.WriteString("  fizzy config show   # show the effective config only\n")
		sb.WriteString("  fizzy doctor        # run a full health check\n")
		sb.WriteString("  fizzy auth list     # inspect saved profiles\n")
	}

	return sb.String()
}

func describeConfigShowValue(v any) string {
	switch value := v.(type) {
	case nil:
		return "<unset>"
	case string:
		if strings.TrimSpace(value) == "" {
			return "<unset>"
		}
		return value
	case map[string]any:
		if s, ok := value["value"].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
		return "<unset>"
	default:
		return fmt.Sprintf("%v", value)
	}
}

func describeConfigShowToken(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	configured, _ := m["configured"].(bool)
	source, _ := m["source"].(string)
	if !configured {
		return "not configured"
	}
	if strings.TrimSpace(source) == "" {
		return "configured"
	}
	return fmt.Sprintf("configured (%s)", source)
}

func describeConfigShowProfiles(v any) string {
	switch items := v.(type) {
	case []string:
		return strings.Join(items, ", ")
	case []any:
		parts := make([]string, 0, len(items))
		for _, item := range items {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
		return strings.Join(parts, ", ")
	default:
		return ""
	}
}

func explainFieldValue(field configExplainField) string {
	if field.Configured != nil {
		if *field.Configured {
			return "configured"
		}
		return "not configured"
	}
	if s, ok := field.Value.(string); ok {
		if strings.TrimSpace(s) == "" {
			return "<unset>"
		}
		return s
	}
	if field.Value == nil {
		return "<unset>"
	}
	return fmt.Sprintf("%v", field.Value)
}

func emptyToNil(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func unsetString(s string) string {
	if strings.TrimSpace(s) == "" {
		return "<unset>"
	}
	return s
}

func configuredString(ok bool, label string) string {
	if !ok {
		return "<unset>"
	}
	return label
}

func profileSourceLabel(resolved, effective string) string {
	name := resolvedProfileOrEffective(resolved, effective)
	if strings.TrimSpace(name) == "" {
		return "profile store"
	}
	return fmt.Sprintf("profile %s", name)
}

func profileTokenSourceLabel(resolved, effective string) string {
	name := resolvedProfileOrEffective(resolved, effective)
	if strings.TrimSpace(name) == "" {
		return "profile credential"
	}
	return fmt.Sprintf("profile credential %s", name)
}

func resolvedProfileOrEffective(resolved, effective string) string {
	if strings.TrimSpace(resolved) != "" {
		return resolved
	}
	return effective
}

func profileBaseURL(p *profile.Profile) string {
	if p == nil {
		return ""
	}
	return p.BaseURL
}

func fieldValue(cfg *cfgpkg.Config, getter func(*cfgpkg.Config) string) string {
	if cfg == nil {
		return ""
	}
	return getter(cfg)
}

func profileCredentialValue(raw string, ok bool) string {
	if !ok {
		return "profile credential missing"
	}
	switch raw {
	case "keyring":
		return "configured in system keyring"
	case "fallback-file":
		return "configured in fallback credential file"
	case "legacy-keyring":
		return "configured in legacy system keyring entry"
	case "legacy-fallback":
		return "configured in legacy fallback credential file"
	default:
		return "configured for selected profile"
	}
}

func displayProfileSource(eff doctorEffectiveConfig, defaultProfile string) string {
	if eff.ProfileSource == "profile store" && strings.TrimSpace(eff.ProfileName) != "" && eff.ProfileName == defaultProfile {
		return "default profile"
	}
	return displayConfigSource(eff.ProfileSource)
}

func displayFieldSource(source, profileLabel string) string {
	if strings.TrimSpace(source) == "profile store" {
		return profileLabel
	}
	return displayConfigSource(source)
}

func displayConfigSource(source string) string {
	switch strings.TrimSpace(source) {
	case "", "unset":
		return "not configured"
	default:
		return source
	}
}

func displayTokenSource(source string) string {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(source) == "not configured" {
		return "not configured"
	}
	return source
}

func savedProfileNames() []string {
	allProfiles, defaultName := profileStoreInfo()
	if len(allProfiles) == 0 {
		return nil
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
	return names
}

func profileStoreInfo() (map[string]*profile.Profile, string) {
	if profiles == nil {
		return nil, ""
	}
	allProfiles, defaultName, err := profiles.List()
	if err != nil {
		return nil, ""
	}
	return allProfiles, defaultName
}

func getEnv(name string) string {
	return strings.TrimSpace(os.Getenv(name))
}
