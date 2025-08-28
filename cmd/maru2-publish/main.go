// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/defenseunicorns/maru2"
	"github.com/defenseunicorns/maru2/cmd"
	"github.com/defenseunicorns/maru2/config"
	configv0 "github.com/defenseunicorns/maru2/config/v0"
)

func main() {
	code := Main()
	os.Exit(code)
}

// Main executes the root command for the maru2-publish CLI.
//
// It returns 0 on success, 1 on failure and logs any errors.
func Main() int {
	var (
		level           string
		ver             bool
		plainHTTP       bool
		insecureSkipTLS bool
		dir             string
		entrypoints     []string
	)

	root := &cobra.Command{
		Use:   "maru2-publish",
		Short: "Pack a maru2 workflow into an OCI artifact and publish",
		Args: func(cmd *cobra.Command, args []string) error {
			if ver && len(args) == 0 {
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			l, err := log.ParseLevel(level)
			if err != nil {
				return err
			}
			logger := log.FromContext(cmd.Context())
			logger.SetLevel(l)

			return nil
		},
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

			logger.Warnf("THIS FEATURE IS IN ALPHA EXPECT FREQUENT BREAKING CHANGES")

			if dir != "" {
				if err := os.Chdir(dir); err != nil {
					return err
				}
			}

			configDir, err := config.DefaultDirectory()
			if err != nil {
				return err
			}

			cfg, err := configv0.LoadConfig(afero.NewBasePathFs(afero.NewOsFs(), configDir))
			if err != nil {
				return err
			}

			ref, err := registry.ParseReference(args[0])
			if err != nil {
				return fmt.Errorf("unable to parse reference: %w", err)
			}
			if err := ref.ValidateReferenceAsTag(); err != nil {
				return fmt.Errorf("reference is not a tag: %w", err)
			}

			dst := &remote.Repository{
				Reference: ref,
				PlainHTTP: plainHTTP,
			}
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.TLSClientConfig.InsecureSkipVerify = insecureSkipTLS

			storeOpts := credentials.StoreOptions{}
			credStore, err := credentials.NewStoreFromDocker(storeOpts)
			if err != nil {
				return err
			}

			client := &auth.Client{
				Client:     &http.Client{Transport: retry.NewTransport(transport)},
				Cache:      auth.NewCache(),
				Credential: credentials.Credential(credStore),
			}
			client.SetUserAgent("maru2-publish")
			dst.Client = client

			return maru2.Publish(ctx, dst, entrypoints, cfg.Aliases)
		},
	}

	root.Flags().StringVarP(&level, "log-level", "l", "info", "Set log level")
	_ = root.RegisterFlagCompletionFunc("log-level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{log.DebugLevel.String(), log.InfoLevel.String(), log.WarnLevel.String(), log.ErrorLevel.String(), log.FatalLevel.String()}, cobra.ShellCompDirectiveNoFileComp
	})
	root.Flags().BoolVarP(&ver, "version", "V", false, "Print version number and exit")
	root.Flags().BoolVar(&plainHTTP, "plain-http", false, "Force the connections over HTTP instead of HTTPS")
	root.Flags().BoolVar(&insecureSkipTLS, "insecure-skip-tls-verify", false, "Allow connections to SSL registry without certs")
	root.Flags().StringVarP(&dir, "directory", "C", "", "Change to directory before doing anything")
	_ = root.MarkFlagDirname("directory")
	root.Flags().StringSliceVarP(&entrypoints, "entrypoint", "e", entrypoints, "Slice(s) of relative paths to workflows")

	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	logger.SetStyles(cmd.DefaultStyles())

	ctx = log.WithContext(ctx, logger)

	if err := root.ExecuteContext(ctx); err != nil {
		logger.Error(err)
		return 1
	}
	return 0
}
