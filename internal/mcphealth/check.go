package mcphealth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

var initializeParams = json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"qsdev-health","version":"1.0"}}`)

var toolsListParams = json.RawMessage(`{}`)

// CheckServer probes a single MCP server and returns its health status.
// The provided context controls cancellation and timeout; callers should use
// context.WithTimeout to enforce a deadline.
func CheckServer(ctx context.Context, cfg ServerConfig) *ServerHealth {
	h := &ServerHealth{Name: cfg.Name}

	prereqs := checkPrerequisites(cfg)
	h.Prerequisites = prereqs

	unmet := false
	for _, p := range prereqs {
		if !p.Met {
			unmet = true
			break
		}
	}

	if unmet {
		h.Status = StatusDegraded
		h.Error = "one or more prerequisites not met"
		return h
	}

	start := time.Now()

	if cfg.URL != "" {
		return checkHTTPServer(ctx, cfg, h, start)
	}

	proc, err := startServer(cfg.Command, cfg.Args, cfg.Env)
	if err != nil {
		h.Status = StatusUnreachable
		h.Error = fmt.Sprintf("starting server: %s", err)
		return h
	}
	defer proc.Close()

	type probeResult struct {
		status    string
		err       string
		toolCount int
	}

	ch := make(chan probeResult, 1)
	go func() {
		if _, err := proc.SendRequest(1, "initialize", initializeParams); err != nil {
			ch <- probeResult{status: StatusUnreachable, err: fmt.Sprintf("initialize: %s", err)}
			return
		}

		result, err := proc.SendRequest(2, "tools/list", toolsListParams)
		if err != nil {
			ch <- probeResult{status: StatusUnreachable, err: fmt.Sprintf("tools/list: %s", err)}
			return
		}

		ch <- probeResult{status: StatusHealthy, toolCount: countTools(result)}
	}()

	var r probeResult
	select {
	case r = <-ch:
	case <-ctx.Done():
		r = probeResult{status: StatusUnreachable, err: "health check timed out"}
	}

	h.Status = r.status
	h.Error = r.err
	h.ToolCount = r.toolCount

	h.ResponseMs = time.Since(start).Milliseconds()
	return h
}

// CheckAll probes all servers in parallel and returns an aggregated report.
// The provided context controls cancellation and timeout for each individual
// server check.
func CheckAll(ctx context.Context, servers map[string]ServerConfig) *HealthReport {
	report := &HealthReport{
		TotalCount: len(servers),
		CheckedAt:  time.Now(),
	}

	if len(servers) == 0 {
		report.Servers = []ServerHealth{}
		return report
	}

	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]ServerHealth, len(names))
	var wg sync.WaitGroup

	for i, name := range names {
		wg.Add(1)
		go func(idx int, cfg ServerConfig) {
			defer wg.Done()
			h := CheckServer(ctx, cfg)
			results[idx] = *h
		}(i, servers[name])
	}

	wg.Wait()

	report.Servers = results
	for _, s := range results {
		if s.Status == StatusHealthy {
			report.HealthyCount++
		}
	}

	return report
}

func checkPrerequisites(cfg ServerConfig) []PrerequisiteStatus {
	var prereqs []PrerequisiteStatus

	for _, envKey := range cfg.RequiredEnv {
		_, set := os.LookupEnv(envKey)
		p := PrerequisiteStatus{
			Name: envKey,
			Type: "env",
			Met:  set,
		}
		if !set {
			p.Detail = fmt.Sprintf("environment variable %s is not set", envKey)
		}
		prereqs = append(prereqs, p)
	}

	return prereqs
}

type toolsListResult struct {
	Tools []json.RawMessage `json:"tools"`
}

func countTools(result json.RawMessage) int {
	var tlr toolsListResult
	if err := json.Unmarshal(result, &tlr); err != nil {
		return 0
	}
	return len(tlr.Tools)
}

func checkHTTPServer(ctx context.Context, cfg ServerConfig, h *ServerHealth, start time.Time) *ServerHealth {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		h.Status = StatusUnreachable
		h.Error = fmt.Sprintf("building request: %s", err)
		h.ResponseMs = time.Since(start).Milliseconds()
		return h
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.Status = StatusUnreachable
		h.Error = fmt.Sprintf("connecting: %s", err)
		h.ResponseMs = time.Since(start).Milliseconds()
		return h
	}
	resp.Body.Close()

	h.Status = StatusHealthy
	h.ResponseMs = time.Since(start).Milliseconds()
	return h
}
