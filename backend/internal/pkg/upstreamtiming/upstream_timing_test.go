package upstreamtiming

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestTracePreservesResponseAndContext(t *testing.T) {
	const payload = "data: {\"choices\":[{}]}\n\ndata: [DONE]\n\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		input, _ := io.ReadAll(r.Body)
		if string(input) != "original-body" || r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Error("request modified")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if _, err := io.WriteString(w, payload); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()
	var snapshots []Snapshot
	trace := New(func(s Snapshot) { snapshots = append(snapshots, s) })
	ctx := context.WithoutCancel(trace.Context(context.Background()))
	req, _ := http.NewRequestWithContext(ctx, "POST", server.URL, strings.NewReader("original-body"))
	req.Header.Set("Authorization", "Bearer test-secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	trace.Wrap(resp)
	got, err := io.ReadAll(resp.Body)
	if err != nil || string(got) != payload {
		t.Fatalf("response changed: %q %v", got, err)
	}
	trace.FirstSSE()
	trace.FirstSSE()
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	trace.Finish(false)
	if len(snapshots) != 2 {
		t.Fatalf("snapshot count %d", len(snapshots))
	}
	names := map[string]bool{}
	for _, e := range snapshots[1].Events {
		names[e.Name] = true
	}
	for _, name := range []string{"get_conn", "got_conn", "wrote_request", "first_response_byte", "headers_ready", "first_body_read", "first_sse", "finished"} {
		if !names[name] {
			t.Errorf("missing %s", name)
		}
	}
}

func TestConcurrentCallbacksAreBounded(t *testing.T) {
	trace := New(nil)
	hooks := httptrace.ContextClientTrace(trace.Context(context.Background()))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				hooks.GetConn("secret-host-not-logged")
				hooks.GotConn(httptrace.GotConnInfo{Reused: true})
				hooks.WroteRequest(httptrace.WroteRequestInfo{Err: errors.New("secret-not-logged")})
				trace.snapshot("test")
			}
		}()
	}
	wg.Wait()
	s := trace.snapshot("test")
	if len(s.Events) != 128 || s.Dropped == 0 {
		t.Fatal("unbounded trace")
	}
	copyBefore := append([]Event(nil), s.Events...)
	trace.record("extra", false, false)
	if !reflect.DeepEqual(copyBefore, s.Events) {
		t.Fatal("mutable snapshot")
	}
}

func TestHTTP2ConnectionReuse(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.WriteString(w, "data: {}\n\n"); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()
	client := srv.Client()
	defer client.CloseIdleConnections()
	for i := 0; i < 2; i++ {
		trace := New(nil)
		req, _ := http.NewRequestWithContext(trace.Context(context.Background()), "GET", srv.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		trace.Wrap(resp)
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		s := trace.snapshot("test")
		if s.Protocol != "HTTP/2.0" {
			t.Fatalf("unexpected protocol %s", s.Protocol)
		}
		if i == 1 {
			reused := false
			for _, e := range s.Events {
				reused = reused || (e.Name == "got_conn" && e.Reused)
			}
			if !reused {
				t.Fatal("reuse not observed")
			}
		}
	}
}
