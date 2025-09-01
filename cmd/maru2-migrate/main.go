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

	"github.com/defenseunicorns/maru2/schema"
	v0 "github.com/defenseunicorns/maru2/schema/v0"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/goccy/go-yaml"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) > 1 {
		fatal(fmt.Errorf("must provide path(s) to migrate"))
	}

	var to string
	flag.StringVar(&to, "to", v1.SchemaVersion, "version to migrate to")

	flag.Parse()

	paths := os.Args[1:]

	ctx := context.Background()

	fs := afero.NewOsFs()
	for _, p := range paths {
		err := migrate(ctx, fs, p)
		if err != nil {
			fatal(err)
		}
	}
}

func migrate(ctx context.Context, fs afero.Fs, p string) error {
	uri := &url.URL{
		Scheme: "file",
		Opaque: p,
	}
	fetcher := uses.NewLocalFetcher(fs)

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
		b, err := pretty(wf)
		if err != nil {
			return err
		}
		return atomicWriteAndBackup(p, b)
	default:
		return fmt.Errorf("unsupported schema version: %q", version)
	}
}

func pretty(wf v1.Workflow) ([]byte, error) {
	b, err := yaml.MarshalWithOptions(wf, yaml.Indent(2), yaml.IndentSequence(true), yaml.UseLiteralStyleIfMultiline(true))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// going to comment every fuction in this guy cause this is a complex operation
func atomicWriteAndBackup(p string, b []byte) error {
	if filepath.IsAbs(p) {
		return fmt.Errorf("%s cannot be absolute")
	}

	// create a temp file to write to
	tmp, err := os.CreateTemp("", "")
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

	if err := unix.Renameat2(unix.AT_FDCWD, src.Name(), unix.AT_FDCWD, bak.Name(), unix.RENAME_EXCHANGE); err != nil {
		return fmt.Errorf("failed swapping %s and %s: %w", src.Name(), bak.Name(), err)
	}

	return unix.Renameat2(int(tmp.Fd()), tmp.Name(), unix.AT_FDCWD, src.Name(), unix.RENAME_EXCHANGE)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v", err)
	os.Exit(1)
}
