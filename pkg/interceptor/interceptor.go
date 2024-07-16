package interceptor

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryServerLogger() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (
		interface{}, error,
	) {
		slog.Debug("request", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		slog.Debug("response", "method", info.FullMethod, "code", status.Code(err).String())

		if err != nil {
			slog.Error(
				"got node error",
				"method",
				info.FullMethod,
				"code",
				status.Code(err).String(),
				"err",
				err,
			)
		}

		return resp, err
	}
}
