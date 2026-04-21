package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Dixa-public/dixa-cli-public/internal/client"
	"github.com/Dixa-public/dixa-cli-public/internal/config"
	"github.com/Dixa-public/dixa-cli-public/internal/confirm"
	"github.com/Dixa-public/dixa-cli-public/internal/output"
	"github.com/Dixa-public/dixa-cli-public/internal/spec"
)

var version = "dev"

type Env struct {
	HomeDir     string
	Spec        spec.Manifest
	Config      *config.Manager
	HTTPClient  *http.Client
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	IsStdoutTTY bool
	IsStdinTTY  bool
}

type rootFlags struct {
	Profile string
	BaseURL string
	APIKey  string
	Output  string
	Debug   bool
	Yes     bool
}

func NewDefaultEnv() (*Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	manifest, err := spec.Load()
	if err != nil {
		return nil, err
	}
	return &Env{
		HomeDir:     home,
		Spec:        manifest,
		Config:      config.NewManager(home, config.KeyringStore{Service: config.ServiceName}),
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		IsStdoutTTY: term.IsTerminal(int(os.Stdout.Fd())),
		IsStdinTTY:  term.IsTerminal(int(os.Stdin.Fd())),
	}, nil
}

func NewRootCmd(env *Env) *cobra.Command {
	flags := &rootFlags{}
	cmd := &cobra.Command{
		Use:           "dixa",
		Short:         "Terminal CLI for the Dixa API",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
	}

	cmd.PersistentFlags().StringVar(&flags.Profile, "profile", "", "Profile name from ~/.config/dixa/config.toml")
	cmd.PersistentFlags().StringVar(&flags.BaseURL, "base-url", "", "Override the Dixa API base URL")
	cmd.PersistentFlags().StringVar(&flags.APIKey, "api-key", "", "Override the Dixa API key for this command")
	cmd.PersistentFlags().StringVar(&flags.Output, "output", "auto", "Output format: auto, json, table, yaml")
	cmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Print redacted request and response debug logs to stderr")
	cmd.PersistentFlags().BoolVar(&flags.Yes, "yes", false, "Skip interactive confirmation for mutating commands")

	cmd.AddCommand(newAuthCmd(env, flags))

	groupCommands := map[string]*cobra.Command{}
	for _, domain := range env.Spec.Domains {
		group := &cobra.Command{
			Use:     domain.Group,
			Aliases: domain.GroupAliases,
			Short:   domain.Label + " commands",
		}
		groupCommands[domain.Group] = group
		cmd.AddCommand(group)
	}

	for _, operation := range env.Spec.Operations {
		groupCommands[operation.Group].AddCommand(newOperationCmd(env, flags, operation))
	}

	orderChildren(cmd)
	for _, child := range cmd.Commands() {
		orderChildren(child)
	}

	return cmd
}

func newAuthCmd(env *Env, flags *rootFlags) *cobra.Command {
	var loginAPIKey string
	var setDefault bool
	var removeProfile bool

	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage saved Dixa CLI credentials",
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Store an API key in Keychain for the selected profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := selectedProfile(env, flags, cmd)
			apiKey := loginAPIKey
			if apiKey == "" {
				apiKey = flags.APIKey
			}
			if apiKey == "" {
				apiKey = os.Getenv("DIXA_API_KEY")
			}
			if apiKey == "" {
				return fmt.Errorf("auth login requires --api-key or DIXA_API_KEY")
			}
			baseURL := flags.BaseURL
			if baseURL == "" {
				baseURL = config.DefaultBaseURL
			}
			outputValue := flags.Output
			if outputValue == "" {
				outputValue = config.DefaultOutput
			}
			if err := env.Config.UpsertProfile(profile, config.Profile{
				BaseURL: baseURL,
				Output:  outputValue,
			}, apiKey, setDefault); err != nil {
				return err
			}
			return renderResult(env, resolveOutput(flags, env), map[string]any{
				"profile":      profile,
				"base_url":     baseURL,
				"output":       outputValue,
				"key_stored":   true,
				"default_set":  setDefault,
				"config_path":  env.Config.Path,
				"api_key_hint": config.MaskSecret(apiKey),
			})
		},
	}
	loginCmd.Flags().StringVar(&loginAPIKey, "api-key", "", "API key to store in Keychain")
	loginCmd.Flags().BoolVar(&setDefault, "set-default", false, "Set the selected profile as the default profile")

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved API key for the selected profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := selectedProfile(env, flags, cmd)
			if err := env.Config.DeleteProfile(profile, removeProfile); err != nil {
				return err
			}
			return renderResult(env, resolveOutput(flags, env), map[string]any{
				"profile":         profile,
				"removed_secret":  true,
				"removed_profile": removeProfile,
			})
		},
	}
	logoutCmd.Flags().BoolVar(&removeProfile, "remove-profile", false, "Also remove the profile entry from config.toml")

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the resolved profile, base URL, output mode, and masked API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := env.Config.Resolve(flagOverrides(flags, cmd), os.Getenv)
			if err != nil {
				return err
			}
			result := map[string]any{
				"profile":         resolved.Profile,
				"profile_source":  resolved.ProfileSource,
				"base_url":        resolved.BaseURL,
				"base_url_source": resolved.BaseURLSource,
				"output":          resolved.Output,
				"output_source":   resolved.OutputSource,
				"api_key":         config.MaskSecret(resolved.APIKey),
				"api_key_source":  resolved.APIKeySource,
				"config_path":     resolved.ConfigPath,
			}
			return renderResult(env, resolved.Output, result)
		},
	}

	authCmd.AddCommand(loginCmd, logoutCmd, showCmd)
	return authCmd
}

func newOperationCmd(env *Env, flags *rootFlags, operation spec.Operation) *cobra.Command {
	use := operation.Name
	for _, param := range operation.PathParameters() {
		use += " <" + param.Flag + ">"
	}

	cmd := &cobra.Command{
		Use:     use,
		Aliases: operation.Aliases,
		Short:   operation.Summary,
		Long:    operation.Description,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := env.Config.Resolve(flagOverrides(flags, cmd), os.Getenv)
			if err != nil {
				return err
			}
			if resolved.APIKey == "" {
				return fmt.Errorf("no Dixa API key resolved; use `dixa auth login --api-key ...`, set DIXA_API_KEY, or pass --api-key")
			}

			params, err := collectParams(cmd, args, operation, env.Stdin)
			if err != nil {
				return err
			}
			if err := confirm.Ensure(operation, params, flags.Yes, env.Stdin, env.Stderr, env.IsStdinTTY); err != nil {
				return err
			}

			apiClient := client.New(client.Options{
				BaseURL:    resolved.BaseURL,
				APIKey:     resolved.APIKey,
				Debug:      flags.Debug,
				HTTPClient: env.HTTPClient,
				ErrWriter:  env.Stderr,
			})
			result, err := apiClient.ExecuteOperation(context.Background(), operation, params)
			if err != nil {
				return err
			}
			return renderResult(env, resolved.Output, result)
		},
	}

	cmd.Flags().String("input", "", "Read command parameters from a JSON file or `-` for stdin")
	for _, param := range operation.Parameters {
		registerParameterFlags(cmd, param)
	}

	return cmd
}

func registerParameterFlags(cmd *cobra.Command, param spec.Parameter) {
	description := strings.ReplaceAll(strings.TrimPrefix(param.Name, "_"), "_", " ")
	if description == "" {
		description = param.Name
	}
	register := func(name string) {
		switch param.Type {
		case "int":
			cmd.Flags().Int(name, 0, description)
		case "bool":
			cmd.Flags().Bool(name, false, description)
		case "string_slice":
			cmd.Flags().StringSlice(name, nil, description)
		default:
			cmd.Flags().String(name, "", description)
		}
	}
	register(param.Flag)
	for _, alias := range param.FlagAliases {
		register(alias)
		_ = cmd.Flags().MarkHidden(alias)
	}
}

func selectedProfile(env *Env, flags *rootFlags, cmd *cobra.Command) string {
	resolved, err := env.Config.Resolve(flagOverrides(flags, cmd), os.Getenv)
	if err == nil && resolved.Profile != "" {
		return resolved.Profile
	}
	if flags.Profile != "" {
		return flags.Profile
	}
	return config.DefaultProfileName
}

func flagOverrides(flags *rootFlags, cmd *cobra.Command) config.Overrides {
	return config.Overrides{
		Profile:    flags.Profile,
		BaseURL:    flags.BaseURL,
		APIKey:     flags.APIKey,
		Output:     flags.Output,
		ProfileSet: flagChanged(cmd, "profile"),
		BaseURLSet: flagChanged(cmd, "base-url"),
		APIKeySet:  flagChanged(cmd, "api-key"),
		OutputSet:  flagChanged(cmd, "output"),
	}
}

func flagChanged(cmd *cobra.Command, name string) bool {
	flag := cmd.Flags().Lookup(name)
	return flag != nil && flag.Changed
}

func resolveOutput(flags *rootFlags, env *Env) string {
	if flags.Output != "" {
		return flags.Output
	}
	if envValue := os.Getenv("DIXA_OUTPUT"); envValue != "" {
		return envValue
	}
	return config.DefaultOutput
}

func renderResult(env *Env, requestedFormat string, result any) error {
	format := output.ResolveFormat(requestedFormat, env.IsStdoutTTY)
	return output.Render(env.Stdout, format, result)
}

func orderChildren(cmd *cobra.Command) {
	children := cmd.Commands()
	sort.SliceStable(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})
}
