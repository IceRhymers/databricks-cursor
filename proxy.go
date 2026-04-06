package main

import (
	"net/http"

	"github.com/IceRhymers/databricks-claude/pkg/proxy"
	"github.com/IceRhymers/databricks-claude/pkg/tokencache"
)

// newProxy creates the HTTP proxy handler backed by pkg/proxy.NewServer.
func newProxy(gatewayURL string, tp *tokencache.TokenProvider, verbose bool) http.Handler {
	cfg := &proxy.Config{
		InferenceUpstream: gatewayURL,
		TokenSource:       tp,
		Verbose:           verbose,
	}
	return proxy.NewServer(cfg)
}
