// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"net/http"
	"net/url"
	"sync"
)

// route holds the upstream URL a request is forwarded to plus the service
// name and namespace as metadata.
type route struct {
	url         *url.URL
	serviceName string
	namespace   string
}

// clusterEntry is the per-cluster state: the transport used to reach the remote
// API server, that server's host, and the set of exposed-service routes keyed
// by incoming URL.
type clusterEntry struct {
	transport http.RoundTripper
	host      string
	routes    map[string]route
}

// Store is a concurrency-safe registry of per-cluster transports and routes.
// Freshness of credentials lives inside the RoundTripper, so the store is a
// plain map guarded by a mutex rather than a TTL cache.
type Store struct {
	mu       sync.RWMutex
	clusters map[string]clusterEntry
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{
		clusters: make(map[string]clusterEntry),
	}
}

// SetTransport installs or replaces the transport and API host for a cluster,
// preserving any routes already registered for it.
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

// Host returns the remote API server host for a cluster.
func (s *Store) Host(cluster string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.clusters[cluster]
	if !ok || entry.host == "" {
		return "", false
	}
	return entry.host, true
}

// SetRoutes installs or replaces the routes for a cluster, preserving its
// transport.
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

// Transport returns the transport for a cluster.
func (s *Store) Transport(cluster string) (http.RoundTripper, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.clusters[cluster]
	if !ok || entry.transport == nil {
		return nil, false
	}
	return entry.transport, true
}

// Route returns the route for a cluster and incoming URL.
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

// Delete removes a cluster and all its routes from the store.
func (s *Store) Delete(cluster string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clusters, cluster)
}