// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package headscalectl

import (
	"context"
	"fmt"
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
	defaultAPIKeyExpiry = "90d"
	apiPrefixLength     = 10
)

var (
	apiKeyExpiry    string
	socketPath      string
	socketCall      bool
	prefix          string
	secretName      string
	secretNamespace string
)

func init() {
	rootCmd.AddCommand(apiKeyCMD)
	apiKeyCMD.AddCommand(createAPIKeyCmd())
	apiKeyCMD.AddCommand(listAPIKeyCmd())
	apiKeyCMD.AddCommand(expireAPIKeyCmd())
}

var apiKeyCMD = &cobra.Command{
	Use:   "apikey",
	Short: "Commands to interact with Headscale API keys",
}

var createAPIKeyCmdUsage = "create"

type createAPIKeyCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func createAPIKeyCmd() *cobra.Command {
	c := createAPIKeyCmdOptions{}
	cmd := &cobra.Command{
		Use:   createAPIKeyCmdUsage,
		Short: "Creates a new APIKey for Headscale",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var grpcClient headscalev1.HeadscaleServiceClient
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			if !socketCall {
				grpcClient, err = headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			} else {
				grpcClient, err = headscaleGRPCSocketClientFunc(socketPath)
				if err != nil {
					return err
				}
			}
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
	cmd.Flags().StringVarP(&apiKeyExpiry, "expiration", "e", defaultAPIKeyExpiry, "Expiration time for the apikey")
	cmd.Flags().StringVar(&socketPath, "socket-path", "", "Path to the headscale agent socket")
	cmd.Flags().BoolVar(&socketCall, "socket", false, "Call the headscale agent socket")
	cmd.Flags().StringVar(&secretName, "secret-name", "", "Name of the secret to create")
	cmd.Flags().StringVar(&secretNamespace, "secret-namespace", "", "Kubernetes namespace to create the secret in")
	return cmd
}

func (o *createAPIKeyCmdOptions) run() error {
	duration, err := model.ParseDuration(apiKeyExpiry)
	if err != nil {
		log.FromContext(context.Background()).Error(err, "error parsing duration")
		return err
	}
	timeDuration := time.Duration(duration)
	if timeDuration < time.Hour*24 {
		log.FromContext(context.Background()).Info("duration must be greater than 1 day")
		timeDuration = time.Hour * 24
	}

	createResp, err := o.headscaleGRPCClient.CreateApiKey(context.Background(), &headscalev1.CreateApiKeyRequest{
		Expiration: timestamppb.New(time.Now().Add(timeDuration)),
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		switch {
		case !ok:
			return err
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to create APIKey")
		default:
			return err
		}
	}
	if o.outputFormat == "secret" {
		tickerDuration := timeDuration - (time.Hour * 8)
		ticker := time.NewTicker(tickerDuration)
		for ; true; <-ticker.C {
			createOrUpdateSecretInCluster(createResp.ApiKey, secretName, secretNamespace)
			log.FromContext(context.Background()).Info("New APIKey would be generated", "at:", time.Now().Add(tickerDuration))
		}
	}
	Output(createResp.ApiKey, createResp.ApiKey, o.outputFormat)
	return nil
}

var listAPIKeyCmdUsage = "list"

type listAPIKeyCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func listAPIKeyCmd() *cobra.Command {
	c := listAPIKeyCmdOptions{}
	cmd := &cobra.Command{
		Use:   listAPIKeyCmdUsage,
		Short: "Lists all APIKey for Headscale",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var grpcClient headscalev1.HeadscaleServiceClient
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			if !socketCall {
				grpcClient, err = headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			} else {
				grpcClient, err = headscaleGRPCSocketClientFunc(socketPath)
			}
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
	cmd.Flags().StringVar(&socketPath, "socket-path", "", "Path to the headscale agent socket")
	cmd.Flags().BoolVar(&socketCall, "socket", false, "Call the headscale agent socket")
	return cmd
}

func (o *listAPIKeyCmdOptions) run() error {
	listResp, err := o.headscaleGRPCClient.ListApiKeys(context.Background(), &headscalev1.ListApiKeysRequest{})
	if err != nil {
		errStatus, ok := status.FromError(err)
		switch {
		case !ok:
			return err
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to list APIKey")
		default:
			return err
		}
	}
	if o.outputFormat != "" {
		Output(listResp.ApiKeys, "", o.outputFormat)
		return nil
	}
	log.FromContext(context.Background()).Info("APIKey", "apikey", listResp.ApiKeys)
	return nil
}

var expireAPIKeyCmdUsage = "expire"

type expireAPIKeyCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func expireAPIKeyCmd() *cobra.Command {
	c := expireAPIKeyCmdOptions{}
	cmd := &cobra.Command{
		Use:   expireAPIKeyCmdUsage,
		Short: "Expire an APIKey for Headscale",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var grpcClient headscalev1.HeadscaleServiceClient
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			if !socketCall {
				grpcClient, err = headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			} else {
				grpcClient, err = headscaleGRPCSocketClientFunc(socketPath)
			}
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

	cmd.Flags().StringVar(&prefix, "prefix", "", "Prefix of the apikey to expire")
	cmd.Flags().StringVar(&socketPath, "socket-path", "", "Path to the headscale agent socket")
	cmd.Flags().BoolVar(&socketCall, "socket", false, "Call the headscale agent socket")
	return cmd
}

func (o *expireAPIKeyCmdOptions) run() error {
	if len(prefix) != apiPrefixLength {
		return fmt.Errorf("prefix must be exactly %d characters long", apiPrefixLength)
	}
	expResp, err := o.headscaleGRPCClient.ExpireApiKey(context.Background(), &headscalev1.ExpireApiKeyRequest{
		Prefix: prefix,
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		switch {
		case !ok:
			return err
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to expire APIKey")
		case strings.Contains(errStatus.Message(), "record not found"):
			return fmt.Errorf("headscale: APIKey not found")
		default:
			return err
		}
	}
	Output(expResp, "APIKey expired", o.outputFormat)
	return nil
}
