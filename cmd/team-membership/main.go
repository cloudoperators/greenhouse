// Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"os"

	"k8s.io/client-go/rest"

	"github.com/cloudoperators/greenhouse/pkg/scim"
)

const (
	scimBasicAuthUserEnvKey string = "SCIM_BASIC_AUTH_USER"
	scimBasicAuthPwEnvKey   string = "SCIM_BASIC_AUTH_PW" //nolint:gosec
	scimBaseURLEnvKey       string = "SCIM_BASE_URL"      //nolint:gosec
	namespaceEnvKey         string = "NAMESPACE"
)

func main() {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Printf(`Failed to get inCluster config: %s`, err)
		os.Exit(1)
	}

	k8sClient, err := NewK8sClient(k8sConfig)
	if err != nil {
		log.Printf(`Failed to create k8s client: %s`, err)
		os.Exit(1)
	}

	scimBaseURL := os.Getenv(scimBaseURLEnvKey)
	if scimBaseURL == "" {
		log.Printf(`%s needs to be set for running the scim client`, scimBaseURLEnvKey)
		os.Exit(1)
	}
	scimBasicAuthUser := os.Getenv(scimBasicAuthUserEnvKey)
	if scimBaseURL == "" {
		log.Printf(`%s needs to be set for running the scim client`, scimBasicAuthUserEnvKey)
		os.Exit(1)
	}
	scimBasicAuthPw := os.Getenv(scimBasicAuthPwEnvKey)
	if scimBaseURL == "" {
		log.Printf(`%s needs to be set for running the scim client`, scimBasicAuthPwEnvKey)
		os.Exit(1)
	}
	scimConfig := scim.Config{RawURL: scimBaseURL, AuthType: scim.Basic, BasicAuthConfig: &scim.BasicAuthConfig{BasicAuthUser: scimBasicAuthUser, BasicAuthPw: scimBasicAuthPw}}
	scimClient, err := scim.NewScimClient(scimConfig)
	if err != nil {
		log.Printf(`Failed to create scim client: %s`, err)
		os.Exit(1)
	}

	namespace := os.Getenv("NAMESPACE")
	teamUpdater := NewTeamMembershipUpdater(k8sClient, *scimClient, namespace)

	err = teamUpdater.DoUpdates()
	if err != nil {
		log.Printf(`Error updating teams: %s`, err)
		os.Exit(1)
	}
}
