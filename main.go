package main

import (
	"context"
)

func main() {
	srv := NewServer(
		Address("0.0.0.0:6021"),
	)
	RegisterReadHandler(srv, func(sessionID string, bytes []byte) error {
		return nil
	})
	srv.Start(context.Background())
}
