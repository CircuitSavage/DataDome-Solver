package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/CircuitSavage/datadome-solver/internal/builder"
	ddcrypto "github.com/CircuitSavage/datadome-solver/internal/crypto"
	"github.com/CircuitSavage/datadome-solver/pkg/datadome"
)

func main() {
	site := flag.String("site", "", "Origin URL of the protected site (required)")
	proxy := flag.String("proxy", "", "HTTP proxy URL")
	profile := flag.String("profile", "chrome_win10", "Browser profile name")
	key := flag.String("key", "", "DDJS key from the target site (required for -solve and -encrypt)")
	cid := flag.String("cid", "", "Existing DataDome CID, if any")
	solve := flag.Bool("solve", false, "POST fingerprint to /include/tags.js")
	encrypt := flag.Bool("encrypt", false, "Print encrypted jspl instead of JSON")
	output := flag.String("output", "", "Write JSON payload to this file")
	flag.Parse()

	if *site == "" {
		fmt.Fprintln(os.Stderr, "error: -site is required")
		flag.Usage()
		os.Exit(2)
	}

	if *solve {
		if *key == "" {
			fmt.Fprintln(os.Stderr, "error: -key is required with -solve")
			os.Exit(2)
		}
		runSolve(*site, *key, *cid, *proxy, *profile)
		return
	}

	signals := builder.BuildPayload(builder.Options{
		Profile: *profile,
		URL:     *site,
		BPC:     1,
	})

	if *encrypt {
		if *key == "" {
			fmt.Fprintln(os.Stderr, "error: -key is required with -encrypt")
			os.Exit(2)
		}
		jspl, err := ddcrypto.Encrypt(signals, *key, *cid, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encrypt: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(jspl)
		return
	}

	m := make(map[string]any, len(signals))
	for _, s := range signals {
		m[s.Key] = s.Value
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json: %v\n", err)
		os.Exit(1)
	}
	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %d signals to %s\n", len(signals), *output)
		return
	}
	fmt.Print(string(data))
}

func runSolve(site, key, cid, proxy, profile string) {
	opts := []datadome.Option{
		datadome.WithDDJSKey(key),
		datadome.WithCID(cid),
		datadome.WithProfile(profile),
	}
	if proxy != "" {
		opts = append(opts, datadome.WithProxy(proxy))
	}

	client, err := datadome.New(site, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.Solve(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(result.Cookie)
}
