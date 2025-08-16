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

func IsInternalError(err error) bool {
	return IsUnknownError(err)
}

func IsUnknownError(err error) bool {
	code := GetRpcErrCode(err)
	return code == int32(codes.Unknown) || code == -1
}

func GetRpcErrCode(err error) int32 {
	st, ok := status.FromError(err)
	if ok {
		return int32(st.Code())
	}
	return -1
}

func GetRpcErrMsg(err error) string {
	st, ok := status.FromError(err)
	if ok {
		return st.Message()
	}
	return err.Error()
}
