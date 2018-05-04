PB = $(wildcard *.proto)
GO = $(PB:.proto=.pb.go)

all: $(GO) mockgen

%.pb.go: %.proto
	protoc \
	-I.:$(GOPATH)/src:$(GOPATH)/src/github.com/medibloc/go-medibloc/core/pb:$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--gogo_out=plugins=grpc:. \
	--grpc-gateway_out=logtostderr=true:. \
	--swagger_out=logtostderr=true:. \
	$<

%.proto:

mockgen:
	mockgen -source ./rpc.pb.go -destination ../mock_pb/mock_pb.go

clean:
	rm *.pb.go
