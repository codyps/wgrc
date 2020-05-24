package main

import (
	"log"
	"context"
	"io"

	pb "github.com/jmesmon/wgrc/api"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial(":7777", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewStreamerClient(conn)

	req := pb.ListenReq{}
	stream, err := client.Listen(context.Background(), &req)
	if err != nil {
		log.Fatalf("could not rpc Listen: %v", err)
	}

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.Listen(_) = _, %v", client, err)
		}

		log.Println(event)
	}
}
