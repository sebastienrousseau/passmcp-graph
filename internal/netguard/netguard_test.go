// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package netguard

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDenyRefusesEveryRequestAndRestores(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()
	restore := Deny()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	if _, err := http.DefaultClient.Do(req); !errors.Is(err, ErrOffline) {
		restore()
		t.Fatalf("want ErrOffline, got %v", err)
	}
	restore()
	req, _ = http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("restore did not put the transport back: %v", err)
	}
	_ = resp.Body.Close()
}
