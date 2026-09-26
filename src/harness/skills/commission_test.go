package skills

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

// D-C-5: the commission id enters the sealed origin block (opaque
// attribution, recorded verbatim by L7) and never the model-facing
// payload; absence leaves no key; a non-UUID refuses at instantiation.
func TestCommissionTravelsInOriginOnly(t *testing.T) {
	b := newBundle(t)
	req := b.request(t)
	req.Commission = "b1be6f86-2ecd-451f-9411-95f1f32fd501"
	path, err := Instantiate(b.catalogPath, "investigate-cve@1", req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var env struct {
		Origin  map[string]string `json:"origin"`
		Payload string            `json:"payload"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env.Origin["commission"] != req.Commission {
		t.Fatalf("origin: %v", env.Origin)
	}
	if strings.Contains(env.Payload, req.Commission) {
		t.Fatal("the commission must never be model-facing task data")
	}
	// Absent: no key at all (never an empty string).
	req2 := b.request(t)
	path2, err := Instantiate(b.catalogPath, "investigate-cve@1", req2)
	if err != nil {
		t.Fatal(err)
	}
	raw2, _ := os.ReadFile(path2)
	var env2 struct {
		Origin map[string]string `json:"origin"`
	}
	_ = json.Unmarshal(raw2, &env2)
	if _, ok := env2.Origin["commission"]; ok {
		t.Fatal("absent commission must leave no key")
	}
	// Shape only, fail closed.
	req3 := b.request(t)
	req3.Commission = "C-123"
	if _, err := Instantiate(b.catalogPath, "investigate-cve@1", req3); !errors.Is(err, ErrInputs) {
		t.Fatalf("non-UUID commission: %v", err)
	}
}
