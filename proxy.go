package main

import (
	"net/http"

	"github.com/IceRhymers/databricks-claude/pkg/proxy"
	"github.com/IceRhymers/databricks-claude/pkg/tokencache"
)

// newProxy creates the HTTP proxy handler backed by pkg/proxy.NewServer.
// HTTP only — UCMetricsTable is always empty for Cursor.
func newProxy(gatewayURL, otelUpstream string, tp *tokencache.TokenProvider, verbose bool) http.Handler {
	cfg := &proxy.Config{
		InferenceUpstream: gatewayURL,
		OTELUpstream:      otelUpstream,
		UCMetricsTable:    "", // Cursor does not emit metrics
		UCLogsTable:       "",
		TokenSource:       tp,
		Verbose:           verbose,
	}
	return proxy.NewServer(cfg)
}
