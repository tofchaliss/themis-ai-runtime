package ratchet

// Door admission resolution — the C-1 remediation. D-L11-5 §2: "at
// comparison time L11 RESOLVES the owning door's record" — the
// observation is constructed HERE, from the door registry's actual
// bytes, never accepted as caller-asserted JSON. L11 still owns no
// admission fact: the observation records what the door's bytes
// said, grounded in their hash so cold reconstruction can re-verify
// it (ReverifyAdmission, used by Reconstruct). One shared derivation
// (deriveObservation) serves both the live path and re-verification,
// so the two can never drift apart.

import (
	"encoding/json"
	"fmt"
)

// doorHashFields is the closed v1 door table: for each known door,
// the field carrying the entry's artifact commitment. Adding a door
// is a reviewed code change, never data.
var doorHashFields = map[string]string{
	"l9-catalog":            "composition_sha256",
	"l10-contract-registry": "contract_sha256",
	"l11-criteria":          "artifact_sha256",
	"l11-regression-sets":   "artifact_sha256",
}

// deriveObservation derives the (name, version) admission
// observation from door registry bytes. Returns (nil, nil) when the
// entry does not exist — admission is then not establishable, and
// L11 invents nothing. Door/DoorRegistryHash/ObservedAt are filled
// by the caller.
func deriveObservation(door string, raw []byte, name string, version int) (*AdmissionObservation, error) {
	hashField, ok := doorHashFields[door]
	if !ok {
		return nil, fmt.Errorf("%w: unknown door %q — doors are a closed reviewed table", ErrResolve, door)
	}
	var reg struct {
		Entries []json.RawMessage `json:"entries"`
	}
	// Deliberately lenient on unknown top-level fields: each door
	// owns its own schema; L11 reads the entries list only.
	if err := json.Unmarshal(raw, &reg); err != nil {
		return nil, fmt.Errorf("%w: door registry: %v", ErrResolve, err)
	}
	var found *AdmissionObservation
	maxActiveVersion := 0
	for _, e := range reg.Entries {
		var entry struct {
			Name    string `json:"name"`
			Version int    `json:"version"`
			State   string `json:"state"`
		}
		if err := json.Unmarshal(e, &entry); err != nil {
			return nil, fmt.Errorf("%w: door registry: bad entry: %v", ErrResolve, err)
		}
		if entry.Name == name && entry.State == "active" && entry.Version > maxActiveVersion {
			maxActiveVersion = entry.Version
		}
		if entry.Name == name && entry.Version == version {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(e, &fields); err != nil {
				return nil, fmt.Errorf("%w: door registry: bad entry: %v", ErrResolve, err)
			}
			var h string
			hraw, ok := fields[hashField]
			if !ok {
				return nil, fmt.Errorf("%w: door registry entry %s@%d lacks %s", ErrResolve, name, version, hashField)
			}
			if err := json.Unmarshal(hraw, &h); err != nil || !shaSyntax.MatchString(h) {
				return nil, fmt.Errorf("%w: door registry entry %s@%d: %s is not a sha256 digest", ErrResolve, name, version, hashField)
			}
			found = &AdmissionObservation{Door: door, Name: entry.Name, Version: entry.Version, ArtifactSHA256: h, State: entry.State}
		}
	}
	if found == nil {
		return nil, nil
	}
	found.CurrentActive = found.State == "active" && found.Version == maxActiveVersion
	return found, nil
}

// ObserveAdmission resolves (name, version) against the door
// registry file and returns the grounded observation. (nil, nil)
// means admission is not establishable — the caller refuses with
// unadmitted-baseline. Errors are machinery failures.
func ObserveAdmission(door, registryPath, name string, version int, observedAt string) (*AdmissionObservation, error) {
	raw, err := readGoverned(registryPath, maxRegistryBytes, ErrResolve)
	if err != nil {
		return nil, err
	}
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: door registry %s: %v", ErrResolve, registryPath, err)
	}
	obs, err := deriveObservation(door, raw, name, version)
	if err != nil || obs == nil {
		return obs, err
	}
	obs.DoorRegistryHash = hashBytes(raw)
	obs.ObservedAt = observedAt
	return obs, nil
}

// ReverifyAdmission re-derives a recorded observation from supplied
// door registry bytes (D-L11-5 §2 re-verification, exercised by
// Reconstruct). Returns nil when the observation re-derives
// identically; otherwise the disagreement descriptions. The caller
// checks the byte hash FIRST — hash-mismatched bytes are a
// missing-input, not a disagreement.
func ReverifyAdmission(obs AdmissionObservation, doorRegistryBytes []byte) []string {
	re, err := deriveObservation(obs.Door, doorRegistryBytes, obs.Name, obs.Version)
	if err != nil {
		return []string{fmt.Sprintf("door registry bytes no longer support re-derivation: %v", err)}
	}
	if re == nil {
		return []string{fmt.Sprintf("door registry bytes contain no entry %s@%d — the recorded observation cannot be re-derived", obs.Name, obs.Version)}
	}
	var disc []string
	if re.ArtifactSHA256 != obs.ArtifactSHA256 {
		disc = append(disc, "re-derived admission artifact hash disagrees with the recorded observation")
	}
	if re.State != obs.State {
		disc = append(disc, "re-derived admission state disagrees with the recorded observation")
	}
	if re.CurrentActive != obs.CurrentActive {
		disc = append(disc, "re-derived current-active disagrees with the recorded observation")
	}
	return disc
}
