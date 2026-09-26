package contracts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const good = `{"version":1,"governance_base_url":"http://127.0.0.1:8083","registry_base_url":"http://127.0.0.1:8082","governance_spec_sha256":"79b107be6f3b1c40f02675207cf7b64b8d899c23ca7a9309df62d88e375f399e","registry_spec_sha256":"1d87cbd8d9fbb18f461b27c78be1b1c7c6aa756ac85852ce1303cf2e752df31d","themis_commit":"1414997bf4564978da5d1b046ddb4693839c063c"}`

func TestContractLoadsAndHashesExactBytes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(p, []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.GovernanceBaseURL != "http://127.0.0.1:8083" || c.ThemisCommit[:7] != "1414997" || len(c.Hash) != 64 || string(c.Raw) != good {
		t.Fatalf("%+v", c)
	}
	// One byte of whitespace is a different artifact.
	c2, err := Parse([]byte(good+"\n"), "x")
	if err != nil || c2.Hash == c.Hash {
		t.Fatalf("hash must cover exact bytes: %v %v", err, c2.Hash == c.Hash)
	}
}

func TestContractRefusals(t *testing.T) {
	cases := map[string]string{
		"duplicate key":   strings.Replace(good, `"version":1,`, `"version":1,"version":1,`, 1),
		"unknown field":   strings.Replace(good, `"version":1,`, `"version":1,"data":"x",`, 1),
		"trailing":        good + "{}",
		"version":         strings.Replace(good, `"version":1`, `"version":2`, 1),
		"url with query":  strings.Replace(good, `8083"`, `8083?x=1"`, 1),
		"url with creds":  strings.Replace(good, `http://127.0.0.1:8083`, `http://k:s@127.0.0.1:8083`, 1),
		"url scheme":      strings.Replace(good, `http://127.0.0.1:8082`, `ftp://127.0.0.1:8082`, 1),
		"spec hash short": strings.Replace(good, `79b107be6f3b1c40f02675207cf7b64b8d899c23ca7a9309df62d88e375f399e`, `d71707fa`, 1),
		"commit short":    strings.Replace(good, `1414997bf4564978da5d1b046ddb4693839c063c`, `1414997`, 1),
	}
	for name, raw := range cases {
		if _, err := Parse([]byte(raw), name); !errors.Is(err, ErrContract) {
			t.Errorf("%s: %v", name, err)
		}
	}
	dir := t.TempDir()
	if _, err := Load(filepath.Join(dir, "missing.json")); !errors.Is(err, ErrContract) {
		t.Errorf("missing: %v", err)
	}
	real := filepath.Join(dir, "real.json")
	_ = os.WriteFile(real, []byte(good), 0o644)
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(real, link); err == nil {
		if _, err := Load(link); !errors.Is(err, ErrContract) {
			t.Errorf("symlink: %v", err)
		}
	}
}
