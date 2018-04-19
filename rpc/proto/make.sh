protoc -I/usr/local/include -I. \
  -I$GOPATH/src \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --grpc-gateway_out=logtostderr=true:. \
  --go_out=plugins=grpc:. \
  ./rpc.proto
mockgen -source ./rpc.pb.go -destination ../mock_pb/mock_pb.go
