// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package tailscalestarter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
)

func fileExists(fileName string) bool {
	fileInfo, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return !fileInfo.IsDir()
}

func readFile(fileName string) string {
	buff, err := os.ReadFile(fileName)
	if err != nil {
		return ""
	}
	return string(buff)
}

func setPreAuthKey() error {
	switch {
	case os.Getenv("TS_AUTHKEY") != "", os.Getenv("TS_AUTH_KEY") != "":
		log.FromContext(ctrl.SetupSignalHandler()).Info("TS_AUTHKEY or TS_AUTH_KEY is set, skipping preauthkey")
		return nil
	case fileExists("/preauthkey/key"):
		if err := os.Setenv("TS_AUTHKEY", readFile("/preauthkey/key")); err != nil {
			return err
		}
		return nil
	default:
		return errors.New("no preauthkey found, stopping tailscale")
	}
}

func isRunningOrStarting(status *ipnstate.Status) (description string, ok bool) {
	switch status.BackendState {
	case ipn.Stopped.String():
		return "Tailscale is stopped.", false
	case ipn.NeedsLogin.String():
		return "Logged out.", false
	case ipn.NeedsMachineAuth.String():
		return "Client needs to be approved", false
	case ipn.Running.String(), ipn.Starting.String():
		return status.BackendState, true
	default:
		return "unknown state: " + status.BackendState, false
	}
}

type healthResponse struct {
	TailscaleVersion *string  `json:"tailscaleVersion,omitempty"`
	ErrorMessage     *string  `json:"errorMessage,omitempty"`
	HealthStatus     []string `json:"healthStatus,omitempty"`
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	response := new(healthResponse)
	w.Header().Set("Content-Type", "application/json")
	localClient.UseSocketOnly = true
	status, err := localClient.StatusWithoutPeers(context.Background())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.HealthStatus = append(response.HealthStatus, err.Error())
	}
	description, ok := isRunningOrStarting(status)
	if ok {
		w.WriteHeader(http.StatusOK)
		response.TailscaleVersion = &status.Version
		response.HealthStatus = append(response.HealthStatus, description)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		if len(status.Health) > 0 && (status.BackendState == ipn.Starting.String() || status.BackendState == ipn.NoState.String()) {
			response.HealthStatus = append(response.HealthStatus, status.Health...)
		}
		response.TailscaleVersion = &status.Version
		response.HealthStatus = append(response.HealthStatus, description)
	}
	jsonResp, _ := json.Marshal(response)
	_, err = w.Write(jsonResp)
	if err != nil {
		log.FromContext(ctrl.SetupSignalHandler()).Error(err, "error during json Marshal")
	}
}

func newHealthMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	return mux
}
