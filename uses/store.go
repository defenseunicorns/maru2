// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package uses provides a cache+clients for storing and retrieving remote workflows.
package uses

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/afero"
)

// DigestPattern is the regexp for a digest entry in an index
var DigestPattern = regexp.MustCompile(`^h1:([a-fA-F0-9]{64})$`)

// Descriptor describes a file to use for caching.
type Descriptor struct {
	Size int64
	Hex  string
}

// IndexFileName is the name of the index file.
const IndexFileName = "index.txt"

// Storage interface for storing and retrieving cached remote workflows.
type Storage interface {
	Fetcher
	Exists(uri *url.URL) (bool, error)
	Store(r io.Reader, uri *url.URL) error
	List() iter.Seq2[string, Descriptor]
}

// LocalStore is a cache for storing and retrieving cached remote workflows from a filesystem.
type LocalStore struct {
	index map[string]Descriptor

	fsys afero.Fs

	mu sync.RWMutex
}

// NewLocalStore creates a new store at the given path.
func NewLocalStore(fsys afero.Fs) (*LocalStore, error) {
	index := make(map[string]Descriptor, 0)

	_, err := fsys.Stat(IndexFileName)
	if os.IsNotExist(err) {
		f, err := fsys.Create(IndexFileName)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		return &LocalStore{
			fsys:    fsys,
			index: index,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	f, err := fsys.Open(IndexFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	index, err = ParseIndex(f)
	if err != nil {
		return nil, err
	}

	return &LocalStore{
		fsys:    fsys,
		index: index,
	}, nil
}

// ParseIndex parses an index file.
func ParseIndex(r io.Reader) (map[string]Descriptor, error) {
	index := make(map[string]Descriptor, 0)

	scanner := bufio.NewScanner(bufio.NewReader(r))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var desc Descriptor
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid line format")
		}
		var err error
		desc.Size, err = strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return nil, err
		}
		matches := DigestPattern.FindStringSubmatch(fields[1])
		if len(matches) < 2 {
			return nil, fmt.Errorf("invalid digest format or unable to extract hex: %s", fields[1])
		}
		desc.Hex = matches[1]

		_, err = url.Parse(fields[0])
		if err != nil {
			return nil, err
		}

		index[fields[0]] = desc
	}

	return index, nil
}

// Fetch retrieves a workflow from the store
func (s *LocalStore) Fetch(_ context.Context, uri *url.URL) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	desc, ok := s.index[s.id(uri)]
	if !ok {
		return nil, fmt.Errorf("descriptor not found")
	}

	f, err := s.fsys.Open(desc.Hex)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// Store a workflow in the store.
func (s *LocalStore) Store(rc io.Reader, uri *url.URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hasher := sha256.New()

	var buf bytes.Buffer

	mw := io.MultiWriter(hasher, &buf)

	if _, err := io.Copy(mw, rc); err != nil {
		return err
	}

	encoded := hex.EncodeToString(hasher.Sum(nil))

	if err := afero.WriteFile(s.fsys, encoded, buf.Bytes(), 0o644); err != nil {
		return err
	}

	s.index[s.id(uri)] = Descriptor{
		Size: int64(buf.Len()),
		Hex:  encoded,
	}

	keys := make([]string, 0, len(s.index))
	for key := range s.index {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var b []byte
	for _, key := range keys {
		desc := s.index[key]
		b = fmt.Appendf(b, "%s h1:%s %d\n", key, desc.Hex, desc.Size)
	}

	return afero.WriteFile(s.fsys, IndexFileName, b, 0o644)
}

// Exists checks if a workflow exists in the store.
func (s *LocalStore) Exists(uri *url.URL) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	desc, ok := s.index[s.id(uri)]
	if !ok {
		return false, nil
	}

	fi, err := s.fsys.Stat(desc.Hex)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("descriptor exists in index, but no corresponding file was found, possible cache corruption: %s", desc.Hex)
		}
		return false, err
	}

	if fi.Size() != desc.Size {
		return false, fmt.Errorf("size mismatch, expected %d, got %d", desc.Size, fi.Size())
	}

	hasher := sha256.New()

	f, err := s.fsys.Open(desc.Hex)
	if err != nil {
		return false, err
	}
	defer f.Close()

	if _, err := io.Copy(hasher, f); err != nil {
		return false, err
	}

	if hex.EncodeToString(hasher.Sum(nil)) != desc.Hex {
		return false, errors.New("hash mismatch")
	}

	return true, nil
}

// List returns a Go 1.23+ iterator to loop over all of the stored workflows
//
// ok but does this really need to be an iterator, no
// ill prob move it to a regular map access w/ maps.Copy, but this was still fun
func (s *LocalStore) List() iter.Seq2[string, Descriptor] {
	return func(yield func(string, Descriptor) bool) {
		for k, v := range s.index {
			if !yield(k, v) {
				return
			}
		}
	}
}

// GC performs garbage collection on the store.
func (s *LocalStore) GC() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := afero.ReadDir(s.fsys, ".")
	if err != nil {
		return err
	}

outer:
	for _, fi := range all {
		if fi.IsDir() || fi.Name() == "index.txt" {
			continue
		}

		for _, desc := range s.index {
			if desc.Hex == fi.Name() {
				continue outer
			}
		}
		if err := s.fsys.Remove(fi.Name()); err != nil {
			return err
		}
	}

	return nil
}

func (s *LocalStore) id(uri *url.URL) string {
	clone := *uri
	clone.RawQuery = ""
	clone.User = nil
	return clone.String()
}
