package relationdiag

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"cidx/internal/buildinfo"
	"cidx/internal/config"
	"cidx/internal/eval"
	"cidx/internal/store"
	_ "modernc.org/sqlite"
)

type BuildRequest struct {
	RunID                     string
	EvaluationRoot            string
	SourceRoot                string
	RepositoryRoot            string
	Corpus                    eval.VerifiedCorpus
	CorpusManifestFingerprint string
	Index                     store.IndexSnapshot
	Parents                   store.SemanticParentSnapshot
	Node                      string
	TypeScriptHelper          string
	TypeScriptRoot            string
	TypeScriptLock            string
	TSConfig                  string
	Executable                string
	Go                        string
	Reproof                   func(context.Context) (store.IndexSnapshot, store.SemanticParentSnapshot, error)
}

type GraphManifest struct {
	SchemaVersion                 int                 `json:"schema_version"`
	Kind                          string              `json:"kind"`
	RunID                         string              `json:"run_id"`
	CreatedAt                     string              `json:"created_at"`
	Corpus                        eval.VerifiedCorpus `json:"corpus"`
	CorpusManifestFingerprint     string              `json:"corpus_manifest_fingerprint"`
	IndexGeneration               int64               `json:"index_generation"`
	IndexManifestSHA256           string              `json:"index_manifest_sha256"`
	Profiles                      map[string]string   `json:"profiles"`
	SemanticParentInventorySHA256 string              `json:"semantic_parent_inventory_sha256"`
	IndexedFileInventorySHA256    string              `json:"indexed_file_inventory_sha256"`
	ResolverPolicy                map[string]string   `json:"resolver_policy"`
	Implementation                buildinfo.Info      `json:"implementation"`
	Counts                        map[string]int      `json:"counts"`
	DatabaseSHA256                string              `json:"database_sha256"`
	LogicalGraphSHA256            string              `json:"logical_graph_sha256"`
}

type BuildResult struct {
	RunID          string `json:"run_id"`
	Reference      string `json:"reference"`
	DatabaseSHA256 string `json:"database_sha256"`
	Occurrences    int    `json:"occurrences"`
}

func Build(ctx context.Context, request BuildRequest) (BuildResult, error) {
	if err := validateBuildRequest(request); err != nil {
		return BuildResult{}, err
	}
	parents, err := ParentInventory(request.Parents.Parents)
	if err != nil {
		return BuildResult{}, err
	}
	if request.Index.Applied.ActiveGeneration != request.Parents.Generation || request.Index.Applied.ManifestSHA256 != request.Parents.ManifestSHA256 {
		return BuildResult{}, fmt.Errorf("NON_REPRODUCIBLE_RUN")
	}
	if err := verifyIndexedFiles(request.SourceRoot, request.Index.Files); err != nil {
		return BuildResult{}, err
	}
	if err := verifyParentBodies(request.SourceRoot, parents); err != nil {
		return BuildResult{}, err
	}
	candidates, preResolved, extractedFiles, err := extractAll(ctx, request.SourceRoot, parents, request.Index.Files)
	if err != nil {
		return BuildResult{}, err
	}
	goOccurrences, goFiles, err := ResolveGo(ctx, request.SourceRoot, parents, candidates, request.Index.Files, request.Go)
	if err != nil {
		return BuildResult{}, err
	}
	tsOccurrences, tsFiles, err := ResolveTypeScript(ctx, request.Node, request.TypeScriptHelper, request.TypeScriptRoot, request.SourceRoot, parents, typeScriptUniverse(request.Index.Files), candidates)
	if err != nil {
		return BuildResult{}, err
	}
	occurrences := append(append(append([]Occurrence{}, preResolved...), goOccurrences...), tsOccurrences...)
	if err := validateOccurrenceSet(occurrences); err != nil {
		return BuildResult{}, err
	}
	files := mergeFileResolutions(extractedFiles, goFiles, tsFiles)
	parentHash, err := inventoryHash(parents)
	if err != nil {
		return BuildResult{}, err
	}
	fileHash, err := indexedFileHash(request.Index.Files)
	if err != nil {
		return BuildResult{}, err
	}
	manifest := GraphManifest{
		SchemaVersion: SchemaVersion, Kind: "cidx.relation_graph.v3", RunID: request.RunID, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Corpus: request.Corpus, CorpusManifestFingerprint: request.CorpusManifestFingerprint,
		IndexGeneration: request.Parents.Generation, IndexManifestSHA256: request.Parents.ManifestSHA256,
		Profiles:                      map[string]string{"index": string(request.Index.Applied.Fingerprints.Index), "canonical_text": string(request.Index.Applied.Fingerprints.CanonicalText), "source": string(request.Index.Applied.Fingerprints.Source), "vector_space": string(request.Index.Applied.Fingerprints.VectorSpace), "vector_storage": string(request.Index.Applied.Fingerprints.VectorStorage)},
		SemanticParentInventorySHA256: parentHash, IndexedFileInventorySHA256: fileHash,
		ResolverPolicy: map[string]string{"protocol": ProtocolVersion, "identity": IdentityPolicyID, "metadata": MetadataPolicyID, "go": "go/packages-go/types-v1", "typescript": "typescript-6.0.3-program-typechecker-v1", "typescript_version": TypeScriptVersion, "schema": "relation-sidecar-v3"},
		Implementation: buildinfo.Current(), Counts: map[string]int{"semantic_parents": len(parents), "parent_traits": len(parents), "relation_occurrences": len(occurrences), "file_resolution": len(files)},
	}
	if err := bindToolchains(&manifest, request); err != nil {
		return BuildResult{}, err
	}
	logical, err := logicalGraphHash(manifest, parents, occurrences, files)
	if err != nil {
		return BuildResult{}, err
	}
	manifest.LogicalGraphSHA256 = logical
	metadata, err := graphMetadata(manifest)
	if err != nil {
		return BuildResult{}, err
	}
	target := filepath.Join(request.EvaluationRoot, request.RunID)
	if _, err := os.Lstat(target); err == nil {
		return BuildResult{}, fmt.Errorf("relation graph artifact already exists")
	} else if !os.IsNotExist(err) {
		return BuildResult{}, err
	}
	temporary, err := os.MkdirTemp(request.EvaluationRoot, ".relation-graph-")
	if err != nil {
		return BuildResult{}, err
	}
	defer os.RemoveAll(temporary)
	databasePath := filepath.Join(temporary, "relations.db")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return BuildResult{}, err
	}
	if err := createSchema(ctx, db); err == nil {
		err = insertGraph(ctx, db, metadata, parents, occurrences, files)
	}
	if err == nil {
		_, err = db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	}
	if err == nil {
		err = graphIntegrity(ctx, db)
	}
	if closeErr := db.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return BuildResult{}, err
	}
	digest, err := fileSHA256(databasePath)
	if err != nil {
		return BuildResult{}, err
	}
	manifest.DatabaseSHA256 = digest
	if err := writePortableJSON(filepath.Join(temporary, "graph-manifest.json"), manifest, request.SourceRoot); err != nil {
		return BuildResult{}, err
	}
	if err := writePortableJSON(filepath.Join(temporary, "resolution-summary.json"), resolutionSummary(occurrences, files), request.SourceRoot); err != nil {
		return BuildResult{}, err
	}
	if err := reproveGraph(ctx, databasePath, manifest); err != nil {
		return BuildResult{}, err
	}
	if err := writeChecksums(temporary); err != nil {
		return BuildResult{}, err
	}
	// The last action before publication is a fresh proof of every input that
	// can affect the graph.  The SQLite sidecar itself has already been closed
	// and reproved above; this closes the verify/use interval for the source,
	// snapshots, resolver helper, and local compiler toolchains.
	if err := reproveLive(ctx, request, parentHash, fileHash, manifest); err != nil {
		return BuildResult{}, err
	}
	if err := os.Rename(temporary, target); err != nil {
		return BuildResult{}, err
	}
	return BuildResult{RunID: request.RunID, Reference: filepath.ToSlash(filepath.Join("evaluations", request.RunID)), DatabaseSHA256: digest, Occurrences: len(occurrences)}, nil
}

func validateBuildRequest(request BuildRequest) error {
	if !strings.HasPrefix(request.RunID, "relation-graph-") || !validRelative(request.RunID) || request.EvaluationRoot == "" || request.SourceRoot == "" || request.RepositoryRoot == "" || request.CorpusManifestFingerprint == "" || !request.Corpus.Clean || request.Parents.Generation < 0 || request.Parents.ManifestSHA256 == "" || request.Reproof == nil {
		return fmt.Errorf("invalid relation graph build request")
	}
	for _, root := range []string{request.EvaluationRoot, request.SourceRoot, request.RepositoryRoot} {
		info, err := os.Lstat(root)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe relation graph root")
		}
	}
	return nil
}

func extractAll(ctx context.Context, root string, parents []Parent, indexed map[string]store.IndexedFile) ([]Candidate, []Occurrence, []FileResolution, error) {
	files := map[string]string{}
	for file := range indexed {
		if language := languageForPath(file); language != "" {
			files[file] = language
		}
	}
	var all []Candidate
	var pre []Occurrence
	var states []FileResolution
	for _, file := range sortedKeys(files) {
		source, err := readSourceFile(root, file)
		if err != nil {
			return nil, nil, nil, err
		}
		candidates, unresolvedOccurrences, err := ExtractCandidates(ctx, file, files[file], source, parents)
		if err != nil {
			return nil, nil, nil, err
		}
		all, pre = append(all, candidates...), append(pre, unresolvedOccurrences...)
		states = append(states, FileResolution{Path: file, Language: files[file], Outcome: "EXTRACTED"})
	}
	return all, pre, states, nil
}

func typeScriptUniverse(indexed map[string]store.IndexedFile) []TSFile {
	seen := map[string]TSFile{}
	for file, entry := range indexed {
		language := languageForPath(file)
		if language != "typescript" && language != "tsx" {
			continue
		}
		seen[file] = TSFile{Path: file, Language: language, IndexedSHA256: entry.SHA256}
	}
	files := make([]TSFile, 0, len(seen))
	for _, file := range seen {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func verifyIndexedFiles(root string, files map[string]store.IndexedFile) error {
	for file, entry := range files {
		content, err := readSourceFile(root, file)
		if err != nil {
			return err
		}
		if sha256Hex(content) != entry.SHA256 {
			return fmt.Errorf("indexed file source reproof failed")
		}
	}
	return nil
}
func verifyParentBodies(root string, parents []Parent) error {
	for _, parent := range parents {
		content, err := readSourceFile(root, parent.Path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != parent.IndexedSHA256 || parent.EndByte > len(content) || string(content[parent.StartByte:parent.EndByte]) != parent.SourceBody {
			return fmt.Errorf("semantic parent source reproof failed")
		}
	}
	return nil
}

func readSourceFile(root, relative string) ([]byte, error) {
	if !validRelative(relative) {
		return nil, fmt.Errorf("unsafe indexed source path")
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	canonicalFile, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return nil, err
	}
	delta, err := filepath.Rel(canonicalRoot, canonicalFile)
	if err != nil || delta == ".." || strings.HasPrefix(filepath.ToSlash(delta), "../") {
		return nil, fmt.Errorf("indexed source path escapes root")
	}
	return os.ReadFile(canonicalFile)
}

func languageForPath(file string) string {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	default:
		return ""
	}
}

func validateOccurrenceSet(values []Occurrence) error {
	seen := map[string]bool{}
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if seen[value.ID] {
			return fmt.Errorf("duplicate occurrence")
		}
		seen[value.ID] = true
	}
	return nil
}

func mergeFileResolutions(groups ...[]FileResolution) []FileResolution {
	seen := map[string]FileResolution{}
	for _, group := range groups {
		for _, item := range group {
			seen[item.Path] = item
		}
	}
	result := make([]FileResolution, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func graphMetadata(manifest GraphManifest) (map[string]string, error) {
	canonical, err := config.CanonicalJSON(manifest)
	if err != nil {
		return nil, err
	}
	return map[string]string{"schema_version": fmt.Sprintf("%d", SchemaVersion), "protocol": ProtocolVersion, "manifest_sha256": sha256Hex(canonical), "parent_inventory_sha256": manifest.SemanticParentInventorySHA256, "indexed_file_inventory_sha256": manifest.IndexedFileInventorySHA256, "logical_graph_sha256": manifest.LogicalGraphSHA256}, nil
}

func logicalGraphHash(manifest GraphManifest, parents []Parent, occurrences []Occurrence, files []FileResolution) (string, error) {
	bound := struct {
		Schema          int                 `json:"schema"`
		Corpus          eval.VerifiedCorpus `json:"corpus"`
		CorpusManifest  string              `json:"corpus_manifest"`
		Generation      int64               `json:"generation"`
		IndexManifest   string              `json:"index_manifest"`
		ParentInventory string              `json:"parent_inventory"`
		FileInventory   string              `json:"file_inventory"`
		Profiles        map[string]string   `json:"profiles"`
		Resolver        map[string]string   `json:"resolver"`
		Implementation  buildinfo.Info      `json:"implementation"`
		Parents         []Parent            `json:"parents"`
		Occurrences     []Occurrence        `json:"occurrences"`
		Files           []FileResolution    `json:"files"`
	}{Schema: SchemaVersion, Corpus: manifest.Corpus, CorpusManifest: manifest.CorpusManifestFingerprint, Generation: manifest.IndexGeneration, IndexManifest: manifest.IndexManifestSHA256, ParentInventory: manifest.SemanticParentInventorySHA256, FileInventory: manifest.IndexedFileInventorySHA256, Profiles: manifest.Profiles, Resolver: manifest.ResolverPolicy, Implementation: manifest.Implementation, Parents: parents, Occurrences: occurrences, Files: files}
	sort.Slice(bound.Occurrences, func(i, j int) bool { return bound.Occurrences[i].ID < bound.Occurrences[j].ID })
	sort.Slice(bound.Files, func(i, j int) bool { return bound.Files[i].Path < bound.Files[j].Path })
	return canonicalHash(bound)
}

func bindToolchains(manifest *GraphManifest, request BuildRequest) error {
	if manifest == nil {
		return fmt.Errorf("graph manifest required")
	}
	manifest.ResolverPolicy["go_packages_pattern"] = "./...;tests=true"
	manifest.ResolverPolicy["x_tools_version"] = "v0.49.0"
	manifest.ResolverPolicy["go_version"] = manifest.Implementation.GoVersion
	manifest.ResolverPolicy["target_os"] = manifest.Implementation.TargetOS
	manifest.ResolverPolicy["target_arch"] = manifest.Implementation.TargetArch
	goBinding, err := goToolchainBinding(request.Go)
	if err != nil {
		return err
	}
	for key, value := range goBinding {
		manifest.ResolverPolicy[key] = value
	}
	executable := request.Executable
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve build executable: %w", err)
		}
	}
	resolvedExecutable, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("resolve build executable: %w", err)
	}
	executableHash, err := fileSHA256(resolvedExecutable)
	if err != nil {
		return fmt.Errorf("hash build executable: %w", err)
	}
	manifest.ResolverPolicy["build_executable_sha256"] = executableHash
	if len(typeScriptUniverse(request.Index.Files)) == 0 {
		manifest.ResolverPolicy["typescript"] = "not_applicable"
		return nil
	}
	for name, file := range map[string]string{"typescript_helper_sha256": request.TypeScriptHelper, "typescript_lock_sha256": request.TypeScriptLock, "tsconfig_sha256": request.TSConfig} {
		if file == "" {
			return fmt.Errorf("%s is required", name)
		}
		digest, err := fileSHA256(file)
		if err != nil {
			return err
		}
		manifest.ResolverPolicy[name] = digest
	}
	resolvedNode, err := filepath.EvalSymlinks(request.Node)
	if err != nil {
		return fmt.Errorf("resolve Node executable: %w", err)
	}
	manifest.ResolverPolicy["node_executable"] = filepath.Base(resolvedNode)
	nodeHash, err := fileSHA256(resolvedNode)
	if err != nil {
		return err
	}
	nodeVersion, err := exec.Command(resolvedNode, "--version").Output()
	if err != nil {
		return fmt.Errorf("read Node version: %w", err)
	}
	manifest.ResolverPolicy["node_executable_sha256"] = nodeHash
	manifest.ResolverPolicy["node_version"] = strings.TrimSpace(string(nodeVersion))
	payloadHash, err := directoryPayloadHash(filepath.Join(request.TypeScriptRoot, "node_modules", "typescript"))
	if err != nil {
		return fmt.Errorf("hash TypeScript runtime payload: %w", err)
	}
	manifest.ResolverPolicy["typescript_runtime_payload_sha256"] = payloadHash
	tsUniverseHash, err := canonicalHash(typeScriptUniverse(request.Index.Files))
	if err != nil {
		return err
	}
	manifest.ResolverPolicy["typescript_indexed_universe_sha256"] = tsUniverseHash
	return nil
}

func goToolchainBinding(command string) (map[string]string, error) {
	if command == "" {
		command = "go"
	}
	resolved, err := resolveGoExecutable(command)
	if err != nil {
		return nil, err
	}
	digest, err := fileSHA256(resolved)
	if err != nil {
		return nil, fmt.Errorf("hash Go executable: %w", err)
	}
	commandEnv := exec.Command(resolved, "env", "GOVERSION", "GOOS", "GOARCH", "GOFLAGS", "GOTOOLCHAIN")
	commandEnv.Env = controlledGoEnvironment(resolved)
	output, err := commandEnv.Output()
	if err != nil {
		return nil, fmt.Errorf("read controlled Go environment: %w", err)
	}
	parts := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(parts) != 5 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[4] == "" {
		return nil, fmt.Errorf("invalid controlled Go environment")
	}
	return map[string]string{
		"go_executable":             filepath.Base(resolved),
		"go_executable_sha256":      digest,
		"go_env_goversion":          parts[0],
		"go_env_goos":               parts[1],
		"go_env_goarch":             parts[2],
		"go_env_goflags":            parts[3],
		"go_env_gotoolchain":        parts[4],
		"go_packages_command":       "PATH-first:go",
		"go_packages_command_limit": "requires-resolved-basename-go",
	}, nil
}

func resolveGoExecutable(command string) (string, error) {
	if command == "" {
		command = "go"
	}
	located, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("locate Go executable: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(located)
	if err != nil {
		return "", fmt.Errorf("resolve Go executable: %w", err)
	}
	if filepath.Base(resolved) != "go" {
		return "", fmt.Errorf("go/packages requires an executable named go")
	}
	return resolved, nil
}

func controlledGoEnvironment(goExecutable string) []string {
	values := make([]string, 0, len(os.Environ())+3)
	pathValue := ""
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "GOFLAGS=") || strings.HasPrefix(value, "GOTOOLCHAIN=") {
			continue
		}
		if strings.HasPrefix(value, "PATH=") {
			pathValue = strings.TrimPrefix(value, "PATH=")
			continue
		}
		values = append(values, value)
	}
	values = append(values, "PATH="+filepath.Dir(goExecutable)+string(os.PathListSeparator)+pathValue, "GOFLAGS=", "GOTOOLCHAIN=local")
	return values
}

func directoryPayloadHash(root string) (string, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("unsafe runtime payload")
	}
	values := map[string]string{}
	err = filepath.WalkDir(root, func(file string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("runtime payload contains symlink")
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("runtime payload contains non-regular file")
		}
		relative, err := filepath.Rel(root, file)
		if err != nil || !validRelative(filepath.ToSlash(relative)) {
			return fmt.Errorf("unsafe runtime payload path")
		}
		digest, err := fileSHA256(file)
		if err != nil {
			return err
		}
		values[filepath.ToSlash(relative)] = digest
		return nil
	})
	if err != nil {
		return "", err
	}
	return canonicalHash(values)
}

func reproveLive(ctx context.Context, request BuildRequest, expectedParents, expectedFiles string, expectedManifest GraphManifest) error {
	index, parents, err := request.Reproof(ctx)
	if err != nil {
		return err
	}
	if index.Applied.ActiveGeneration != request.Index.Applied.ActiveGeneration || index.Applied.ManifestSHA256 != request.Index.Applied.ManifestSHA256 || parents.Generation != request.Parents.Generation || parents.ManifestSHA256 != request.Parents.ManifestSHA256 {
		return fmt.Errorf("BASE_GENERATION_CHANGED")
	}
	parentValues, err := ParentInventory(parents.Parents)
	if err != nil {
		return err
	}
	parentHash, err := inventoryHash(parentValues)
	if err != nil {
		return err
	}
	fileHash, err := indexedFileHash(index.Files)
	if err != nil {
		return err
	}
	if parentHash != expectedParents || fileHash != expectedFiles {
		return fmt.Errorf("BASE_GENERATION_CHANGED")
	}
	if err := verifyIndexedFiles(request.SourceRoot, index.Files); err != nil {
		return fmt.Errorf("BASE_INPUT_CHANGED: %w", err)
	}
	if err := verifyParentBodies(request.SourceRoot, parentValues); err != nil {
		return fmt.Errorf("BASE_INPUT_CHANGED: %w", err)
	}
	current := GraphManifest{Implementation: buildinfo.Current(), ResolverPolicy: map[string]string{"protocol": ProtocolVersion, "identity": IdentityPolicyID, "metadata": MetadataPolicyID, "go": "go/packages-go/types-v1", "typescript": "typescript-6.0.3-program-typechecker-v1", "typescript_version": TypeScriptVersion, "schema": "relation-sidecar-v3"}}
	if err := bindToolchains(&current, request); err != nil {
		return fmt.Errorf("resolver input reproof: %w", err)
	}
	if !reflect.DeepEqual(current.Implementation, expectedManifest.Implementation) || !reflect.DeepEqual(current.ResolverPolicy, expectedManifest.ResolverPolicy) {
		return fmt.Errorf("RESOLVER_OR_BUILD_INPUT_CHANGED")
	}
	return nil
}

func inventoryHash(parents []Parent) (string, error) { return canonicalHash(parents) }
func indexedFileHash(files map[string]store.IndexedFile) (string, error) {
	keys := sortedKeys(files)
	values := make([]store.IndexedFile, 0, len(keys))
	for _, key := range keys {
		values = append(values, files[key])
	}
	return canonicalHash(values)
}
func canonicalHash(value any) (string, error) {
	data, err := config.CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}
func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func fileSHA256(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}

func resolutionSummary(occurrences []Occurrence, files []FileResolution) map[string]any {
	byOutcome := map[string]int{}
	byKind := map[string]int{}
	byFileOutcome := map[string]int{}
	for _, occurrence := range occurrences {
		byOutcome[string(occurrence.Outcome)]++
		byKind[string(occurrence.Kind)]++
	}
	for _, file := range files {
		byFileOutcome[file.Outcome]++
	}
	return map[string]any{"schema_version": SchemaVersion, "occurrence_count": len(occurrences), "file_count": len(files), "by_outcome": byOutcome, "by_file_outcome": byFileOutcome, "by_relation_kind": byKind}
}

func writePortableJSON(file string, value any, forbidden string) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if (forbidden != "" && bytesContain(data, []byte(forbidden))) || !portableNoAbsolute(data) {
		return fmt.Errorf("portable relation artifact contains local path")
	}
	return os.WriteFile(file, data, 0o600)
}
func bytesContain(a, b []byte) bool { return len(b) > 0 && strings.Contains(string(a), string(b)) }

func reproveGraph(ctx context.Context, file string, manifest GraphManifest) error {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(file)+"?mode=ro&immutable=1")
	if err != nil {
		return err
	}
	defer db.Close()
	if err := graphIntegrity(ctx, db); err != nil {
		return err
	}
	var parents, traits, occurrences, files int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM semantic_parents`).Scan(&parents); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM parent_traits`).Scan(&traits); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM relation_occurrences`).Scan(&occurrences); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM file_resolution`).Scan(&files); err != nil {
		return err
	}
	if parents != manifest.Counts["semantic_parents"] || traits != manifest.Counts["parent_traits"] || occurrences != manifest.Counts["relation_occurrences"] || files != manifest.Counts["file_resolution"] {
		return fmt.Errorf("relation graph reproof count mismatch")
	}
	var logical string
	if err := db.QueryRowContext(ctx, `SELECT value FROM graph_meta WHERE key='logical_graph_sha256'`).Scan(&logical); err != nil || logical != manifest.LogicalGraphSHA256 {
		return fmt.Errorf("relation graph logical reproof mismatch")
	}
	return nil
}
