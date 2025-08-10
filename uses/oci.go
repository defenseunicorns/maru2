// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/charmbracelet/log"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// OCIClient fetches workflows from OCI repositories
type OCIClient struct {
	client    remote.Client
	plainHTTP bool
}

// NewOCIClient creates a new ORAS client
func NewOCIClient(baseClient *http.Client, insecureSkipTLSVerify, plainHTTP bool) (*OCIClient, error) {
	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: baseClient.Timeout,
	}

	if baseClient.Transport != nil {
		if transport, ok := baseClient.Transport.(*http.Transport); ok {
			clone := transport.Clone()
			if clone.TLSClientConfig == nil {
				clone.TLSClientConfig = &tls.Config{}
			}
			clone.TLSClientConfig.InsecureSkipVerify = insecureSkipTLSVerify
			httpClient.Transport = clone
		}
	} else {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig.InsecureSkipVerify = insecureSkipTLSVerify
		httpClient.Transport = retry.NewTransport(transport)
	}

	client := &auth.Client{
		Client:     httpClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
	}
	client.SetUserAgent("maru2")
	return &OCIClient{client, plainHTTP}, nil
}

// Fetch uses ORAS to fetch the workflow out of the OCI repository
func (c *OCIClient) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	if uri == nil {
		return nil, fmt.Errorf("uri is nil")
	}

	clone := *uri

	if clone.Scheme != "oci" {
		return nil, fmt.Errorf("scheme is not \"oci\"")
	}

	log.FromContext(ctx).Warnf("THIS FEATURE IS IN ALPHA EXPECT FREQUENT BREAKING CHANGES")

	clone.Scheme = ""
	path := clone.Fragment
	clone.Fragment = ""
	clone.RawQuery = ""

	path, err := url.QueryUnescape(path)
	if err != nil {
		return nil, err
	}

	repo, err := remote.NewRepository(clone.String())
	if err != nil {
		return nil, err
	}
	repo.Client = c.client
	repo.PlainHTTP = c.plainHTTP

	rootDesc, rootReadCloser, err := repo.FetchReference(ctx, clone.String())
	if err != nil {
		return nil, err
	}

	if rootDesc.MediaType != ocispec.MediaTypeImageManifest {
		return nil, fmt.Errorf("unexpected mediatype, want %q got %q", ocispec.MediaTypeImageManifest, rootDesc.MediaType)
	}

	b, err := io.ReadAll(rootReadCloser)
	if err != nil {
		return nil, err
	}

	var manifest ocispec.Manifest

	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, err
	}

	if path == "" {
		path = "file:" + DefaultFileName
	}

	for _, desc := range manifest.Layers {
		if desc.Annotations != nil && desc.Annotations[ocispec.AnnotationTitle] == path {
			return repo.Fetch(ctx, desc)
		}
	}

	return nil, fmt.Errorf("%s: not found", path)
}
