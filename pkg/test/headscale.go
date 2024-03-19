// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	"errors"

	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"google.golang.org/grpc"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudoperators/greenhouse/pkg/clientutil"
)

// FakeHeadscaleClient is a fake implementation of the HeadscaleClient interface.
//
//nolint:stylecheck
type FakeHeadscaleClient struct {
	// IsUserDeleted is used to simulate the deletion of a user. If set to true, the GetUserFunc will return an error.
	IsUserDeleted bool

	CreateApiKeyFunc       func(context.Context, *v1.CreateApiKeyRequest, ...grpc.CallOption) (*v1.CreateApiKeyResponse, error) //no-lint:stylecheck
	CreatePreAuthKeyFunc   func(context.Context, *v1.CreatePreAuthKeyRequest, ...grpc.CallOption) (*v1.CreatePreAuthKeyResponse, error)
	CreateUserFunc         func(context.Context, *v1.CreateUserRequest, ...grpc.CallOption) (*v1.CreateUserResponse, error)
	DebugCreateMachineFunc func(context.Context, *v1.DebugCreateMachineRequest, ...grpc.CallOption) (*v1.DebugCreateMachineResponse, error)
	DeleteMachineFunc      func(context.Context, *v1.DeleteMachineRequest, ...grpc.CallOption) (*v1.DeleteMachineResponse, error)
	DeleteRouteFunc        func(context.Context, *v1.DeleteRouteRequest, ...grpc.CallOption) (*v1.DeleteRouteResponse, error)
	DeleteUserFunc         func(context.Context, *v1.DeleteUserRequest, ...grpc.CallOption) (*v1.DeleteUserResponse, error)
	DisableRouteFunc       func(context.Context, *v1.DisableRouteRequest, ...grpc.CallOption) (*v1.DisableRouteResponse, error)
	EnableRouteFunc        func(context.Context, *v1.EnableRouteRequest, ...grpc.CallOption) (*v1.EnableRouteResponse, error)
	ExpireApiKeyFunc       func(context.Context, *v1.ExpireApiKeyRequest, ...grpc.CallOption) (*v1.ExpireApiKeyResponse, error) //no-lint:stylecheck
	ExpireMachineFunc      func(context.Context, *v1.ExpireMachineRequest, ...grpc.CallOption) (*v1.ExpireMachineResponse, error)
	ExpirePreAuthKeyFunc   func(context.Context, *v1.ExpirePreAuthKeyRequest, ...grpc.CallOption) (*v1.ExpirePreAuthKeyResponse, error)
	GetMachineFunc         func(context.Context, *v1.GetMachineRequest, ...grpc.CallOption) (*v1.GetMachineResponse, error)
	GetMachineRoutesFunc   func(context.Context, *v1.GetMachineRoutesRequest, ...grpc.CallOption) (*v1.GetMachineRoutesResponse, error)
	GetRoutesFunc          func(context.Context, *v1.GetRoutesRequest, ...grpc.CallOption) (*v1.GetRoutesResponse, error)
	GetUserFunc            func(context.Context, *v1.GetUserRequest, ...grpc.CallOption) (*v1.GetUserResponse, error)
	ListApiKeysFunc        func(context.Context, *v1.ListApiKeysRequest, ...grpc.CallOption) (*v1.ListApiKeysResponse, error)
	ListMachinesFunc       func(context.Context, *v1.ListMachinesRequest, ...grpc.CallOption) (*v1.ListMachinesResponse, error)
	ListPreAuthKeysFunc    func(context.Context, *v1.ListPreAuthKeysRequest, ...grpc.CallOption) (*v1.ListPreAuthKeysResponse, error)
	ListUsersFunc          func(context.Context, *v1.ListUsersRequest, ...grpc.CallOption) (*v1.ListUsersResponse, error)
	MoveMachineFunc        func(context.Context, *v1.MoveMachineRequest, ...grpc.CallOption) (*v1.MoveMachineResponse, error)
	RegisterMachineFunc    func(context.Context, *v1.RegisterMachineRequest, ...grpc.CallOption) (*v1.RegisterMachineResponse, error)
	RenameMachineFunc      func(context.Context, *v1.RenameMachineRequest, ...grpc.CallOption) (*v1.RenameMachineResponse, error)
	RenameUserFunc         func(context.Context, *v1.RenameUserRequest, ...grpc.CallOption) (*v1.RenameUserResponse, error)
	SetTagsFunc            func(context.Context, *v1.SetTagsRequest, ...grpc.CallOption) (*v1.SetTagsResponse, error)
}

func (f FakeHeadscaleClient) CreateUser(ctx context.Context, in *v1.CreateUserRequest, opts ...grpc.CallOption) (*v1.CreateUserResponse, error) {
	if f.CreateUserFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to CreateUser")
	}
	return f.CreateUserFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) DeleteUser(ctx context.Context, in *v1.DeleteUserRequest, opts ...grpc.CallOption) (*v1.DeleteUserResponse, error) {
	if f.DeleteUserFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to DeleteUser")
	}
	return f.DeleteUserFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) GetUser(ctx context.Context, in *v1.GetUserRequest, opts ...grpc.CallOption) (*v1.GetUserResponse, error) {
	if f.GetUserFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to GetUser")
	}
	return f.GetUserFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) CreatePreAuthKey(ctx context.Context, in *v1.CreatePreAuthKeyRequest, opts ...grpc.CallOption) (*v1.CreatePreAuthKeyResponse, error) {
	if f.CreatePreAuthKeyFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to CreatePreAuthKey")
	}
	return f.CreatePreAuthKeyFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) DeleteMachine(ctx context.Context, in *v1.DeleteMachineRequest, opts ...grpc.CallOption) (*v1.DeleteMachineResponse, error) {
	if f.DeleteMachineFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to DeleteMachine")
	}
	return f.DeleteMachineFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) ListMachines(ctx context.Context, in *v1.ListMachinesRequest, opts ...grpc.CallOption) (*v1.ListMachinesResponse, error) {
	if f.ListMachinesFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to ListMachines")
	}
	return f.ListMachinesFunc(ctx, in, opts...)
}

//nolint:stylecheck
func (f FakeHeadscaleClient) CreateApiKey(ctx context.Context, in *v1.CreateApiKeyRequest, opts ...grpc.CallOption) (*v1.CreateApiKeyResponse, error) {
	if f.CreateApiKeyFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to CreateApiKey")
	}
	return f.CreateApiKeyFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) DebugCreateMachine(ctx context.Context, in *v1.DebugCreateMachineRequest, opts ...grpc.CallOption) (*v1.DebugCreateMachineResponse, error) {
	if f.DebugCreateMachineFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to DebugCreateMachine")
	}
	return f.DebugCreateMachineFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) DeleteRoute(ctx context.Context, in *v1.DeleteRouteRequest, opts ...grpc.CallOption) (*v1.DeleteRouteResponse, error) {
	if f.DeleteRouteFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to DeleteRoute")
	}
	return f.DeleteRouteFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) DisableRoute(ctx context.Context, in *v1.DisableRouteRequest, opts ...grpc.CallOption) (*v1.DisableRouteResponse, error) {
	if f.DisableRouteFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to DisableRoute")
	}
	return f.DisableRouteFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) EnableRoute(ctx context.Context, in *v1.EnableRouteRequest, opts ...grpc.CallOption) (*v1.EnableRouteResponse, error) {
	if f.EnableRouteFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to EnableRoute")
	}
	return f.EnableRouteFunc(ctx, in, opts...)
}

//nolint:stylecheck
func (f FakeHeadscaleClient) ExpireApiKey(ctx context.Context, in *v1.ExpireApiKeyRequest, opts ...grpc.CallOption) (*v1.ExpireApiKeyResponse, error) {
	if f.ExpireApiKeyFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to ExpireApiKey")
	}
	return f.ExpireApiKeyFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) ExpireMachine(ctx context.Context, in *v1.ExpireMachineRequest, opts ...grpc.CallOption) (*v1.ExpireMachineResponse, error) {
	if f.ExpireMachineFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to ExpireMachine")
	}
	return f.ExpireMachineFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) ExpirePreAuthKey(ctx context.Context, in *v1.ExpirePreAuthKeyRequest, opts ...grpc.CallOption) (*v1.ExpirePreAuthKeyResponse, error) {
	if f.ExpirePreAuthKeyFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to ExpirePreAuthKey")
	}
	return f.ExpirePreAuthKeyFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) GetMachine(ctx context.Context, in *v1.GetMachineRequest, opts ...grpc.CallOption) (*v1.GetMachineResponse, error) {
	if f.GetMachineFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to GetMachine")
	}
	return f.GetMachineFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) GetMachineRoutes(ctx context.Context, in *v1.GetMachineRoutesRequest, opts ...grpc.CallOption) (*v1.GetMachineRoutesResponse, error) {
	if f.GetMachineRoutesFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to GetMachineRoutes")
	}
	return f.GetMachineRoutesFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) GetRoutes(ctx context.Context, in *v1.GetRoutesRequest, opts ...grpc.CallOption) (*v1.GetRoutesResponse, error) {
	if f.GetRoutesFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to GetRoutes")
	}
	return f.GetRoutesFunc(ctx, in, opts...)
}

//nolint:stylecheck
func (f FakeHeadscaleClient) ListApiKeys(ctx context.Context, in *v1.ListApiKeysRequest, opts ...grpc.CallOption) (*v1.ListApiKeysResponse, error) {
	if f.ListApiKeysFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to ListApiKeys")
	}
	return f.ListApiKeysFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) ListPreAuthKeys(ctx context.Context, in *v1.ListPreAuthKeysRequest, opts ...grpc.CallOption) (*v1.ListPreAuthKeysResponse, error) {
	if f.ListPreAuthKeysFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to ListPreAuthKeys")
	}
	return f.ListPreAuthKeysFunc(ctx, in, opts...)
}
func (f FakeHeadscaleClient) ListUsers(ctx context.Context, in *v1.ListUsersRequest, opts ...grpc.CallOption) (*v1.ListUsersResponse, error) {
	if f.ListUsersFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to ListUsers")
	}
	return f.ListUsersFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) MoveMachine(ctx context.Context, in *v1.MoveMachineRequest, opts ...grpc.CallOption) (*v1.MoveMachineResponse, error) {
	if f.MoveMachineFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to MoveMachines")
	}
	return f.MoveMachineFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) RegisterMachine(ctx context.Context, in *v1.RegisterMachineRequest, opts ...grpc.CallOption) (*v1.RegisterMachineResponse, error) {
	if f.RegisterMachineFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to RegisterMachine")
	}
	return f.RegisterMachineFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) RenameMachine(ctx context.Context, in *v1.RenameMachineRequest, opts ...grpc.CallOption) (*v1.RenameMachineResponse, error) {
	if f.RenameMachineFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to RenameMachine")
	}
	return f.RenameMachineFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) RenameUser(ctx context.Context, in *v1.RenameUserRequest, opts ...grpc.CallOption) (*v1.RenameUserResponse, error) {
	if f.RenameUserFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to RenameUser")
	}
	return f.RenameUserFunc(ctx, in, opts...)
}

func (f FakeHeadscaleClient) SetTags(ctx context.Context, in *v1.SetTagsRequest, opts ...grpc.CallOption) (*v1.SetTagsResponse, error) {
	if f.SetTagsFunc == nil {
		return nil, errors.New("FakeHeadscaleClient was not configured to respond to SetTags")
	}
	return f.SetTagsFunc(ctx, in, opts...)
}

// DummyTailscaleClienGetter is a dummy tailscale client getter for testing purposes. As we do not have a headscale setup for testing, we need to mock the tailscale client getter.
func DummyTailscaleClientGetter(restClientGetter genericclioptions.RESTClientGetter, proxy, headscaleAddress string) (client.Client, error) {
	cfg, err := restClientGetter.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return clientutil.NewK8sClient(cfg)
}
