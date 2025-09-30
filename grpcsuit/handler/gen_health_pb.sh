mkdir -p ./health
protoc --go_out=./health --go-grpc_out=./health --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --proto_path=./proto ./proto/health.proto
