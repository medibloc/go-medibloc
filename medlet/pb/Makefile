PB = $(wildcard *.proto)
GO = $(PB:.proto=.pb.go)

all: $(GO)

%.pb.go: %.proto
	protoc \
	-I .:$(GOPATH)/src \
	--gogo_out=. $<
%.proto:

clean:
	rm *.pb.go
