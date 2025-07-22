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
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/defenseunicorns/maru2"
	"github.com/defenseunicorns/maru2/cmd"
	"github.com/defenseunicorns/maru2/config"
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
		plainHTTP       bool
		insecureSkipTLS bool
		dir             string
		entrypoints     []string
	)

	root := &cobra.Command{
		Use:           "maru2-publish",
		Short:         "Pack a maru2 workflow into an OCI artifact and publish",
		Args:          cobra.ExactArgs(1),
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

			loader := &config.FileSystemConfigLoader{
				Fs: afero.NewBasePathFs(afero.NewOsFs(), configDir),
			}

			cfg, err := loader.LoadConfig()
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

			dst, err := remote.NewRepository(ref.String())
			if err != nil {
				return err
			}
			dst.PlainHTTP = plainHTTP
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.TLSClientConfig.InsecureSkipVerify = insecureSkipTLS
			dst.Client = &http.Client{
				Transport: retry.NewTransport(transport),
			}

			return maru2.Publish(ctx, cfg, dst, entrypoints)
		},
	}

	root.Flags().StringVarP(&level, "log-level", "l", "info", "Set log level")
	_ = root.RegisterFlagCompletionFunc("log-level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{log.DebugLevel.String(), log.InfoLevel.String(), log.WarnLevel.String(), log.ErrorLevel.String(), log.FatalLevel.String()}, cobra.ShellCompDirectiveNoFileComp
	})
	root.Flags().BoolVar(&plainHTTP, "plain-http", false, "Allow insecure connections to registry without SSL check")
	root.Flags().BoolVar(&insecureSkipTLS, "insecure-skip-tls-verify", false, "Allow connections to SSL registry without certs")
	root.Flags().StringVarP(&dir, "directory", "C", "", "Change to directory before doing anything")
	_ = root.MarkFlagDirname("directory")
	root.Flags().StringSliceVarP(&entrypoints, "entrypoint", "e", entrypoints, "Slice(s) of relative paths to workflows")

	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	var logger = log.NewWithOptions(os.Stderr, log.Options{
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
