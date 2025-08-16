package grpcsuit

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func NewErrEnumWithMsg(err protoreflect.Enum, errMsg string) error {
	if errMsg == "" {
		errMsg = string(err.Descriptor().Values().ByNumber(err.Number()).Name())
	}
	return status.Errorf(codes.Code(err.Number()), errMsg)
}

func NewErrEnum(err protoreflect.Enum) error {
	return NewErrEnumWithMsg(err, "")
}

func NewRpcErr(err protoreflect.Enum) error {
	return NewErrEnum(err)
}

func NewRpcErrWithMsg(err protoreflect.Enum, msg string) error {
	return NewErrEnumWithMsg(err, msg)
}
