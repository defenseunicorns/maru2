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
	configv0 "github.com/defenseunicorns/maru2/config/v0"
	"github.com/defenseunicorns/maru2/schema"
	"github.com/defenseunicorns/maru2/uses"
)

// NewRootCmd creates the root command for the maru2 CLI.
func NewRootCmd() *cobra.Command {
	var (
		w          map[string]string
		withFile   string
		level      string
		ver        bool
		list       bool
		explain    bool
		from       string
		policy     = uses.DefaultFetchPolicy // VarP does not allow you to set a default value
		s          string
		timeout    time.Duration
		dry        bool
		dir        string
		configPath string
		fetchAll   bool
		gc         bool
	)

	var cfg *configv0.Config // cfg is not set via CLI flag

	// closure initializer
	loadConfig := func(cmd *cobra.Command) error {
		switch {
		case cmd.Flags().Changed("config"):
			f, err := os.Open(configPath)
			if err != nil {
				return fmt.Errorf("failed to open config file: %w", err)
			}
			defer f.Close()
			cfg, err = configv0.LoadConfig(f)
			if err != nil {
				return fmt.Errorf("failed to load config file: %w", err)
			}
		case os.Getenv("MARU2_CONFIG") != "":
			f, err := os.Open(os.Getenv("MARU2_CONFIG"))
			if err != nil {
				return fmt.Errorf("failed to open config file: %w", err)
			}
			defer f.Close()
			cfg, err = configv0.LoadConfig(f)
			if err != nil {
				return fmt.Errorf("failed to load config file: %w", err)
			}
		default:
			var err error
			cfg, err = configv0.LoadDefaultConfig()
			if err != nil {
				return err
			}
		}

		// default < cfg < flags
		if !cmd.Flags().Changed("fetch-policy") && cfg.FetchPolicy != policy {
			if err := policy.Set(cfg.FetchPolicy.String()); err != nil {
				return err // since config validates and has defaults during loading, this error is basically impossible to trigger, but leaving in case a regression happens in schema validation
			}
		}

		if policy == uses.FetchPolicyNever && fetchAll {
			return fmt.Errorf("cannot fetch all with fetch policy %q", policy)
		}

		return nil
	}

	root := &cobra.Command{
		Use:   "maru2",
		Short: "A simple task runner",
		Long: `
 ███╗   ███╗ █████╗ ██████╗ ██╗   ██╗██████╗
 ████╗ ████║██╔══██╗██╔══██╗██║   ██║╚════██╗
 ██╔████╔██║███████║██████╔╝██║   ██║ █████╔╝
 ██║╚██╔╝██║██╔══██║██╔══██╗██║   ██║██╔═══╝
 ██║ ╚═╝ ██║██║  ██║██║  ██║╚██████╔╝███████╗
 ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝
`,
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

			return loadConfig(cmd)
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
				if err := loadConfig(cmd); err != nil {
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

			names := make([]string, 0, len(wf.Tasks))
			for _, name := range wf.Tasks.OrderedTaskNames() {
				names = append(names, strings.Join([]string{name, wf.Tasks[name].Description}, "\t"))
			}

			for name, alias := range wf.Aliases {
				if alias.Path != "" {
					next, err := uses.ResolveRelative(resolved, strings.Join([]string{"file", alias.Path}, ":"), wf.Aliases)
					if err != nil {
						return nil, cobra.ShellCompDirectiveError
					}
					aliasedWF, err := maru2.Fetch(cmd.Context(), svc, next)
					if err != nil {
						return nil, cobra.ShellCompDirectiveError
					}
					for _, n := range aliasedWF.Tasks.OrderedTaskNames() {
						names = append(names, strings.Join([]string{fmt.Sprintf("%s:%s", name, n), wf.Tasks[name].Description}, "\t"))
					}
				}
			}

			return names, cobra.ShellCompDirectiveNoFileComp
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
				t, err := maru2.NewDetailedTaskList(ctx, svc, resolved, wf)
				if err != nil {
					return err
				}

				fmt.Fprintln(os.Stdout, "Available tasks:")
				fmt.Fprintln(os.Stdout, t)

				return nil
			}

			if explain {
				md, err := maru2.Explain(wf, args...)
				if err != nil {
					return err
				}
				fmt.Fprintln(os.Stdout, md)
				return nil
			}

			if fetchAll {
				logger.Debug("fetching all", "tasks", wf.Tasks.OrderedTaskNames(), "from", resolved)
				if err := maru2.FetchAll(ctx, svc, wf, resolved); err != nil {
					return err
				}
				// allow no args w/ fetch all
				if len(args) == 0 {
					if gc {
						return store.GC()
					}
					return nil
				}
			}

			with := make(schema.With, len(w))
			for k, v := range w {
				with[k] = v
			}

			if withFile != "" {
				f, err := fs.Open(withFile)
				if err != nil {
					return fmt.Errorf("failed opening with-file %q: %w", withFile, err)
				}
				defer f.Close()
				outputs, err := maru2.ParseOutput(f)
				if err != nil {
					return fmt.Errorf("failed reading with-file %q: %w", withFile, err)
				}
				for k, v := range outputs {
					_, ok := with[k]
					if !ok { // CLI --with takes priority
						with[k] = v
					}
				}
			}

			if len(args) == 0 {
				args = append(args, schema.DefaultTaskName)
			}

			opts := maru2.RuntimeOptions{
				Dry: dry,
				Env: os.Environ(),
			}

			for _, call := range args {
				parts := strings.SplitN(call, ":", 2)

				if len(parts) == 2 {
					next, err := uses.ResolveRelative(resolved, call, wf.Aliases)
					if err != nil {
						return err
					}
					nextWf, err := maru2.Fetch(ctx, svc, next)
					if err != nil {
						return err
					}

					_, err = maru2.Run(ctx, svc, nextWf, parts[1], with, next, opts)
					if err != nil {
						return err
					}
					continue
				}

				_, err := maru2.Run(ctx, svc, wf, call, with, resolved, opts)
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
	root.Flags().StringVar(&withFile, "with-file", "", "Extra text file to parse as key=value pairs to pass to the called task(s)")
	_ = root.MarkFlagFilename("with-file", "txt")
	root.Flags().StringVarP(&level, "log-level", "l", "info", "Set log level")
	_ = root.RegisterFlagCompletionFunc("log-level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{log.DebugLevel.String(), log.InfoLevel.String(), log.WarnLevel.String(), log.ErrorLevel.String(), log.FatalLevel.String()}, cobra.ShellCompDirectiveNoFileComp
	})
	root.Flags().BoolVarP(&ver, "version", "V", false, "Print version number and exit")
	root.Flags().BoolVar(&list, "list", false, "Print list of available tasks and exit")
	root.Flags().BoolVar(&explain, "explain", false, "Print explanation of workflow/task(s) and exit")
	root.Flags().StringVarP(&from, "from", "f", "file:"+uses.DefaultFileName, "Read location as workflow definition")
	root.Flags().DurationVarP(&timeout, "timeout", "t", time.Hour, "Maximum time allowed for execution")
	root.Flags().BoolVar(&dry, "dry-run", false, "Don't actually run anything; just print")
	root.Flags().StringVarP(&dir, "directory", "C", "", "Change to directory before doing anything")
	_ = root.MarkFlagDirname("directory")
	root.Flags().StringVarP(&configPath, "config", "", "${HOME}/.maru2/config.yaml", "Path to maru2 config file") // mirrors config.DefaultDirectory
	_ = root.MarkFlagFilename("config", "yaml", "yml")
	root.Flags().VarP(&policy, "fetch-policy", "p", fmt.Sprintf(`Set fetch policy ("%s")`, strings.Join(uses.AvailablePolicies(), `", "`)))
	_ = root.RegisterFlagCompletionFunc("fetch-policy", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return uses.AvailablePolicies(), cobra.ShellCompDirectiveNoFileComp
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

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM)
	defer cancel()

	logger := log.NewWithOptions(os.Stderr, log.Options{
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
			if status.Exited() {
				return status.ExitStatus()
			}
			if status.Signaled() {
				if status.Signal() == syscall.SIGINT {
					return 130
				}
			}
		}
	}
	return 1
}
