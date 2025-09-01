// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/goccy/go-yaml"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"

	maru2cmd "github.com/defenseunicorns/maru2/cmd"
	"github.com/defenseunicorns/maru2/schema"
	v0 "github.com/defenseunicorns/maru2/schema/v0"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

func main() {
	ctx := context.Background()
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		Level:           log.DebugLevel,
	})
	logger.SetStyles(maru2cmd.DefaultStyles())
	ctx = log.WithContext(ctx, logger)

	if len(os.Args) < 2 {
		logger.Fatal("must provide path(s) to migrate")
	}

	var to string
	flag.StringVar(&to, "to", v1.SchemaVersion, "version to migrate to")

	flag.Parse()

	paths := os.Args[1:]

	for _, p := range paths {
		err := migrate(ctx, p)
		if err != nil {
			logger.Fatal(err)
		}
		logger.Info("migrated and backed up", "path", p, "backup", p+".bak")
	}
}

func migrate(ctx context.Context, p string) error {
	uri := &url.URL{
		Scheme: "file",
		Opaque: p,
	}
	fetcher := uses.NewLocalFetcher(afero.NewOsFs())

	rc, err := fetcher.Fetch(ctx, uri)
	if err != nil {
		return err
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	var versioned schema.Versioned
	if err := yaml.Unmarshal(b, &versioned); err != nil {
		return err
	}
	switch version := versioned.SchemaVersion; version {
	case v1.SchemaVersion:
		return nil
	case v0.SchemaVersion:
		var v0Workflow v0.Workflow
		if err := yaml.Unmarshal(b, &v0Workflow); err != nil {
			return err
		}
		wf, err := v1.Migrate(v0Workflow)
		if err != nil {
			return err
		}
		prefix := []byte("# yaml-language-server: $schema=https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json\n")
		b, err := pretty(wf, prefix)
		if err != nil {
			return err
		}
		return atomicWriteAndBackup(p, b)
	default:
		return fmt.Errorf("unsupported schema version: %q", version)
	}
}

func pretty(wf v1.Workflow, prefix []byte) ([]byte, error) {
	b, err := yaml.MarshalWithOptions(wf, yaml.Indent(2), yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true), yaml.UseSingleQuote(false))
	if err != nil {
		return nil, err
	}
	return append(prefix, b...), nil
}

// going to comment every fuction in this guy cause this is a complex operation
func atomicWriteAndBackup(p string, b []byte) error {
	if filepath.IsAbs(p) {
		return fmt.Errorf("%s cannot be absolute", p)
	}

	// create a temp file to write to
	tmp, err := os.Create(p + ".tmp")
	if err != nil {
		return err
	}
	defer func() {
		// ignore cleanup errors, since a successful operation destroys the temp file
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	// write the bytes to the temp file
	_, err = tmp.Write(b)
	if err != nil {
		return err
	}

	// get a file pointer to the original workflow
	src, err := os.Open(p)
	if err != nil {
		return err
	}

	// grab src file info, to perform checks and ensure the tmp file has the same perms
	info, err := os.Stat(p)
	if err != nil {
		return err
	}

	// only handle regular files, fail otherwise
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a path to a regular file", p)
	}

	// ensure the tmp file has the same permissions as the src (not sure if Renameat2 handles this)
	if err := os.Chmod(tmp.Name(), info.Mode()); err != nil {
		return err
	}

	// create a file pointer to backup the original workflow to
	bak, err := os.Create(p + ".bak")
	if err != nil {
		return err
	}

	// atomic rename src -> src.bak
	if err := unix.Renameat2(unix.AT_FDCWD, src.Name(), unix.AT_FDCWD, bak.Name(), unix.RENAME_EXCHANGE); err != nil {
		return fmt.Errorf("failed swapping %s and %s: %w", src.Name(), bak.Name(), err)
	}

	// atomic rename tmp -> src
	if err := unix.Renameat2(unix.AT_FDCWD, tmp.Name(), unix.AT_FDCWD, src.Name(), unix.RENAME_EXCHANGE); err != nil {
		return fmt.Errorf("failed swapping %s and %s: %w", tmp.Name(), src.Name(), err)
	}
	return nil
}
