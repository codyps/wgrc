//go:generate protoc --go_out=plugins=grpc:api stream.proto
package main

import (
	"log"
	"net"
	"time"
	"sync"
	"math/rand"

	pb "github.com/jmesmon/wgrc/api"
	"google.golang.org/grpc"
)

type listenerEvent struct {
	id uint64
	newState bool
}

func (le *listenerEvent) AsPbEvent() pb.ListenEvent {
	ns := -1
	if le.newState {
		ns = 1
	}
	return pb.ListenEvent {
		Id: le.id,
		NewState: int32(ns),
	}
}

type listener struct {
	events chan listenerEvent
}

type item struct {
	id uint64
	state bool
}

func (i *item) AsPbEvent() pb.ListenEvent {
	ns := -1
	if i.state {
		ns = 1
	}
	return pb.ListenEvent {
		Id: i.id,
		NewState: int32(ns),
	}
}

type streamerServer struct {
	pb.UnimplementedStreamerServer

	stateLock sync.Mutex
	items []*item
	listeners []*listener
}

const MAX_ITEMS = 10000

func (s *streamerServer) EventsGenerate() {
	next_id := uint64(1)
	for {
		time.Sleep(1000000000)
		if len(s.items) < MAX_ITEMS {
			add := rand.Int31n(2)
			if add != 0 {
				log.Println("adding new item ", next_id)
				s.stateLock.Lock()
				e := item {
					id: next_id,
					state: true,
				}
				next_id++

				s.items = append(s.items, &e)

				for _, listener := range s.listeners {
					listener.events <-listenerEvent {
						id: e.id,
						newState: e.state,
					}
				}
				s.stateLock.Unlock()
			} else {
				//log.Println("skipping new item add")
			}
		}

		if up := rand.Int31n(2); up == 0 {
			// state down
			s.stateLock.Lock()
			if len(s.items) != 0 {
				e := s.items[rand.Intn(len(s.items))]
				if e.state {
					log.Println("item disable ", e.id)
					e.state = false
					for _, listener := range s.listeners {
						listener.events <-listenerEvent {
							id: e.id,
							newState: e.state,
						}
					}
				}
			}
			s.stateLock.Unlock()
		} else {
			s.stateLock.Lock()
			if len(s.items) != 0 {
				e := s.items[rand.Intn(len(s.items))]
				if !e.state {
					log.Println("item enable ", e.id)
					e.state = true
					for _, listener := range s.listeners {
						listener.events <-listenerEvent {
							id: e.id,
							newState: e.state,
						}
					}
				}
			}
			s.stateLock.Unlock()
		}
	}
}

func NewServer() *streamerServer {
	items := make([]*item, 0)
	listeners := make([]*listener, 0)

	s := streamerServer {
		items: items,
		listeners: listeners,
	}
	go s.EventsGenerate()
	return &s
}

func (s *streamerServer) Listen(in *pb.ListenReq, stream pb.Streamer_ListenServer) error {
	s.stateLock.Lock()
	for _, e := range s.items {
		log.Println("sending ", e.id)
		n := e.AsPbEvent()
		stream.Send(&n)
	}
	q := listener { events: make(chan listenerEvent) }
	s.listeners = append(s.listeners, &q)
	s.stateLock.Unlock()

	for {
		v := <-q.events
		n := v.AsPbEvent()
		stream.Send(&n)
	}
}

func main() {
	listen, err := net.Listen("tcp", ":7777")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	srv := NewServer()
	pb.RegisterStreamerServer(s, srv)
	log.Println("serving")

	if err := s.Serve(listen); err != nil {
		log.Fatalf("failed to server: %s", err)
	}

}
