package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.trivaris.org/trivaris/unitracker/internal/api"
)

func main() {
	connStr := flag.String("conn", os.Getenv("CONN_STRING"), "The MySQL")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	
	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

	store, err := api.NewSQLStore(initCtx, *connStr)
    if err != nil {
        fmt.Println(err)
		return
	}
	defer store.Close()

	routing := api.NewRouting(store)

	go func() {
		err := routing.Route()
		if err != nil {
			fmt.Println(err)
		}
	}()
	
	<-ctx.Done()
	fmt.Println()
	fmt.Println(context.Cause(ctx))
	fmt.Println("exiting")
}
