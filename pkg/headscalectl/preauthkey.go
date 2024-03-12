// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

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

//get
//create

package headscalectl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultPreAuthKeyExpiry = "1d"
)

var (
	reusable    bool
	ephemeral   bool
	durationStr string
	tags        []string
	user        string
	key         string
	filePath    string
	force       bool
)

func init() {
	rootCmd.AddCommand(preauthKeyCmd)
	preauthKeyCmd.AddCommand(createPreAuthKeyCmd())
	preauthKeyCmd.AddCommand(listPreAuthKeyCmd())
	preauthKeyCmd.AddCommand(expirePreAuthKeyCmd())
	createPreAuthKeyCmd().DisableSuggestions = true
}

var preauthKeyCmd = &cobra.Command{
	Use:   "preauthkey",
	Short: "Command to issue PreAuthKey",
}

var createPreAuthKeyCmdUsage = "create [flags] [username]"

type createPreAuthKeyCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func createPreAuthKeyCmd() *cobra.Command {
	c := createPreAuthKeyCmdOptions{}
	cmd := &cobra.Command{
		Use:   createPreAuthKeyCmdUsage,
		Short: "Create a preauthkey for a headscale user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			grpcClient, err := headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			if err != nil {
				return err
			}
			c.headscaleGRPCClient = grpcClient
			return c.run()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateFlags()
		},
	}
	cmd.Flags().BoolVar(&reusable, "reusable", false, "Create a reusable preauthkey")
	cmd.Flags().BoolVar(&ephemeral, "ephemeral", false, "Create an ephemeral preauthkey")
	cmd.Flags().StringVar(&durationStr, "expiration", DefaultPreAuthKeyExpiry, "Expiration time for the preauthkey")
	cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "Tags for the preauthkey")
	cmd.Flags().StringVarP(&user, "user", "u", "", "User for the preauthkey")
	cmd.Flags().BoolVar(&force, "force", false, "Force the creation of a preauthkey, by creating the user if it doesn't exist")
	cmd.Flags().StringVar(&filePath, "file", "", "File to write the preauthkey to")
	return cmd
}

func (o *createPreAuthKeyCmdOptions) run() error {
	if user == "" {
		return fmt.Errorf("user is required to create preauthkeys")
	}
	duration, err := model.ParseDuration(durationStr)
	if err != nil {
		log.FromContext(context.Background()).Error(err, "error parsing duration")
		return err
	}

	createResp, err := o.headscaleGRPCClient.CreatePreAuthKey(context.Background(), &headscalev1.CreatePreAuthKeyRequest{
		User:       user,
		Reusable:   reusable,
		Ephemeral:  ephemeral,
		Expiration: timestamppb.New(time.Now().Add(time.Duration(duration))),
		AclTags:    tags,
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		switch {
		case !ok:
			return err
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to create preauthkey for user %s", user)
		case strings.Contains(errStatus.Message(), "User not found"):
			if force {
				createResp, err := o.headscaleGRPCClient.CreateUser(context.Background(), &headscalev1.CreateUserRequest{
					Name: user,
				})
				if err != nil {
					return err
				}
				Output(createResp.User, "", o.outputFormat)
				return o.run()
			}
			return fmt.Errorf("headscale: user %s not found", user)
		default:
			return err
		}
	}

	if filePath != "" {
		err := os.WriteFile(filePath, []byte(createResp.PreAuthKey.Key), 0644)
		if err != nil {
			log.FromContext(context.Background()).Error(err, "error writing preauthkey to file")
			return err
		}
		fmt.Println("PreAuthKey written to file", filePath)
		return nil
	}
	Output(createResp.PreAuthKey, createResp.PreAuthKey.Key, o.outputFormat)

	return nil
}

var listPreAuthKeyCmdUsage = "list [flags] [username]"

type listPreAuthKeyCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func listPreAuthKeyCmd() *cobra.Command {
	c := listPreAuthKeyCmdOptions{}
	cmd := &cobra.Command{
		Use:   listPreAuthKeyCmdUsage,
		Short: "List a preauthkey for a headscale user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			grpcClient, err := headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			if err != nil {
				return err
			}
			c.headscaleGRPCClient = grpcClient
			return c.run()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateFlags()
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "", "User for the preauthkey")
	return cmd
}

func (o *listPreAuthKeyCmdOptions) run() error {
	if user == "" {
		return fmt.Errorf("user is required to list preauthkeys")
	}
	listResp, err := o.headscaleGRPCClient.ListPreAuthKeys(context.Background(), &headscalev1.ListPreAuthKeysRequest{
		User: user,
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		switch {
		case !ok:
			return err
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to list preauthkey for user %s", user)
		default:
			return err
		}
	}
	if o.outputFormat != "" {
		Output(listResp.PreAuthKeys, "", o.outputFormat)
		return nil
	}
	log.FromContext(context.Background()).Info("PreAuthKey", "user", listResp.PreAuthKeys)
	return nil
}

var expirePreAuthKeyCmdUsage = "expire [flags] [username]"

type expirePreAuthKeyCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func expirePreAuthKeyCmd() *cobra.Command {
	c := expirePreAuthKeyCmdOptions{}
	cmd := &cobra.Command{
		Use:   expirePreAuthKeyCmdUsage,
		Short: "Expire a preauthkey for a headscale user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			grpcClient, err := headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			if err != nil {
				return err
			}
			c.headscaleGRPCClient = grpcClient
			return c.run()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateFlags()
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "", "User for the preauthkey")
	cmd.Flags().StringVar(&key, "key", "", "preauthkey")

	return cmd
}

func (o *expirePreAuthKeyCmdOptions) run() error {
	if key == "" || user == "" {
		return fmt.Errorf("user and key are required to expire preauthkeys")
	}
	delResp, err := o.headscaleGRPCClient.ExpirePreAuthKey(context.Background(), &headscalev1.ExpirePreAuthKeyRequest{
		User: user,
		Key:  key,
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		switch {
		case !ok:
			return err
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to list preauthkey for user %s", user)
		case strings.Contains(errStatus.Message(), "AuthKey expired"):
			return fmt.Errorf("headscale: preauthkey %s for user %s already expired", key, user)
		default:
			return err
		}
	}
	Output(delResp, "PreAuthKey expired", o.outputFormat)
	return nil
}
