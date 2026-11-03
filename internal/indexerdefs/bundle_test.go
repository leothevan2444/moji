package indexerdefs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const fixtureDefinition = `id: example
name: Example
description: Synthetic fixture
language: en-US
type: public
encoding: UTF-8
links: [https://example.invalid/]
caps:
  categories: {Movies: Movies}
  modes: {search: [q]}
search:
  paths: [{path: /}]
  rows: {selector: article}
  fields:
    title: {selector: h2}
    category: {text: Movies}
    size: {text: 1GB}
    seeders: {text: 1}
    download: {selector: a, attribute: href}
`

type archiveEntry struct {
	name    string
	content string
	kind    byte
	link    string
}

func makeArchive(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	writer := tar.NewWriter(gz)
	for _, entry := range entries {
		kind := entry.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0600, Size: int64(len(entry.content)), Typeflag: kind, Linkname: entry.link}
		if kind != tar.TypeReg {
			header.Size = 0
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := writer.Write([]byte(entry.content)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func validArchive(t *testing.T, definition string) []byte {
	return makeArchive(t, archiveEntry{name: "Indexers-fixed/definitions/v11/schema.json", content: "{}"}, archiveEntry{name: "Indexers-fixed/definitions/v11/example.yml", content: definition})
}

func TestStagePublishReloadAndImmutableRevision(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	revision := strings.Repeat("a", 40)
	stage, err := StageArchive(ctx, root, revision, bytes.NewReader(validArchive(t, fixtureDefinition)), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, revision)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("stage published prematurely")
	}
	bundle, err := Publish(ctx, root, stage)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBundle(ctx, bundle.Directory, bundle.ManifestSHA256)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Manifest.DefinitionCount != 1 || loaded.Plans["example"].Definition.Name != "Example" {
		t.Fatal("wrong loaded definition")
	}
	other, err := StageArchive(ctx, root, revision, bytes.NewReader(validArchive(t, strings.Replace(fixtureDefinition, "name: Example", "name: Changed", 1))), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(other.Directory)
	if _, err := Publish(ctx, root, other); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("immutable revision overwritten: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle.Directory, "example.yml"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBundle(ctx, bundle.Directory, bundle.ManifestSHA256); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("corrupt contents accepted: %v", err)
	}
}

func TestStageRejectsUnsafeAndIncompatibleArchives(t *testing.T) {
	cases := map[string][]byte{
		"traversal":        makeArchive(t, archiveEntry{name: "../escape", content: "bad"}),
		"symlink":          makeArchive(t, archiveEntry{name: "Indexers/definitions/v11/example.yml", kind: tar.TypeSymlink, link: "/etc/passwd"}),
		"duplicate":        makeArchive(t, archiveEntry{name: "Indexers/definitions/v11/example.yml", content: fixtureDefinition}, archiveEntry{name: "Indexers/definitions/v11/example.yml", content: fixtureDefinition}),
		"incompatible":     validArchive(t, strings.Replace(fixtureDefinition, "{selector: h2}", "{selector: h2, filters: [{name: execute}]}", 1)),
		"unknown encoding": validArchive(t, strings.Replace(fixtureDefinition, "encoding: UTF-8", "encoding: moji-unknown-codepage", 1)),
		"identity":         validArchive(t, strings.Replace(fixtureDefinition, "id: example", "id: bad/id", 1)),
		"empty":            makeArchive(t, archiveEntry{name: "Indexers/definitions/v12/example.yml", content: fixtureDefinition}),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := StageArchive(context.Background(), root, strings.Repeat("b", 40), bytes.NewReader(data), time.Now()); err == nil {
				t.Fatal("invalid archive accepted")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatal("failed stage left artifacts", err)
			}
		})
	}
}

func TestStageFixedCorpusWhenExplicitlyProvided(t *testing.T) {
	archive := os.Getenv("MOJI_INDEXER_CORPUS_ARCHIVE")
	if archive == "" {
		t.Skip("set MOJI_INDEXER_CORPUS_ARCHIVE to verify a local pinned archive")
	}
	f, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	root := t.TempDir()
	bundle, err := StageArchive(context.Background(), root, "5f5db4e58d06be18619c3a858b9a91ade6f341ca", f, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.DefinitionCount != 579 {
		t.Fatalf("unexpected corpus count %d", bundle.Manifest.DefinitionCount)
	}
	bundle, err = Publish(context.Background(), root, bundle)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadBundle(context.Background(), bundle.Directory, bundle.ManifestSHA256)
	if err != nil || len(reloaded.Plans) != 579 {
		t.Fatalf("full bundle reload failed: %v", err)
	}
}

func TestStageRejectsExhaustedStorageBudgetAndTruncatedGzip(t *testing.T) {
	root := t.TempDir()
	data := validArchive(t, fixtureDefinition)
	if _, err := stageArchive(context.Background(), root, strings.Repeat("a", 40), bytes.NewReader(data), time.Now(), 1); !errors.Is(err, ErrBudget) {
		t.Fatalf("storage budget ignored: %v", err)
	}
	data[len(data)-1] ^= 0xff
	if _, err := StageArchive(context.Background(), root, strings.Repeat("a", 40), bytes.NewReader(data), time.Now()); !errors.Is(err, ErrInvalidBundle) {
		t.Fatalf("gzip trailer not verified: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatal("failed stage artifacts retained", err)
	}
}
