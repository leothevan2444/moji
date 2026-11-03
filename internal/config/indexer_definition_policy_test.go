package config

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
)

func TestDefinitionPolicyPartialUpdatesKeepOtherConfigurationAndPersistExplicitFalse(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	proxy := "http://synthetic-user:synthetic-password@proxy.example:8080"
	if _, _, err := s.SaveIndexerNetworkProfile(ctx, s.Revision(), "private", true, IndexerNetworkProfilePatch{ProxyURL: &proxy}); err != nil {
		t.Fatal(err)
	}
	before := s.Config()
	hours, enabled := 48, true
	saved, ack, err := s.UpdateIndexerDefinitionPolicy(ctx, s.Revision(), IndexerDefinitionPolicyPatch{CheckIntervalHours: &hours, AutoUpdate: &enabled})
	if err != nil || ack != s.Revision() || saved.Indexers.Builtin.Definitions.CheckIntervalHours != 48 || !saved.Indexers.Builtin.Definitions.AutoUpdate {
		t.Fatal("policy acknowledgement did not match persisted write", err)
	}
	expected := before.Indexers.Clone()
	expected.Builtin.Definitions.CheckIntervalHours = hours
	expected.Builtin.Definitions.AutoUpdate = true
	if !reflect.DeepEqual(expected, saved.Indexers) || !reflect.DeepEqual(before.Connection, saved.Connection) {
		t.Fatal("policy write changed unrelated indexer/account values")
	}
	enabled = false
	_, ack, err = s.UpdateIndexerDefinitionPolicy(ctx, ack, IndexerDefinitionPolicyPatch{AutoUpdate: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(s.Path())
	if err != nil || reopened.Revision() != ack || reopened.Config().Indexers.Builtin.Definitions.AutoUpdate || reopened.Config().Indexers.Builtin.Definitions.CheckIntervalHours != 48 {
		t.Fatal("omission/false/reopen changed policy semantics", err)
	}
	if reopened.Config().Indexers.Builtin.Definitions.Source != "prowlarr" || reopened.Config().Indexers.Builtin.Definitions.SchemaVersion != 11 || reopened.Config().Indexers.Builtin.NetworkProfiles["private"].ProxyURL != proxy {
		t.Fatal("fixed source/schema or credentials changed")
	}
}
func TestDefinitionPolicyRejectsInvalidStaleAndCancelledEditsWithoutChangingFile(t *testing.T) {
	s := patchStore(t)
	ctx := context.Background()
	initial := s.Revision()
	hours := 12
	if _, _, err := s.UpdateIndexerDefinitionPolicy(ctx, initial, IndexerDefinitionPolicyPatch{CheckIntervalHours: &hours}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, hours := range []int{0, -1, 169} {
		if _, _, err = s.UpdateIndexerDefinitionPolicy(ctx, s.Revision(), IndexerDefinitionPolicyPatch{CheckIntervalHours: &hours}); !errors.Is(err, ErrInvalidSettings) {
			t.Fatal("invalid interval accepted", hours, err)
		}
	}
	enabled := true
	if _, _, err = s.UpdateIndexerDefinitionPolicy(ctx, initial, IndexerDefinitionPolicyPatch{AutoUpdate: &enabled}); !errors.Is(err, ErrConfigRevision) {
		t.Fatal("stale editor overwrote new policy", err)
	}
	if _, _, err = s.UpdateIndexerDefinitionPolicy(ctx, s.Revision(), IndexerDefinitionPolicyPatch{}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatal("empty patch acknowledged", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err = s.UpdateIndexerDefinitionPolicy(cancelled, s.Revision(), IndexerDefinitionPolicyPatch{AutoUpdate: &enabled}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	after, err := os.ReadFile(s.Path())
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("rejected policy edit changed durable configuration", err)
	}
}
func TestConcurrentDefinitionPolicyEditsShareOneRevisionFence(t *testing.T) {
	s := patchStore(t)
	revision := s.Revision()
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, hours := range []int{12, 48} {
		wg.Add(1)
		go func(hours int) {
			defer wg.Done()
			<-start
			_, _, err := s.UpdateIndexerDefinitionPolicy(context.Background(), revision, IndexerDefinitionPolicyPatch{CheckIntervalHours: &hours})
			errs <- err
		}(hours)
	}
	close(start)
	wg.Wait()
	close(errs)
	success, conflict := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConfigRevision) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("concurrent edits were not fenced", success, conflict)
	}
}
