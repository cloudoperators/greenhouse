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
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type KubeConfig struct {
	config  api.Config
	logger  *logrus.Logger
	dataDir string
}

func (kc KubeConfig) addUser(name string, cfg *rest.Config, namespace string) {
	kc.config.Contexts[name] = &api.Context{
		Cluster:  "default",
		AuthInfo: name,
	}
	if namespace != "" {
		kc.config.Contexts[name].Namespace = namespace
	}

	kc.config.AuthInfos[name] = &api.AuthInfo{
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
	}

	kc.logger.Infof("Added user %s to kubeconfig", name)
}

func (kc KubeConfig) writeFile(name string) error {
	kc.logger.Info("Creating kubeconfig file")
	kubeconfigFileName := kc.dataDir + "/" + name
	if err := os.Truncate(kubeconfigFileName, 0); err != nil {
		kc.logger.Printf("Failed to truncate: %v - continuing", err)
	}

	content, err := clientcmd.Write(kc.config)
	if err != nil {
		return fmt.Errorf("unable to write kubeconfig content: %w", err)
	}

	err = os.MkdirAll(kc.dataDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create target dir: %w", err)
	}

	kubeconfigFile, err := os.OpenFile(kubeconfigFileName, os.O_CREATE|os.O_WRONLY, os.FileMode(0755))
	if err != nil {
		return fmt.Errorf("unable to open kubeconfig file: %w", err)
	}
	defer func() {
		if err = kubeconfigFile.Close(); err != nil {
			err = fmt.Errorf("unable to close kubeconfig file: %w", err)
		}
	}()
	if _, err := kubeconfigFile.Write(content); err != nil {
		return fmt.Errorf("unable to write kubeconfig file: %w", err)
	}
	kc.logger.Infof("Created kubeconfig file: %s", kubeconfigFile.Name())

	return err
}

func (kc KubeConfig) writeCertDataToFiles() error {
	kc.logger.Info("Creating cert files")

	for name, authData := range kc.config.AuthInfos {
		certFileNameStub := kc.dataDir + "/" + name + ".client"
		if err := os.Truncate(certFileNameStub+".key", 0); err != nil {
			kc.logger.Printf("Failed trying to truncate: %v - continuing", err)
		}
		if err := os.Truncate(certFileNameStub+".crt", 0); err != nil {
			kc.logger.Printf("Failed trying to truncate: %v - continuing", err)
		}

		err := os.MkdirAll(kc.dataDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to create target dir: %w", err)
		}
		certFile, err := os.OpenFile(certFileNameStub+".crt", os.O_CREATE|os.O_WRONLY, os.FileMode(0755))
		if err != nil {
			return fmt.Errorf("unable to open certfile: %w", err)
		}
		// FIXME: defer called in for loop
		defer func() {
			if err := certFile.Close(); err != nil {
				kc.logger.Errorf("unable to close certfile: %s", err.Error())
			}
		}()
		if _, err := certFile.Write(authData.ClientCertificateData); err != nil {
			return fmt.Errorf("unable to write certfile: %w", err)
		}
		kc.logger.Infof("Created certfile: %s", certFile.Name())

		keyFile, err := os.OpenFile(certFileNameStub+".key", os.O_CREATE|os.O_WRONLY, os.FileMode(0755))
		if err != nil {
			return fmt.Errorf("unable to open keyfile: %w", err)
		}
		// FIXME: defer called in for loop
		defer func() {
			if err := keyFile.Close(); err != nil {
				kc.logger.Errorf("unable to close keyfile: %s", err.Error())
			}
		}()
		if _, err := keyFile.Write(authData.ClientKeyData); err != nil {
			return fmt.Errorf("unable to write keyfile: %w", err)
		}
		kc.logger.Infof("Created keyfile: %s", keyFile.Name())
	}

	return nil
}
