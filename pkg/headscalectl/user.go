// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package headscalectl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	headscalev1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(createUserCmd())
	userCmd.AddCommand(deleteUserCmd())
	userCmd.AddCommand(getUserCmd())
	userCmd.AddCommand(listUserCmd())
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Commands to interact with Users",
}

var createUserCmdUsage = "create [username]"

type createUserCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func createUserCmd() *cobra.Command {
	c := createUserCmdOptions{}
	return &cobra.Command{
		Use:   createUserCmdUsage,
		Short: "Create a User in Headscale",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			userName := args[0]
			grpcClient, err := headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			if err != nil {
				return err
			}
			c.headscaleGRPCClient = grpcClient
			return c.run(userName)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateFlags()
		},
		Args: cobra.ExactArgs(1),
	}
}

func (o *createUserCmdOptions) run(userName string) error {
	createResp, err := o.headscaleGRPCClient.CreateUser(context.Background(), &headscalev1.CreateUserRequest{
		Name: userName,
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return err
		}
		switch {
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to create user %s", userName)
		case strings.Contains(errStatus.Message(), "already exists"):
			log.FromContext(context.Background()).Info("User already exists", "user", userName)
			return nil
		}
		return err
	}

	Output(createResp.User, "User created", o.outputFormat)
	return nil
}

var deleteUserCmdUsage = "delete [username]"

type deleteUserCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func deleteUserCmd() *cobra.Command {
	c := deleteUserCmdOptions{}
	return &cobra.Command{
		Use:   deleteUserCmdUsage,
		Short: "Delete a User in Headscale",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			userName := args[0]
			grpcClient, err := headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			if err != nil {
				return err
			}
			c.headscaleGRPCClient = grpcClient
			return c.run(userName)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateFlags()
		},
		Args: cobra.ExactArgs(1),
	}
}

func (o *deleteUserCmdOptions) run(userName string) error {
	delResp, err := o.headscaleGRPCClient.DeleteUser(context.Background(), &headscalev1.DeleteUserRequest{
		Name: userName,
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return err
		}
		switch {
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to delete user %s", userName)
		case strings.Contains(errStatus.Message(), "not found"):
			return fmt.Errorf("headscale: user %s not found", userName)
		}
		return err
	}
	Output(delResp, "User deleted", o.outputFormat)
	return nil
}

var getUserCmdUsage = "get [username]"

type getUserCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func getUserCmd() *cobra.Command {
	c := getUserCmdOptions{}
	return &cobra.Command{
		Use:   getUserCmdUsage,
		Short: "Get a User in Headscale",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o, err := cmd.Flags().GetString("output"); err != nil {
				return fmt.Errorf("invalid value for flag --output: %w", err)
			} else {
				c.outputFormat = o
			}
			userName := args[0]
			grpcClient, err := headscaleGRCPClientFunc(headscaleGRPCURL, headscaleAPIKey)
			if err != nil {
				return err
			}
			c.headscaleGRPCClient = grpcClient
			return c.run(userName)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateFlags()
		},
		Args: cobra.ExactArgs(1),
	}
}

func (o *getUserCmdOptions) run(userName string) error {
	getResp, err := o.headscaleGRPCClient.GetUser(context.Background(), &headscalev1.GetUserRequest{
		Name: userName,
	})
	if err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return err
		}
		switch {
		case strings.Contains(errStatus.Message(), "Unauthorized"):
			return fmt.Errorf("headscale: unauthorized to get user %s", userName)
		case strings.Contains(errStatus.Message(), "not found"):
			return fmt.Errorf("headscale: user %s not found", userName)
		}
		return err
	}
	if o.outputFormat != "" {
		Output(getResp.User, "", o.outputFormat)
		return nil
	}
	log.FromContext(context.Background()).Info("User", "user", getResp.User.Name, "created at", getResp.User.CreatedAt)
	return nil
}

var listUserCmdUsage = "list"

type listUserCmdOptions struct {
	headscaleGRPCClient headscalev1.HeadscaleServiceClient
	outputFormat        string
}

func listUserCmd() *cobra.Command {
	c := listUserCmdOptions{}
	return &cobra.Command{
		Use:   listUserCmdUsage,
		Short: "List Users in Headscale",
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
		Args: cobra.NoArgs,
	}
}

func (o *listUserCmdOptions) run() error {
	listResp, err := o.headscaleGRPCClient.ListUsers(context.Background(), &headscalev1.ListUsersRequest{})
	if err != nil {
		errStatus, ok := status.FromError(err)
		if !ok {
			return err
		}

		if strings.Contains(errStatus.Message(), "Unauthorized") {
			return errors.New("headscale: unauthorized to list users")
		}
		return err
	}
	if o.outputFormat != "" {
		Output(listResp.Users, "", o.outputFormat)
		return nil
	}
	var userNames []string
	for _, user := range listResp.Users {
		userNames = append(userNames, user.Name)
	}
	log.FromContext(context.Background()).Info("Users", "users", userNames)
	return nil
}
