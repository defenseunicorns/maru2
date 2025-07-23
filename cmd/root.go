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
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/defenseunicorns/maru2"
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
)

// NewRootCmd creates the root command for the maru2 CLI.
func NewRootCmd() *cobra.Command {
	var (
		w        map[string]string
		level    string
		ver      bool
		list     bool
		from     string
		policy   = config.DefaultFetchPolicy // VarP does not allow you to set a default value
		s        string
		timeout  time.Duration
		dry      bool
		dir      string
		fetchAll bool
		gc       bool
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
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
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

			// default < cfg < flags
			if !cmd.Flags().Changed("fetch-policy") {
				if err := policy.Set(cfg.FetchPolicy.String()); err != nil {
					return err
				}
			}

			if policy == config.FetchPolicyNever && fetchAll {
				return fmt.Errorf("cannot fetch all with fetch policy %q", policy)
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

			// if we are a sub-command, load the cfg as PersistentPreRun isnt run
			// when performing tab completions on sub-commands
			if cmd.Parent() != nil {
				configDir, err := config.DefaultDirectory()
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}

				loader := &config.FileSystemConfigLoader{
					Fs: afero.NewBasePathFs(afero.NewOsFs(), configDir),
				}

				cfg, err = loader.LoadConfig()
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
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
				switch bi.Main.Path {
				case "github.com/defenseunicorns/maru2":
					fmt.Fprintln(os.Stdout, bi.Main.Version)
				default:
					for _, dep := range bi.Deps {
						if dep.Path == "github.com/defenseunicorns/maru2" {
							fmt.Fprintln(os.Stdout, dep.Version)
							break
						}
					}
				}
				return nil
			}

			// fix fish needing "'pkg:...'" for tab completion
			from = strings.Trim(from, `"`)
			from = strings.Trim(from, `'`)

			fs := afero.NewOsFs()

			createDir := true
			if !cmd.Flags().Changed("store") {
				localStorePath := ".maru2/store"
				if fi, err := fs.Stat(localStorePath); err == nil && fi.IsDir() {
					s = localStorePath
					createDir = false
				}
			}

			s = filepath.Clean(os.ExpandEnv(s))
			if s == "." {
				s = ".maru2/store"
			}

			if createDir {
				if err := fs.MkdirAll(s, 0o744); err != nil {
					return err
				}
			}

			store, err := uses.NewLocalStore(afero.NewBasePathFs(fs, s))
			if err != nil {
				return fmt.Errorf("failed to initialize store: %w", err)
			}

			svc, err := uses.NewFetcherService(
				uses.WithAliases(cfg.Aliases),
				uses.WithStorage(store),
				uses.WithFetchPolicy(policy),
			)
			if err != nil {
				return fmt.Errorf("failed to initialize fetcher service: %w", err)
			}

			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
				cmd.SetContext(ctx)
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

				logger.Print("Available:\n")
				for _, n := range names {
					logger.Printf("- %s", n)
				}

				return nil
			}

			if fetchAll {
				logger.Debug("fetching all", "tasks", wf.Tasks.OrderedTaskNames(), "from", resolved)
				if err := maru2.FetchAll(ctx, svc, wf, resolved); err != nil {
					return err
				}
			}

			with := make(maru2.With, len(w))
			for k, v := range w {
				with[k] = v
			}

			if len(args) == 0 {
				args = append(args, maru2.DefaultTaskName)
			}

			for _, call := range args {
				_, err := maru2.Run(ctx, svc, wf, call, with, resolved, dry)
				if err != nil {
					return err
				}
			}

			if gc {
				return store.GC()
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
	root.Flags().BoolVar(&gc, "gc", false, "Perform garbage collection on the store")
	root.Flags().BoolVar(&fetchAll, "fetch-all", false, "Fetch all tasks")

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

	logger.SetStyles(DefaultStyles())

	ctx = log.WithContext(ctx, logger)
	cmd, err := cli.ExecuteContextC(ctx)
	if err != nil {
		logger.Print("")

		if errors.Is(cmd.Context().Err(), context.DeadlineExceeded) {
			logger.Error("task timed out")
		}

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
	}
	return ParseExitCode(err)
}

// ParseExitCode calculates the exit code from a given error
//
// 0 - the error was nil
// 1 - there was some error
// n - the underlying error from an exec.Command
func ParseExitCode(err error) int {
	if err == nil {
		return 0
	}

	var eErr *exec.ExitError
	if errors.As(err, &eErr) {
		if status, ok := eErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	return 1
}
