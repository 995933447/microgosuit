package micro

import (
	"context"
	"github.com/995933447/microgosuit/env"
	"github.com/995933447/microgosuit/grpcsuit"
)

func InitSuitWithGrpc(ctx context.Context, metaFilePath, resolveSchema string) error {
	if err := env.InitMeta(metaFilePath); err != nil {
		return err
	}

	if err := grpcsuit.InitGrpcResolver(ctx, resolveSchema); err != nil {
		return err
	}

	return nil
}
