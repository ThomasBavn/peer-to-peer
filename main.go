package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	node "github.com/ThomasBavn/peer-to-peer/grpc"
	"google.golang.org/grpc"
)

type peer struct {
	node.UnimplementedNodeServer
	id            int32
	amountOfPings map[int32]int32
	clients       map[int32]node.NodeClient
	ctx           context.Context
	state 		  string 
	lamport		  int32
}

const (
	RELEASED = "Released"
	WANTED = "Wanted"
	HELD = "Held"
)

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	ownPort := int32(arg1) + 5000 // TODO: Remove the 5000 before final commit

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := &peer{
		id:            ownPort,
		amountOfPings: make(map[int32]int32),
		clients:       make(map[int32]node.NodeClient),
		ctx:           ctx,
		state: 	   	   RELEASED,
		lamport:	   int32(0),
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	node.RegisterNodeServer(grpcServer, p)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)

		if port == ownPort {
			continue
		}

		var conn *grpc.ClientConn
		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := node.NewNodeClient(conn)
		p.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		p.sendPingToAll()
		p.requestAccess()
	}
}


func (p *peer) Ping(ctx context.Context, req *node.Request) (*node.Reply, error) {
	id := req.Id
	p.amountOfPings[id]++

	rep := &node.Reply{Amount: p.amountOfPings[id]}
	return rep, nil
}

func (p *peer) sendPingToAll() {
	request := &node.Request{Id: p.id}
	for id, client := range p.clients {
		reply, err := client.Node(p.ctx, request)
		if err != nil {
			log.Println("something went wrong")
		}
		log.Printf("Got reply from id %v: %v\n", id, reply.Amount)
	}
}

func (p *peer ) requestAccess() {
	p.state = WANTED
	confirmsNeeded := len(p.clients) - 1
	
}

func req(lamport int32, p *peer) (confirm bool){
	return p.state == WANTED || p.state == HELD && lamport < p.lamport
}

func (p *peer) criticalSection() {
	if (p.state != HELD) {
		log.Printf("%v tried to enter critical section without permission\n",p.id)
		return;
	}

	log.Printf("%v has entered the critical section\n",p.id)
	p.state = RELEASED	
}
