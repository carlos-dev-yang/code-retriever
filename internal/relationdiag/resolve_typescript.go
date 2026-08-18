package relationdiag

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const TypeScriptVersion = "6.0.3"

type TypeScriptResolverRequest struct {
	Protocol       string      `json:"protocol"`
	TypeScriptRoot string      `json:"typescript_root"`
	SourceRoot     string      `json:"source_root"`
	Files          []TSFile    `json:"files"`
	Candidates     []Candidate `json:"candidates"`
}

type TSFile struct {
	Path          string `json:"path"`
	Language      string `json:"language"`
	IndexedSHA256 string `json:"indexed_sha256"`
}

type typeScriptResolverResponse struct {
	Protocol      string  `json:"protocol"`
	ID            string  `json:"id"`
	Outcome       Outcome `json:"outcome"`
	TargetPath    string  `json:"target_path,omitempty"`
	TargetStart   int     `json:"target_start_byte,omitempty"`
	TargetEnd     int     `json:"target_end_byte,omitempty"`
	TypeScript    string  `json:"typescript_version"`
	ResolverScope string  `json:"resolver_scope"`
}

// ResolveTypeScript invokes a pinned local helper. It neither invokes npm nor
// lets the helper select files outside the indexed generation universe.
func ResolveTypeScript(ctx context.Context, node, helper, typescriptRoot, sourceRoot string, parents []Parent, universe []TSFile, candidates []Candidate) ([]Occurrence, []FileResolution, error) {
	var selected []Candidate
	for _, candidate := range candidates {
		if candidate.Language != "typescript" && candidate.Language != "tsx" {
			continue
		}
		selected = append(selected, candidate)
	}
	if len(selected) == 0 {
		return nil, tsNoOccurrenceStates(universe), nil
	}
	if node == "" || helper == "" || typescriptRoot == "" || sourceRoot == "" {
		return nil, nil, fmt.Errorf("TypeScript resolver executable, helper, toolchain, and source root are required")
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	fileList := append([]TSFile(nil), universe...)
	sort.Slice(fileList, func(i, j int) bool { return fileList[i].Path < fileList[j].Path })
	universePaths := map[string]bool{}
	for _, file := range fileList {
		if !validRelative(file.Path) || (file.Language != "typescript" && file.Language != "tsx") || len(file.IndexedSHA256) != 64 || universePaths[file.Path] {
			return nil, nil, fmt.Errorf("invalid indexed TypeScript universe")
		}
		universePaths[file.Path] = true
	}
	request := TypeScriptResolverRequest{Protocol: ProtocolVersion, TypeScriptRoot: typescriptRoot, SourceRoot: sourceRoot, Files: fileList, Candidates: selected}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}
	command := exec.CommandContext(ctx, node, helper)
	command.Stdin = bytes.NewReader(append(data, '\n'))
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := command.Start(); err != nil {
		return nil, nil, err
	}
	responses, err := readTypeScriptResponses(stdout, selected)
	if waitErr := command.Wait(); waitErr != nil {
		return nil, nil, fmt.Errorf("TypeScript resolver failed: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
	}
	if err != nil {
		return nil, nil, err
	}
	occurrences := make([]Occurrence, 0, len(selected))
	for _, candidate := range selected {
		response := responses[candidate.ID]
		occurrence := Occurrence{ID: candidate.ID, SourceParentID: candidate.SourceParentID, Path: candidate.Path, Language: candidate.Language, Kind: candidate.Kind, StartByte: candidate.StartByte, EndByte: candidate.EndByte, Outcome: response.Outcome, Resolver: "typescript-6.0.3-program-typechecker-v1", Metadata: candidate.Metadata}
		if response.Outcome == ResolvedUnique {
			if !universePaths[response.TargetPath] {
				return nil, nil, fmt.Errorf("TypeScript resolver target is outside indexed universe")
			}
			target, ok := ParentContaining(parents, response.TargetPath, response.TargetStart, response.TargetEnd)
			if !ok {
				occurrence.Outcome = ParentMappingFail
			} else {
				occurrence.TargetParentID = target.ID
			}
		}
		if err := occurrence.Validate(); err != nil {
			return nil, nil, err
		}
		occurrences = append(occurrences, occurrence)
	}
	states := make([]FileResolution, 0, len(fileList))
	for _, file := range fileList {
		states = append(states, FileResolution{Path: file.Path, Language: file.Language, Outcome: "RESOLVED"})
	}
	return occurrences, states, nil
}

func tsNoOccurrenceStates(files []TSFile) []FileResolution {
	states := make([]FileResolution, 0, len(files))
	for _, file := range files {
		states = append(states, FileResolution{Path: file.Path, Language: file.Language, Outcome: "NO_OCCURRENCES"})
	}
	return states
}

func readTypeScriptResponses(reader io.Reader, expected []Candidate) (map[string]typeScriptResolverResponse, error) {
	expectedIDs := map[string]bool{}
	for _, candidate := range expected {
		expectedIDs[candidate.ID] = true
	}
	responses := map[string]typeScriptResolverResponse{}
	scanner := bufio.NewScanner(reader)
	// Responses are structural metadata only, but permit a target filename plus
	// diagnostics without a scanner-token ceiling failure.
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var response typeScriptResolverResponse
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&response); err != nil {
			return nil, fmt.Errorf("invalid TypeScript resolver JSONL: %w", err)
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			return nil, fmt.Errorf("invalid TypeScript resolver JSONL trailing data")
		}
		if response.Protocol != ProtocolVersion || response.TypeScript != TypeScriptVersion || response.ResolverScope != "indexed-universe-v1" || response.ID == "" || !expectedIDs[response.ID] || !response.Outcome.Valid() {
			return nil, fmt.Errorf("invalid TypeScript resolver response")
		}
		if _, exists := responses[response.ID]; exists {
			return nil, fmt.Errorf("duplicate TypeScript resolver response")
		}
		if response.Outcome == ResolvedUnique {
			if !validRelative(response.TargetPath) || response.TargetStart < 0 || response.TargetEnd <= response.TargetStart {
				return nil, fmt.Errorf("invalid resolved TypeScript target")
			}
		} else if response.TargetPath != "" || response.TargetStart != 0 || response.TargetEnd != 0 {
			return nil, fmt.Errorf("unresolved TypeScript output has target")
		}
		responses[response.ID] = response
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(responses) != len(expectedIDs) {
		return nil, fmt.Errorf("missing TypeScript resolver responses")
	}
	return responses, nil
}

func DefaultTypeScriptHelper(repositoryRoot string) string {
	return filepath.Join(repositoryRoot, "tools", "relationdiag", "typescript-resolver.mjs")
}
