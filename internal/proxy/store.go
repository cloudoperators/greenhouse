// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"net/http"
	"net/url"
	"sync"
)

type route struct {
	url         *url.URL
	serviceName string
	namespace   string
}

type clusterEntry struct {
	transport http.RoundTripper
	host      string
	routes    map[string]route
}

type Store struct {
	mu       sync.RWMutex
	clusters map[string]clusterEntry
}

func NewStore() *Store {
	return &Store{
		clusters: make(map[string]clusterEntry),
	}
}

func (s *Store) SetTransport(cluster string, transport http.RoundTripper, host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.clusters[cluster]
	if !ok {
		entry = clusterEntry{routes: make(map[string]route)}
	}
	entry.transport = transport
	entry.host = host
	s.clusters[cluster] = entry
}

func (s *Store) Host(cluster string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.clusters[cluster]
	if !ok || entry.host == "" {
		return "", false
	}
	return entry.host, true
}

func (s *Store) SetRoutes(cluster string, routes map[string]route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.clusters[cluster]
	if !ok {
		entry = clusterEntry{}
	}
	entry.routes = routes
	s.clusters[cluster] = entry
}

func (s *Store) Transport(cluster string) (http.RoundTripper, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.clusters[cluster]
	if !ok || entry.transport == nil {
		return nil, false
	}
	return entry.transport, true
}

func (s *Store) Route(cluster, inURL string) (route, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.clusters[cluster]
	if !ok {
		return route{}, false
	}
	r, ok := entry.routes[inURL]
	return r, ok
}

func (s *Store) Delete(cluster string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clusters, cluster)
}
