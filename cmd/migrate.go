// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/defenseunicorns/maru2/schema/migrate"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

// NewMigrateCmd creates a new migrate command
func NewMigrateCmd() *cobra.Command {
	var to string

	cmd := &cobra.Command{
		Use:   "maru2-migrate",
		Short: "Migrate a maru2 workflow to a new schema version",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := log.FromContext(cmd.Context())

			for _, p := range args {
				logger.Info("migrating", "path", p, "to", to)
				err := migrate.Path(cmd.Context(), p, v1.SchemaVersion)
				if err != nil {
					return err
				}
				logger.Info("migrated ", "path", p, "backup", p+".bak")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", v1.SchemaVersion, "version to migrate to")

	return cmd
}
