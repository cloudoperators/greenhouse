// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package clientutil

import (
	"context"
	"net"
	"os"

	v1 "github.com/juanfont/headscale/gen/go/headscale/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	SocketWritePermissions = 0o666
)

// NewHeadscaleGRPCClient returns a new gRPC client for the Headscale API.
func NewHeadscaleGRPCClient(url, apiKey string) (v1.HeadscaleServiceClient, error) {
	grpcOptions := []grpc.DialOption{
		grpc.WithPerRPCCredentials(tokenAuth{
			token: apiKey,
		}),
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}

	cc, err := grpc.DialContext(context.TODO(), url, grpcOptions...)
	if err != nil {
		return nil, err
	}

	cl := v1.NewHeadscaleServiceClient(cc)
	return cl, nil
}

func NewHeadscaleGRPCSocketClient(socketPath string) (v1.HeadscaleServiceClient, error) {
	// Call the headscale agent socket
	grpcOptions := []grpc.DialOption{
		grpc.WithBlock(),
	}

	socket, err := os.OpenFile(socketPath, os.O_WRONLY, SocketWritePermissions)
	if err != nil {
		if os.IsPermission(err) {
			log.FromContext(context.Background()).Error(err, "permission denied to open socket")
			return nil, err
		}
		if os.IsNotExist(err) {
			log.FromContext(context.Background()).Error(err, "socket does not exist", "socketPath", socketPath)
			return nil, err
		}
	}
	socket.Close()

	grpcOptions = append(
		grpcOptions,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(grpcDialer),
	)

	cc, err := grpc.DialContext(context.Background(), socketPath, grpcOptions...)
	if err != nil {
		return nil, err
	}

	cl := v1.NewHeadscaleServiceClient(cc)
	return cl, nil
}

func grpcDialer(ctx context.Context, addr string) (net.Conn, error) {
	var d net.Dialer

	return d.DialContext(ctx, "unix", addr)
}

// tokenAuth struct to pass the token over the gRPC call.
type tokenAuth struct {
	token string
}

// Return value is mapped to request headers.
func (t tokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (tokenAuth) RequireTransportSecurity() bool {
	return true
}
