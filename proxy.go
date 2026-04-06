package main

import (
	"net/http"

	"github.com/IceRhymers/databricks-claude/pkg/proxy"
	"github.com/IceRhymers/databricks-claude/pkg/tokencache"
)

// newProxy creates the HTTP proxy handler backed by pkg/proxy.NewServer.
func newProxy(gatewayURL string, tp *tokencache.TokenProvider, verbose bool, apiKey, tlsCertFile, tlsKeyFile string) http.Handler {
	cfg := &proxy.Config{
		InferenceUpstream: gatewayURL,
		TokenSource:       tp,
		Verbose:           verbose,
		APIKey:            apiKey,
		TLSCertFile:       tlsCertFile,
		TLSKeyFile:        tlsKeyFile,
	}
	return proxy.NewServer(cfg)
}
