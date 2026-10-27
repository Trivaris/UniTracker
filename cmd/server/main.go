package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.trivaris.org/trivaris/unitracker/internal/api"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		store := api.NewStore()
		routing := api.NewRouting(store)
		routing.Route()
	}()
	<-ctx.Done()
	fmt.Println()
	fmt.Println(context.Cause(ctx))
	fmt.Println("exiting")
}
