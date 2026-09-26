package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/integrations/themis/contracts"
)

const (
	findingID = "b1be6f86-2ecd-451f-9411-95f1f32fd501"
	productID = "0f8fad5b-d9cb-469f-a165-70867728950e"
	readKey   = "tk_read_0123456789abcdef"
)

// A Governance/Registry stand-in serving the full FindingView — with
// positions and proposals present — so the projection can be proven.
func standIn(t *testing.T, seenKey *string, idInBody string) (*httptest.Server, *httptest.Server) {
	t.Helper()
	gov := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seenKey = r.Header.Get("X-API-Key")
		if r.URL.Path != "/api/v1/findings/"+findingID {
			http.Error(w, `{"title":"not found"}`, 404)
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + idInBody + `","release_id":"r-1","faultline_id":"f-1","cve":"CVE-2024-1000","stage":"identified",
		  "components":[{"purl":"pkg:golang/example.com/dep@v1.0.0","name":"dep","version":"v1.0.0","ecosystem":"golang","claim_class":"carrier","detection_origin":"discovery","verdict_state":"open"}],
		  "current_position":{"version":3,"stance":"accepted_risk","rationale":"SECRET-DECISION"},
		  "positions":[{"version":1,"stance":"affected"}],
		  "proposals":[{"proposal_id":"p-1","stance":"mitigated","rationale":"AI says fine"}]}`))
	}))
	reg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/products/"+productID {
			http.Error(w, `{"title":"not found"}`, 404)
			return
		}
		_, _ = w.Write([]byte(`{"id":"` + productID + `","name":"demo-vuln-app","secret_field":"x"}`))
	}))
	t.Cleanup(gov.Close)
	t.Cleanup(reg.Close)
	return gov, reg
}

func contractFor(t *testing.T, gov, reg *httptest.Server) *contracts.Contract {
	t.Helper()
	raw := `{"version":1,"governance_base_url":"` + gov.URL + `","registry_base_url":"` + reg.URL + `","governance_spec_sha256":"` + strings.Repeat("a", 64) + `","registry_spec_sha256":"` + strings.Repeat("b", 64) + `","themis_commit":"` + strings.Repeat("c", 40) + `"}`
	c, err := contracts.Parse([]byte(raw), "test")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestReadDoorProjectsAndAuthenticates(t *testing.T) {
	var seen string
	gov, reg := standIn(t, &seen, findingID)
	c, err := New(contractFor(t, gov, reg), readKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Read("finding", findingID)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, forbidden := range []string{"current_position", "positions", "proposals", "SECRET-DECISION", "AI says fine", "detection_origin", "verdict_state"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("projection served %q:\n%s", forbidden, s)
		}
	}
	var v map[string]any
	_ = json.Unmarshal(b, &v)
	if v["id"] != findingID || v["cve"] != "CVE-2024-1000" || v["stage"] != "identified" || v["release_id"] != "r-1" || v["faultline_id"] != "f-1" {
		t.Fatalf("projected finding: %s", s)
	}
	comps := v["components"].([]any)
	if len(comps) != 1 || comps[0].(map[string]any)["purl"] != "pkg:golang/example.com/dep@v1.0.0" {
		t.Fatalf("components: %v", comps)
	}
	if seen != readKey {
		t.Fatalf("read key not presented: %q", seen)
	}
	if strings.Contains(s, readKey) {
		t.Fatal("the credential leaked into the served bytes")
	}
	p, err := c.Read("product", productID)
	if err != nil || string(p) != `{"id":"`+productID+`","name":"demo-vuln-app"}` {
		t.Fatalf("product: %s %v", p, err)
	}
	if c.ContractHash() != contractFor(t, gov, reg).Hash {
		t.Fatal("contract hash")
	}
}

func TestReadDoorFailsClosed(t *testing.T) {
	var seen string
	gov, reg := standIn(t, &seen, findingID)
	c, _ := New(contractFor(t, gov, reg), readKey, nil)
	must := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: %v", name, err)
		}
	}
	_, err := c.Read("finding", "0f8fad5b-d9cb-469f-a165-70867728950f") // unknown → 404
	must("404", err)
	_, err = c.Read("finding", "FIND-2026-0001") // not a UUID
	must("shape", err)
	_, err = c.Read("position", findingID)
	must("kind", err)
	// Identity mismatch: the body says another Finding.
	gov2, reg2 := standIn(t, &seen, "b1be6f86-2ecd-451f-9411-95f1f32fd502")
	c2, _ := New(contractFor(t, gov2, reg2), readKey, nil)
	_, err = c2.Read("finding", findingID)
	must("identity", err)
	if err == nil || !strings.Contains(err.Error(), "identity") {
		t.Errorf("identity mismatch must be named: %v", err)
	}
	// Down: refused, and the error never carries the header name/key.
	gov.Close()
	_, err = c.Read("finding", findingID)
	must("down", err)
	if err != nil && (strings.Contains(err.Error(), readKey) || strings.Contains(err.Error(), "X-API-Key")) {
		t.Errorf("credential in error: %v", err)
	}
}
