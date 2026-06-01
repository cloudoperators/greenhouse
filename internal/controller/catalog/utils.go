// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"fmt"
	"hash/fnv"
	"net/url"
	"strconv"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"

	greenhousev1alpha1 "github.com/cloudoperators/greenhouse/api/v1alpha1"
)

// HashValue - returns a hash of the string
func HashValue(value string) (string, error) {
	h := fnv.New64a()
	_, err := h.Write([]byte(value))
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(h.Sum64(), 10), nil
}

// GetOwnerRepoInfo - extracts host, owner, repo from git repository URL
// host is transformed to replace '.' with '-' to be used as source group in Catalog.Status.Inventory
func GetOwnerRepoInfo(s string) (host, owner, repo string, err error) {
	u, err := url.Parse(s)
	if err != nil {
		err = fmt.Errorf("failed parsing URL %q: %w", s, err)
		return
	}
	id := strings.TrimLeft(u.Path, "/")
	id = strings.TrimSuffix(id, ".git")
	comp := strings.Split(id, "/")
	if len(comp) != 2 {
		err = fmt.Errorf("invalid repository id %q", id)
		return
	}
	host = strings.ReplaceAll(u.Host, ".", "-")
	owner = comp[0]
	repo = comp[1]
	return
}

func getGitRepositoryReference(s *greenhousev1alpha1.CatalogSource) *sourcev1.GitRepositoryRef {
	gitReference := &sourcev1.GitRepositoryRef{}
	if s.Ref != nil {
		// flux precedence 1
		if s.Ref.SHA != nil {
			gitReference.Commit = *s.Ref.SHA
		}
		// flux precedence 2
		if s.Ref.Tag != nil {
			gitReference.Tag = *s.Ref.Tag
		}
		// flux precedence 3
		if s.Ref.Branch != nil {
			gitReference.Branch = *s.Ref.Branch
		}
	}
	return gitReference
}
