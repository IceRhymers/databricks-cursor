package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/IceRhymers/databricks-claude/pkg/authcheck"
	"github.com/IceRhymers/databricks-claude/pkg/proxy"
	"github.com/IceRhymers/databricks-claude/pkg/tokencache"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func main() {
	// --- Flags ---
	portFlag := flag.Int("port", 0, "Port to listen on (0 = auto-assign and persist)")
	profileFlag := flag.String("profile", "DEFAULT", "Databricks CLI profile")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.BoolVar(verbose, "v", false, "Enable verbose logging (shorthand)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	printEnv := flag.Bool("print-env", false, "Print environment variables and exit")
	upstream := flag.String("upstream", "", "Override inference upstream URL")
	proxyAPIKey := flag.String("proxy-api-key", "", "Require this API key on all incoming requests (empty = no auth)")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file (requires --tls-key)")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file (requires --tls-cert)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("databricks-cursor %s\n", Version)
		os.Exit(0)
	}

	// Default: discard all logs (silent by default).
	log.SetOutput(io.Discard)

	if *verbose {
		log.SetOutput(os.Stderr)
	}

	if *printEnv {
		fmt.Printf("DATABRICKS_PROFILE=%s\n", *profileFlag)
		os.Exit(0)
	}

	// --- Auth check ---
	if err := authcheck.EnsureAuthenticated(*profileFlag); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("databricks-cursor: authentication failed: %v", err)
	}

	// --- TLS validation ---
	if err := proxy.ValidateTLSConfig(*tlsCert, *tlsKey); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("databricks-cursor: %v", err)
	}

	// --- Startup security checks ---
	for _, w := range proxy.SecurityChecks() {
		fmt.Fprintln(os.Stderr, w)
	}

	// --- Port resolution ---
	cfg, err := LoadOrCreate(*portFlag, *profileFlag)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("databricks-cursor: %v", err)
	}

	// --- Bind listener ---
	listenAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("databricks-cursor: cannot listen on %s: %v", listenAddr, err)
	}

	// If port was 0 (auto-assigned), save the actual port.
	actualPort := ln.Addr().(*net.TCPAddr).Port
	if cfg.Port == 0 {
		cfg.Port = actualPort
		if err := saveConfig(cfg); err != nil {
			log.Printf("databricks-cursor: warning: cannot save config: %v", err)
		}
	}

	// --- Discover gateway URL ---
	gatewayURL := *upstream
	if gatewayURL == "" {
		host, err := DiscoverHost(*profileFlag)
		if err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("databricks-cursor: %v", err)
		}
		gatewayURL = ConstructGatewayURL(host)
	}
	log.Printf("databricks-cursor: inference upstream: %s", gatewayURL)

	// --- Token provider ---
	fetcher := &databricksFetcher{profile: *profileFlag}
	tp := tokencache.NewTokenProvider(fetcher)

	// --- Start proxy ---
	handler := newProxy(gatewayURL, tp, *verbose, *proxyAPIKey, *tlsCert, *tlsKey)
	useTLS := *tlsCert != "" && *tlsKey != ""
	server := &http.Server{Handler: handler}
	if useTLS {
		go func() {
			if err := server.ServeTLS(ln, *tlsCert, *tlsKey); err != nil && err != http.ErrServerClosed {
				log.Printf("databricks-cursor: server error: %v", err)
			}
		}()
	} else {
		go func() {
			if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("databricks-cursor: server error: %v", err)
			}
		}()
	}

	// --- Print instructions ---
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	fmt.Fprintf(os.Stderr, "\ndatabricks-cursor is running on %s://127.0.0.1:%d\n\n", scheme, actualPort)
	fmt.Fprintf(os.Stderr, "Cursor setup (one-time):\n")
	fmt.Fprintf(os.Stderr, "  1. Open Cursor Settings > Models\n")
	fmt.Fprintf(os.Stderr, "  2. Set \"Override OpenAI Base URL\" to: %s://127.0.0.1:%d/v1\n", scheme, actualPort)
	fmt.Fprintf(os.Stderr, "  3. Set \"OpenAI API Key\" to any non-empty value (e.g., \"databricks-proxy\")\n\n")
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop.\n")

	// --- Block on signal ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Fprintf(os.Stderr, "\ndatabricks-cursor: shutting down...\n")
	server.Shutdown(context.Background())
}
