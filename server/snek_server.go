package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"

	pb "github.com/bcspragu/Snek/proto"
	"google.golang.org/grpc"
)

type updateErr []error

func (u updateErr) Error() string {
	var buf bytes.Buffer
	for i, err := range u {
		buf.WriteString(err.Error())
		if i != len(u)-1 {
			buf.WriteString(", ")
		}
	}
	return buf.String()
}

type ID int64
type snek struct {
	id     ID
	stream pb.Snek_UpdateServer
}

func (s *snek) send(resp *pb.UpdateResponse) error {
	return s.stream.Send(resp)
}

type server struct {
	sync.Mutex
	sneks     map[ID]*snek
	highestID ID
}

func newServer() *server {
	return &server{
		sneks: make(map[ID]*snek),
	}
}

func (s *server) removeSnek(snek *snek) {
	s.Lock()
	defer s.Unlock()
	delete(s.sneks, snek.id)
}

func (s *server) addSnek(stream pb.Snek_UpdateServer) *snek {
	s.Lock()
	defer s.Unlock()
	id := s.highestID + 1
	s.highestID = id
	snek := &snek{id: id, stream: stream}
	s.sneks[id] = snek
	return snek
}

func (s *server) sendUpdates(req *pb.UpdateRequest, id ID) error {
	s.Lock()
	var errs updateErr
	defer s.Unlock()
	resp := &pb.UpdateResponse{
		Id:      int32(id),
		NewHead: req.NewHead,
		OldTail: req.OldTail,
	}
	for _, snek := range s.sneks {
		if snek.id == id {
			continue
		}
		if err := snek.send(resp); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) != 0 {
		return errs
	}
	return nil
}

func (s *server) Update(stream pb.Snek_UpdateServer) error {
	// When we start a stream, we add a new snek to our collection
	snek := s.addSnek(stream)
	log.Printf("Started stream for snek %d", snek.id)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			s.removeSnek(snek)
			return nil
		}
		if err != nil {
			s.removeSnek(snek)
			return err
		}
		if err := s.sendUpdates(in, snek.id); err != nil {
			log.Printf("sendUpdates(%v, %d): %v", in, snek.id, err)
		}
	}
}

func main() {
	grpcServer := grpc.NewServer()
	pb.RegisterSnekServer(grpcServer, newServer())

	l, err := net.Listen("tcp", ":6000")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:6000")
	grpcServer.Serve(l)
}
