// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package cmd provides the root command for the maru2 CLI.
package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
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

	"github.com/a-h/templ"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2"
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/ui"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for the maru2 CLI.
func NewRootCmd() *cobra.Command {
	var (
		w        map[string]string
		level    string
		ver      bool
		list     bool
		filename string
		timeout  time.Duration
		dry      bool
		web      bool
		stdout   io.Writer
		stderr   io.Writer
	)

	root := &cobra.Command{
		Use:   "maru2",
		Short: "A simple task runner",
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			if filename == "" {
				filename = uses.DefaultFileName
			}
			f, err := os.Open(filename)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			defer f.Close()

			wf, err := maru2.ReadAndValidate(f)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
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
					return cmd.GenFishCompletion(os.Stdout, false)
				case "powershell":
					return cmd.GenPowerShellCompletionWithDesc(os.Stdout)
				default:
					return fmt.Errorf("unsupported shell: %s", args[1])
				}
			}

			if filename == "" {
				filename = uses.DefaultFileName
			}

			f, err := os.Open(filename)
			if err != nil {
				return err
			}
			defer f.Close()

			wf, err := maru2.ReadAndValidate(f)
			if err != nil {
				return err
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

			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			fullPath := filepath.Join(cwd, filename)

			rootOrigin := "file:" + fullPath

			ctx = maru2.WithCWDContext(ctx, filepath.Dir(fullPath))

			configDir, err := config.DefaultDirectory()
			if err != nil {
				return err
			}

			loader := &config.FileSystemConfigLoader{
				Fs: afero.NewBasePathFs(afero.NewOsFs(), configDir),
			}

			cfg, err := loader.LoadConfig()
			if err != nil {
				return err
			}

			svc, err := uses.NewFetcherService(
				uses.WithAliases(cfg.Aliases),
				uses.WithClient(&http.Client{
					Timeout: timeout,
				}),
			)
			if err != nil {
				return fmt.Errorf("failed to initialize fetcher service: %w", err)
			}

			if web {
				port := 8080

				// Setup pipes for logger output
				logPr, logPw := io.Pipe()
				logger.SetOutput(logPw)

				// Setup pipes for stdout and stderr
				stdoutPr, stdoutPw := io.Pipe()
				stderrPr, stderrPw := io.Pipe()

				// Assign the writers for Run function
				stdout = stdoutPw
				stderr = stderrPw

				http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					// Create a multi-reader that combines all three streams
					multiReader := io.MultiReader(logPr, stdoutPr, stderrPr)
					reader := bufio.NewReader(multiReader)
					ch := make(chan []byte)
					go func() {
						// Always remember to close the channel.
						defer close(ch)
						for {
							select {
							case <-r.Context().Done():
								return
							default:
								buf := make([]byte, 1024) // chunk size
								n, err := reader.Read(buf)
								if n > 0 {
									chunk := make([]byte, n)
									copy(chunk, buf[:n])
									ch <- chunk
								}
								if err != nil {
									if err == io.EOF {
										return
									}
									logger.Errorf("read error: %v", err)
									return
								}
							}
						}
					}()

					component := ui.Hello(ch)

					templ.Handler(component, templ.WithStreaming()).ServeHTTP(w, r)
				})
				go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
				logger.Info("Started web server", "port", port)
				// <-ctx.Done()
			}

			for _, call := range args {
				start := time.Now()
				logger.Debug("run", "task", call, "from", rootOrigin, "dry-run", dry)
				defer func() {
					logger.Debug("ran", "task", call, "from", rootOrigin, "dry-run", dry, "duration", time.Since(start))
				}()
				_, err := maru2.Run(ctx, svc, wf, call, with, rootOrigin, dry, stdout, stderr)
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
	root.Flags().BoolVarP(&ver, "version", "V", false, "Print version number and exit")
	root.Flags().BoolVar(&list, "list", false, "Print list of available tasks and exit")
	root.Flags().StringVarP(&filename, "file", "f", "", "Read file as workflow definition")
	root.Flags().DurationVarP(&timeout, "timeout", "t", time.Hour, "Maximum time allowed for execution")
	root.Flags().BoolVar(&dry, "dry-run", false, "Don't actually run anything; just print")
	root.Flags().BoolVar(&web, "web", false, "Run maru2 in web mode")

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
