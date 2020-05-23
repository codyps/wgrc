//go:generate protoc --go_out=plugins=grpc:api stream.proto
package main

import (
	"log"
	"net"
	//"context"

	pb "github.com/jmesmon/wgrc/api"
	"google.golang.org/grpc"
)

type streamerServer struct {
	pb.UnimplementedStreamerServer
}

func (s *streamerServer) Listen(in *pb.ListenReq, stream pb.Streamer_ListenServer) error {
	stream.Send(&pb.ListenEvent{})
	stream.Send(&pb.ListenEvent{ Other: "sssss"})
	return nil
}

func main() {

	listen, err := net.Listen("tcp", ":7777")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	srv := streamerServer{}

	pb.RegisterStreamerServer(s, &srv)

	if err := s.Serve(listen); err != nil {
		log.Fatalf("failed to server: %s", err)
	}

}
