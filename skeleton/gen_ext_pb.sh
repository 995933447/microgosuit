mkdir -p ./pb
protoc --go_out=./pb --go-grpc_out=. --go_opt=paths=source_relative --proto_path=./proto ./proto/ext.proto
