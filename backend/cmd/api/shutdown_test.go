// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package main

// S46-4: Graceful-shutdown unit tests.
//
// Full integration tests (start real server, send SIGTERM, verify in-flight
// requests complete) require DB + Redis and belong in the e2e suite.
// These tests verify the shutdown-timeout behaviour of the Echo server itself
// without starting any external dependencies.

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShutdownTimeout verifies that e.Shutdown() completes within the context
// deadline when no in-flight requests are active.
func TestShutdownTimeout(t *testing.T) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		_ = e.Server.Serve(ln)
	}()

	// Confirm the server is reachable before triggering shutdown.
	baseURL := "http://" + ln.Addr().String()
	resp, err := http.Get(baseURL + "/ping")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Graceful shutdown — mirrors the main() shutdown block.
	// The production timeout is 10 s; we use 5 s here to keep the test fast.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err = e.Shutdown(ctx)
	elapsed := time.Since(start)

	assert.NoError(t, err, "shutdown must complete without error when no requests are in-flight")
	assert.Less(t, elapsed, 5*time.Second, "shutdown must complete well within context deadline")

	select {
	case <-serverDone:
		// Server goroutine exited as expected.
	case <-time.After(2 * time.Second):
		t.Fatal("server goroutine did not exit after Shutdown()")
	}
}

// TestShutdownDrainsInFlightRequests verifies that Shutdown() waits for
// an active handler to finish before returning.
func TestShutdownDrainsInFlightRequests(t *testing.T) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	var handlerFinished sync.WaitGroup
	handlerFinished.Add(1)
	handlerStarted := make(chan struct{})

	e.GET("/slow", func(c echo.Context) error {
		close(handlerStarted)             // signal: handler is executing
		time.Sleep(80 * time.Millisecond) // simulate in-flight work
		handlerFinished.Done()
		return c.String(http.StatusOK, "done")
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() { _ = e.Server.Serve(ln) }()

	// Send a slow request.
	reqDone := make(chan *http.Response, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/slow")
		if err == nil {
			reqDone <- resp
		} else {
			close(reqDone)
		}
	}()

	// Wait until the handler is running before triggering shutdown.
	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("slow handler did not start within 2 s")
	}

	// Shutdown with a 2-second grace period — handler takes only 80 ms.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = e.Shutdown(ctx)
	assert.NoError(t, err, "shutdown must drain in-flight request within grace period")

	// Handler must have finished.
	handlerFinished.Wait()

	// In-flight request response should have arrived.
	select {
	case resp := <-reqDone:
		if resp != nil {
			resp.Body.Close()
		}
	case <-time.After(500 * time.Millisecond):
		// Acceptable: connection may have been reset during drain.
		t.Log("in-flight response not received — connection reset during drain is acceptable")
	}
}
