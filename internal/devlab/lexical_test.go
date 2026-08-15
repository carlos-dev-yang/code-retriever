package devlab

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cidx/internal/buildinfo"
	"cidx/internal/eval"
	"cidx/internal/evalcontract"
)

func TestDraftCaseDigestFraming(t *testing.T) {
	for _, name := range []string{"lexical-go-chi-v5.3.1-draft.json", "lexical-react-hook-form-v7.85.0-draft.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "retrieval", name))
		if err != nil {
			t.Fatal(err)
		}
		dataset, err := eval.LoadDataset(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range dataset.Cases {
			digest, err := DraftCaseDigest(item)
			if err != nil {
				t.Fatal(err)
			}
			if item.Digest != digest {
				t.Fatalf("%s digest=%s want=%s", item.ID, item.Digest, digest)
			}
		}
	}
}

func TestLexicalArtifactRootAndPacketImmutability(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, ".cidx", "lab", "evaluations")
	if got, err := lexicalArtifactRoot(root); err != nil || got != base {
		t.Fatalf("safe artifact root got=%q err=%v", got, err)
	}
	if err := prepareLexicalArtifactRoot(root); err != nil {
		t.Fatal(err)
	}
	symlinkRoot := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(symlinkRoot, ".cidx")); err != nil {
		t.Fatal(err)
	}
	if _, err := lexicalArtifactRoot(symlinkRoot); err == nil {
		t.Fatal("artifact-root symlink accepted")
	}
	corpus := eval.VerifiedCorpus{CorpusID: "sample", PinnedCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ContentSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Clean: true}
	inventory := eval.TruthInventorySnapshot{Generation: 1, ManifestSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Chunks: []eval.IndexedTruth{{Path: "pkg/file.go", IndexedSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", QualifiedSymbol: "pkg.F", Kind: "function", StartByte: 0, EndByte: 1}}}
	first, err := writeLexicalInventory(base, corpus, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", inventory)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := writeLexicalInventory(base, corpus, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", inventory); err != nil || second != first {
		t.Fatalf("idempotent inventory got=%+v err=%v", second, err)
	}
	inventory.Chunks[0].EndByte = 2
	if _, err := writeLexicalInventory(base, corpus, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", inventory); err == nil {
		t.Fatal("inventory collision accepted")
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "retrieval", "lexical-go-chi-v5.3.1-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := eval.LoadDataset(data)
	if err != nil {
		t.Fatal(err)
	}
	review, err := writeLexicalReviewPacket(base, corpus, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", dataset)
	if err != nil {
		t.Fatal(err)
	}
	packetData, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(review.Reference)))
	if err != nil {
		t.Fatal(err)
	}
	var packet struct {
		DatasetStatus        string   `json:"dataset_status"`
		LabelAuthority       string   `json:"label_authority"`
		MissingFloorCoverage []string `json:"missing_floor_coverage"`
		ReviewCases          []struct {
			ID         string `json:"id"`
			Text       string `json:"text"`
			Language   string `json:"language"`
			AnswerMode string `json:"answer_mode"`
		} `json:"review_cases"`
	}
	if err := json.Unmarshal(packetData, &packet); err != nil {
		t.Fatal(err)
	}
	if packet.DatasetStatus != "DRAFT" || packet.LabelAuthority != "MACHINE_PREPARED_UNREVIEWED" || len(packet.MissingFloorCoverage) == 0 || len(packet.ReviewCases) == 0 || packet.ReviewCases[0].ID == "" || packet.ReviewCases[0].Text == "" || packet.ReviewCases[0].Language == "" || packet.ReviewCases[0].AnswerMode == "" {
		t.Fatalf("malformed review packet: %+v", packet)
	}
}

func TestLexicalPacketChildSymlinksCannotEscapeArtifactRoot(t *testing.T) {
	corpus := eval.VerifiedCorpus{CorpusID: "sample", PinnedCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ContentSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Clean: true}
	inventory := eval.TruthInventorySnapshot{Generation: 1, ManifestSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Chunks: []eval.IndexedTruth{{Path: "pkg/file.go", IndexedSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", QualifiedSymbol: "pkg.F", Kind: "function", StartByte: 0, EndByte: 1}}}
	for _, child := range []string{"inventory", "review"} {
		t.Run(child, func(t *testing.T) {
			base := filepath.Join(t.TempDir(), ".cidx", "lab", "evaluations")
			if err := os.MkdirAll(base, 0o700); err != nil {
				t.Fatal(err)
			}
			outside := t.TempDir()
			if err := os.Symlink(outside, filepath.Join(base, child)); err != nil {
				t.Fatal(err)
			}
			var err error
			switch child {
			case "inventory":
				_, err = writeLexicalInventory(base, corpus, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", inventory)
			case "review":
				_, err = writeLexicalReviewPacket(base, corpus, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", eval.EvaluationDataset{})
			}
			if err == nil {
				t.Fatal("child symlink was accepted")
			}
			entries, err := os.ReadDir(outside)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("packet escaped artifact root: %v", entries)
			}
		})
	}
}

func TestLexicalModeRejectsApplyAndInventoryFlags(t *testing.T) {
	for _, args := range [][]string{
		{"retrieval", "evaluate", "--mode", "lexical", "--apply", "--corpus-manifest", "manifest", "--dataset", "dataset"},
		{"retrieval", "evaluate", "--mode", "retrieval", "--inventory-only", "--corpus-manifest", "manifest", "--dataset", "dataset"},
	} {
		if err := (CLI{}).Run(context.Background(), args, os.Stdout, os.Stderr); err == nil {
			t.Fatalf("unsafe flag combination accepted: %v", args)
		}
	}
}

func TestLexicalCodeProvenanceRequiresCleanKnownBuild(t *testing.T) {
	sha1 := "0123456789abcdef0123456789abcdef01234567"
	sha256 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	for _, commit := range []string{sha1, sha256} {
		if err := validateLexicalCodeProvenance(buildinfo.Info{Commit: commit, SourceModified: "false"}); err != nil {
			t.Fatalf("clean provenance rejected for %q: %v", commit, err)
		}
	}
	for _, info := range []buildinfo.Info{
		{Commit: "unknown", SourceModified: "false"},
		{Commit: "abc123", SourceModified: "false"},
		{Commit: "0123456789abcdef0123456789abcdef0123456g", SourceModified: "false"},
		{Commit: "0123456789ABCDEF0123456789abcdef01234567", SourceModified: "false"},
		{Commit: sha1, SourceModified: "true"},
		{Commit: sha1, SourceModified: "unknown"},
	} {
		if err := validateLexicalCodeProvenance(info); err == nil {
			t.Fatalf("unclean provenance accepted: %+v", info)
		}
	}
}

func TestLexicalSmokeRejectsFrozenDataset(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "retrieval", "lexical-go-chi-v5.3.1-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	dataset, err := eval.LoadDataset(data)
	if err != nil {
		t.Fatal(err)
	}
	dataset.Cases[0].Review.State = evalcontract.ReviewFrozen
	dataset.Cases[0].Review.Passes = []evalcontract.ReviewPass{{ID: "one", Reviewer: "reviewer-a"}, {ID: "two", Reviewer: "reviewer-b"}}
	if err := validateDraftCaseDigests(dataset); err == nil {
		t.Fatal("frozen dataset accepted by lexical smoke")
	}
}
