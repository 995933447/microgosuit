package grpcsuit

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func NewErrFromEnumWithMsg(err protoreflect.Enum, errMsg string) error {
	if errMsg == "" {
		errMsg = string(err.Descriptor().Values().ByNumber(err.Number()).Name())
	}
	return status.Errorf(codes.Code(err.Number()), errMsg)
}

func NewErrFromEnum(err protoreflect.Enum) error {
	return NewErrFromEnumWithMsg(err, "")
}

func NewRpcErr(err protoreflect.Enum) error {
	return NewErrFromEnum(err)
}

func NewRpcErrWithMsg(err protoreflect.Enum, msg string) error {
	return NewErrFromEnumWithMsg(err, msg)
}
