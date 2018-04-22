package rpc

import (
	"google.golang.org/grpc"
)

// Dial dials
func Dial(target string) *grpc.ClientConn {
	conn, _ := grpc.Dial(target, grpc.WithInsecure())
	return conn
}
