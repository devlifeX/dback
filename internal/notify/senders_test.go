package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWebhookSenderHTTPSOnly(t *testing.T) {
	s := WebhookSender{}
	err := s.Validate([]byte(`{"url":"http://example.com/x"}`))
	if err == nil {
		t.Fatal("expected http url rejection")
	}
}

func TestWebhookSenderDeliver(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "DBack") {
			t.Fatalf("unexpected body %s", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{target: srv.URL, base: client.Transport}

	cfg, _ := json.Marshal(WebhookConfig{URL: "https://example.com/hook", Method: "POST"})
	sender := WebhookSender{Client: client}
	if err := sender.Send(context.Background(), cfg, TestMessage()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", hits)
	}
}

func TestSlackSenderDeliver(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(SlackConfig{WebhookURL: srv.URL})
	sender := SlackSender{Client: srv.Client()}
	if err := sender.Send(context.Background(), cfg, TestMessage()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", hits)
	}
}

func TestTelegramRedact(t *testing.T) {
	raw := []byte(`{"token":"secret","chat_id":"1"}`)
	out := TelegramSender{}.Redact(raw)
	if strings.Contains(string(out), "secret") {
		t.Fatalf("token not redacted: %s", out)
	}
}

type rewriteTransport struct {
	target string
	base   http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL, _ = req.URL.Parse(t.target)
	if t.base == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return t.base.RoundTrip(req)
}
