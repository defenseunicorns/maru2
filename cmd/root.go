// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package cmd provides the root command for the maru2 CLI.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2"
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for the maru2 CLI.
func NewRootCmd() *cobra.Command {
	var (
		w       map[string]string
		level   string
		ver     bool
		list    bool
		from    string
		policy  = config.DefaultFetchPolicy // VarP does not allow you to set a default value
		s       string
		timeout time.Duration
		dry     bool
		dir     string
	)

	var cfg *config.Config // cfg is not set via CLI flag

	root := &cobra.Command{
		Use:   "maru2",
		Short: "A simple task runner",
		Example: `
maru2 build

maru2 -f ../foo.yaml bar baz -w zab="zaz"

maru2 -f "pkg:github/defenseunicorns/maru2@main#testdata/simple.yaml" echo -w message="hello world"
`,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if dir != "" {
				if err := os.Chdir(dir); err != nil {
					return err
				}
			}

			configDir, err := config.DefaultDirectory()
			if err != nil {
				return err
			}

			loader := &config.FileSystemConfigLoader{
				Fs: afero.NewBasePathFs(afero.NewOsFs(), configDir),
			}

			cfg, err = loader.LoadConfig()
			if err != nil {
				return err
			}

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			svc, err := uses.NewFetcherService(
				uses.WithClient(&http.Client{
					Timeout: 500 * time.Millisecond,
				}),
			)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			resolved, err := uses.ResolveRelative(nil, from, cfg.Aliases)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			wf, err := maru2.Fetch(cmd.Context(), svc, resolved)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return wf.Tasks.OrderedTaskNames(), cobra.ShellCompDirectiveNoFileComp
		},
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			l, err := log.ParseLevel(level)
			if err != nil {
				return err
			}
			logger := log.FromContext(cmd.Context())
			logger.SetLevel(l)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			logger := log.FromContext(ctx)

			if ver && len(args) == 0 {
				bi, ok := debug.ReadBuildInfo()
				if !ok {
					return fmt.Errorf("version information not available")
				}
				logger.Printf("%s", bi.Main.Version)
				return nil
			}

			if cmpl, ok := os.LookupEnv("MARU2_COMPLETION"); ok && cmpl == "true" && len(args) == 2 && args[0] == "completion" {
				switch args[1] {
				case "bash":
					return cmd.GenBashCompletion(os.Stdout)
				case "zsh":
					return cmd.GenZshCompletion(os.Stdout)
				case "fish":
					return cmd.GenFishCompletion(os.Stdout, true)
				case "powershell":
					return cmd.GenPowerShellCompletionWithDesc(os.Stdout)
				default:
					return fmt.Errorf("unsupported shell: %s", args[1])
				}
			}

			// fix fish needing "'pkg:...'" for tab completion
			from = strings.Trim(from, `"`)
			from = strings.Trim(from, `'`)

			s := os.ExpandEnv(s)

			fs := afero.NewOsFs()

			_, err := fs.Stat(s)
			if err != nil {
				if os.IsNotExist(err) {
					if err := fs.MkdirAll(s, 0o744); err != nil {
						return err
					}
				} else {
					return err
				}
			}

			store, err := uses.NewStore(afero.NewBasePathFs(fs, s))
			if err != nil {
				return fmt.Errorf("failed to initialize store: %w", err)
			}

			svc, err := uses.NewFetcherService(
				uses.WithAliases(cfg.Aliases),
				uses.WithStore(store),
				uses.WithFetchPolicy(policy),
			)
			if err != nil {
				return fmt.Errorf("failed to initialize fetcher service: %w", err)
			}

			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			resolved, err := uses.ResolveRelative(nil, from, cfg.Aliases)
			if err != nil {
				return fmt.Errorf("failed to resolve %q: %w", from, err)
			}

			wf, err := maru2.Fetch(ctx, svc, resolved)
			if err != nil {
				return fmt.Errorf("failed to fetch %q: %w", resolved, err)
			}

			if list {
				names := wf.Tasks.OrderedTaskNames()

				if len(names) == 0 {
					return fmt.Errorf("no tasks available")
				}

				logger.Print("Available:\n")
				for _, n := range names {
					logger.Printf("- %s", n)
				}

				return nil
			}

			with := make(maru2.With, len(w))
			for k, v := range w {
				with[k] = v
			}

			if len(args) == 0 {
				args = append(args, maru2.DefaultTaskName)
			}

			for _, call := range args {
				start := time.Now()
				logger.Debug("run", "task", call, "from", resolved, "dry-run", dry)
				defer func() {
					logger.Debug("ran", "task", call, "from", resolved, "dry-run", dry, "duration", time.Since(start))
				}()
				_, err := maru2.Run(ctx, svc, wf, call, with, resolved, dry)
				if err != nil {
					if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						return fmt.Errorf("task %q timed out", call)
					}
					return err
				}
			}
			return nil
		},
	}

	root.Flags().StringToStringVarP(&w, "with", "w", nil, "Pass key=value pairs to the called task(s)")
	root.Flags().StringVarP(&level, "log-level", "l", "info", "Set log level")
	_ = root.RegisterFlagCompletionFunc("log-level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{log.DebugLevel.String(), log.InfoLevel.String(), log.WarnLevel.String(), log.ErrorLevel.String(), log.FatalLevel.String()}, cobra.ShellCompDirectiveNoFileComp
	})
	root.Flags().BoolVarP(&ver, "version", "V", false, "Print version number and exit")
	root.Flags().BoolVar(&list, "list", false, "Print list of available tasks and exit")
	root.Flags().StringVarP(&from, "from", "f", "file:"+uses.DefaultFileName, "Read location as workflow definition")
	root.Flags().DurationVarP(&timeout, "timeout", "t", time.Hour, "Maximum time allowed for execution")
	root.Flags().BoolVar(&dry, "dry-run", false, "Don't actually run anything; just print")
	root.Flags().StringVarP(&dir, "directory", "C", "", "Change to directory before doing anything")
	_ = root.MarkFlagDirname("directory")
	root.Flags().VarP(&policy, "fetch-policy", "p", fmt.Sprintf(`Set fetch policy ("%s")`, strings.Join(config.AvailablePolicies(), `", "`)))
	_ = root.RegisterFlagCompletionFunc("fetch-policy", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return config.AvailablePolicies(), cobra.ShellCompDirectiveNoFileComp
	})
	root.Flags().StringVarP(&s, "store", "s", "${HOME}/.maru2/store", "Set storage directory")
	_ = root.MarkFlagDirname("store")

	root.CompletionOptions.DisableDefaultCmd = true

	return root
}

// Main executes the root command for the maru2 CLI.
//
// It returns 0 on success, 1 on failure and logs any errors.
func Main() int {
	cli := NewRootCmd()

	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	var logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	styles := log.DefaultStyles()

	// https://github.com/charmbracelet/vhs/blob/main/themes.json
	styles.Levels[log.DebugLevel] = styles.Levels[log.DebugLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#2e7de9", // tokyonight-day blue
		Dark:  "#7aa2f7", // tokyonight blue
	})
	styles.Levels[log.InfoLevel] = styles.Levels[log.InfoLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#007197", // tokyonight-day cyan
		Dark:  "#7dcfff", // tokyonight cyan
	})
	styles.Levels[log.WarnLevel] = styles.Levels[log.WarnLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#8c6c3e", // tokyonight-day amber/yellow
		Dark:  "#e0af68", // tokyonight amber/yellow
	})
	styles.Levels[log.ErrorLevel] = styles.Levels[log.ErrorLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#f52a65", // tokyonight-day red
		Dark:  "#f7768e", // tokyonight red
	})
	styles.Levels[log.FatalLevel] = styles.Levels[log.FatalLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#9854f1", // tokyonight-day magenta (deep red alternative)
		Dark:  "#bb9af7", // tokyonight magenta (deep red alternative)
	})

	logger.SetStyles(styles)

	ctx = log.WithContext(ctx, logger)
	if err := cli.ExecuteContext(ctx); err != nil {
		logger.Print("")
		var tErr *maru2.TraceError
		if errors.As(err, &tErr) && len(tErr.Trace) > 0 {
			trace := tErr.Trace
			slices.Reverse(trace)
			if len(trace) == 1 {
				logger.Error(tErr)
				logger.Error(trace[0])
			} else {
				logger.Error(tErr, "traceback (most recent call first)", strings.Join(trace, "\n"))
			}
		} else {
			logger.Error(err)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		return 1
	}
	return 0
}
