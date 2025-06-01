// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package uses provides a cache+clients for storing and retrieving remote workflows.
package uses

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"

	"github.com/spf13/afero"
)

// Descriptor describes a file to use for caching.
type Descriptor struct {
	Size int64
	Hex  string
}

// IndexFileName is the name of the index file.
const IndexFileName = "index.json"

// Storage interface for storing and retrieving cached remote workflows.
type Storage interface {
	Fetcher
	Exists(uri *url.URL) (bool, error)
	Store(r io.Reader, uri *url.URL) error
}

// LocalStore is a cache for storing and retrieving cached remote workflows from a filesystem.
type LocalStore struct {
	index map[string]Descriptor

	fs afero.Fs

	mu sync.RWMutex
}

// NewLocalStore creates a new store at the given path.
func NewLocalStore(fs afero.Fs) (*LocalStore, error) {
	index := make(map[string]Descriptor, 0)

	_, err := fs.Stat(IndexFileName)
	if os.IsNotExist(err) {
		f, err := fs.Create(IndexFileName)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		_, err = f.WriteString("{}")
		if err != nil {
			return nil, err
		}
		return &LocalStore{
			fs:    fs,
			index: index,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	b, err := afero.ReadFile(fs, IndexFileName)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &index); err != nil {
		return nil, err
	}

	return &LocalStore{
		fs:    fs,
		index: index,
	}, nil
}

// Fetch retrieves a workflow from the store
func (s *LocalStore) Fetch(_ context.Context, uri *url.URL) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	desc, ok := s.index[s.id(uri)]
	if !ok {
		return nil, fmt.Errorf("descriptor not found")
	}

	f, err := s.fs.Open(desc.Hex)
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

	hex := fmt.Sprintf("%x", hasher.Sum(nil))

	if err := afero.WriteFile(s.fs, hex, buf.Bytes(), 0644); err != nil {
		return err
	}

	s.index[s.id(uri)] = Descriptor{
		Size: int64(buf.Len()),
		Hex:  hex,
	}

	b, err := json.Marshal(s.index)
	if err != nil {
		return err
	}

	return afero.WriteFile(s.fs, IndexFileName, b, 0644)
}

// Exists checks if a workflow exists in the store.
func (s *LocalStore) Exists(uri *url.URL) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	desc, ok := s.index[s.id(uri)]
	if !ok {
		return false, nil
	}

	fi, err := s.fs.Stat(desc.Hex)
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

	f, err := s.fs.Open(desc.Hex)
	if err != nil {
		return false, err
	}
	defer f.Close()

	if _, err := io.Copy(hasher, f); err != nil {
		return false, err
	}

	if fmt.Sprintf("%x", hasher.Sum(nil)) != desc.Hex {
		return false, errors.New("hash mismatch")
	}

	return true, nil
}

func (s *LocalStore) id(uri *url.URL) string {
	clone := *uri
	clone.RawQuery = ""
	return clone.String()
}
