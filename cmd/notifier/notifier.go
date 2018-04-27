package main

import (
	"context"
	"log"

	"github.com/appleboy/gorush/rpc/proto"

	"google.golang.org/grpc"
)

// Notifier : handles android and ios push notifications
type Notifier struct {
	address string
	conn    *grpc.ClientConn
	c       proto.GorushClient
}

// New : Set up a connection to the server.
func New(address string) *Notifier {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return nil
	}

	c := proto.NewGorushClient(conn)

	return &Notifier{conn: conn, c: c, address: address}
}

// Send : Sends a push notification to given devices
func (n *Notifier) Send(tokens []string, message string) error {
	_, err := n.c.Send(context.Background(), &proto.NotificationRequest{
		Platform: 2,
		Tokens:   tokens,
		Message:  message,
		Badge:    1,
	})

	return err
}

// Close the current connection
func (n *Notifier) Close() error {
	return n.conn.Close()
}
