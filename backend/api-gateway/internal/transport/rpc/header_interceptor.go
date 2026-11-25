package rpc

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ForwardMetadataInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if inMD, ok := metadata.FromIncomingContext(ctx); ok {
		filtered := metadata.MD{}
		for k, vals := range inMD {
			if !strings.HasPrefix(k, ":") {
				filtered[k] = vals
			}
		}
		if outMD, ok := metadata.FromOutgoingContext(ctx); ok {
			filtered = metadata.Join(outMD, filtered)
		}
		ctx = metadata.NewOutgoingContext(ctx, filtered)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
