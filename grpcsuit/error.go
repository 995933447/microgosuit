package grpcsuit

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const ErrCodeUnknown = -1

func newErrFromEnumWithMsg(err protoreflect.Enum, errMsg string) error {
	if errMsg == "" {
		errMsg = string(err.Descriptor().Values().ByNumber(err.Number()).Name())
	}
	return status.Errorf(codes.Code(err.Number()), errMsg)
}

func newErrFromEnum(err protoreflect.Enum) error {
	return newErrFromEnumWithMsg(err, "")
}

func NewRpcErrWithMsg(err protoreflect.Enum, errMsg string) error {
	return newErrFromEnumWithMsg(err, errMsg)
}

func NewRpcErr(err protoreflect.Enum) error {
	return newErrFromEnum(err)
}

func IsUnknownError(err error) bool {
	code := GetRpcErrCode(err)
	return code == protoreflect.EnumNumber(codes.Unknown) || code == -1
}

func GetRpcErrCode(err error) protoreflect.EnumNumber {
	st, ok := status.FromError(err)
	if ok {
		return protoreflect.EnumNumber(st.Code())
	}
	return ErrCodeUnknown
}

func GetRpcErrMsg(err error) string {
	st, ok := status.FromError(err)
	if ok {
		return st.Message()
	}
	return err.Error()
}
