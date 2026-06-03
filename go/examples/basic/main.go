// Example: solve flow with site URL, DDJS key, and optional proxy.
// Set SITE_URL and DDJS_KEY before running.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/CircuitSavage/datadome-solver/pkg/datadome"
)

func main() {
	site := os.Getenv("SITE_URL")
	key := os.Getenv("DDJS_KEY")
	if site == "" || key == "" {
		log.Fatal("set SITE_URL and DDJS_KEY")
	}

	opts := []datadome.Option{datadome.WithDDJSKey(key)}
	if p := os.Getenv("PROXY_URL"); p != "" {
		opts = append(opts, datadome.WithProxy(p))
	}

	client, err := datadome.New(site, opts...)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.Solve(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Cookie)
}
