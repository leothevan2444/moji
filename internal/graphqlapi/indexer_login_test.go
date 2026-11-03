package graphqlapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerlogin"
)

type fakeIndexerLogin struct {
	input      indexerlogin.Input
	id, answer string
	cancels    int
	image      []byte
}

func (f *fakeIndexerLogin) Begin(_ context.Context, in indexerlogin.Input) (indexerlogin.Report, error) {
	f.input = in
	return indexerlogin.Report{ID: "public-flow", InstanceID: in.InstanceID, ConfigRevision: in.ExpectedRevision, RuleRevision: "fixed-rules", State: "WAITING", ExpiresAt: time.Unix(123, 0), Challenge: &indexer.Challenge{ID: "private-session-challenge", Type: "text", Prompt: "Synthetic question", Image: f.image}}, nil
}
func (f *fakeIndexerLogin) Answer(_ context.Context, id string, in indexerlogin.Input, answer string) (indexerlogin.Report, error) {
	f.input = in
	f.id = id
	f.answer = answer
	return indexerlogin.Report{ID: id, InstanceID: in.InstanceID, ConfigRevision: in.ExpectedRevision, RuleRevision: "fixed-rules", State: "READY", ExpiresAt: time.Unix(123, 0)}, nil
}
func (f *fakeIndexerLogin) Cancel(_ context.Context, id string, in indexerlogin.Input) error {
	f.input = in
	f.id = id
	f.cancels++
	return nil
}
func TestIndexerLoginGraphQLUsesScopedCapabilityAndDoesNotReturnSessionIdentityOrAnswer(t *testing.T) {
	service := &fakeIndexerLogin{}
	r := NewResolver(nil, nil, nil, "test")
	r.IndexerLogins = service
	var reply struct {
		Errors []any
		Data   struct {
			Begin  *model.IndexerLoginReport `json:"beginIndexerLogin"`
			Answer *model.IndexerLoginReport `json:"answerIndexerChallenge"`
			Cancel bool                      `json:"cancelIndexerLogin"`
		}
	}
	executeDefinitionGraphQLInto(t, r, `mutation{beginIndexerLogin(input:{instanceId:"saved-site",expectedRevision:"configuration"}){id instanceId configRevision ruleRevision state expiresAt challenge{id type prompt imageData}}}`, &reply)
	if len(reply.Errors) != 0 || reply.Data.Begin == nil || reply.Data.Begin.State != "WAITING" || reply.Data.Begin.Challenge == nil || reply.Data.Begin.Challenge.ID != "public-flow" || reply.Data.Begin.Challenge.ImageData != nil || service.input.InstanceID != "saved-site" || service.input.ExpectedRevision != "configuration" {
		t.Fatal(reply)
	}
	data, _ := json.Marshal(reply)
	if strings.Contains(string(data), "private-session-challenge") {
		t.Fatal("raw session capability exposed")
	}
	executeDefinitionGraphQLInto(t, r, `mutation{answerIndexerChallenge(id:"public-flow",input:{instanceId:"saved-site",expectedRevision:"configuration"},answer:"private-answer"){id state challenge{id prompt}}}`, &reply)
	if len(reply.Errors) != 0 || reply.Data.Answer == nil || reply.Data.Answer.State != "READY" || reply.Data.Answer.Challenge != nil || service.id != "public-flow" || service.answer != "private-answer" {
		t.Fatal(reply)
	}
	data, _ = json.Marshal(reply)
	if strings.Contains(string(data), "private-answer") {
		t.Fatal("answer echoed")
	}
	executeDefinitionGraphQLInto(t, r, `mutation{cancelIndexerLogin(id:"public-flow",input:{instanceId:"saved-site",expectedRevision:"configuration"})}`, &reply)
	if len(reply.Errors) != 0 || !reply.Data.Cancel || service.cancels != 1 || service.id != "public-flow" {
		t.Fatal(reply)
	}
}
func TestIndexerLoginGraphQLImageProjectionAndRejectedImageCleanup(t *testing.T) {
	for _, scenario := range []struct {
		name  string
		image []byte
		ok    bool
	}{{"png", []byte("\x89PNG\r\n\x1a\nsynthetic"), true}, {"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), false}} {
		t.Run(scenario.name, func(t *testing.T) {
			service := &fakeIndexerLogin{image: scenario.image}
			r := NewResolver(nil, nil, nil, "test")
			r.IndexerLogins = service
			var reply struct {
				Errors []any
				Data   struct {
					Report *model.IndexerLoginReport `json:"beginIndexerLogin"`
				}
			}
			executeDefinitionGraphQLInto(t, r, `mutation{beginIndexerLogin(input:{instanceId:"saved-site",expectedRevision:"configuration"}){id state challenge{id imageData}}}`, &reply)
			if scenario.ok {
				if len(reply.Errors) != 0 || reply.Data.Report == nil || reply.Data.Report.Challenge.ImageData == nil || !strings.HasPrefix(*reply.Data.Report.Challenge.ImageData, "data:image/png;base64,") || service.cancels != 0 {
					t.Fatal(reply)
				}
			} else if len(reply.Errors) == 0 || service.cancels != 1 {
				t.Fatal("unrenderable challenge retained resources", reply, service.cancels)
			}
		})
	}
}

func (f *fakeIndexerLogin) Pending(ctx context.Context, in indexerlogin.Input) (*indexerlogin.Report, error) {
	report, err := f.Begin(ctx, in)
	return &report, err
}

func TestIndexerLoginGraphQLRestoresPendingChallengeWithPublicCapability(t *testing.T) {
	service := &fakeIndexerLogin{}
	r := NewResolver(nil, nil, nil, "test")
	r.IndexerLogins = service
	var reply struct {
		Errors []any
		Data   struct {
			Report *model.IndexerLoginReport `json:"indexerLogin"`
		}
	}
	executeDefinitionGraphQLInto(t, r, `query{indexerLogin(input:{instanceId:"saved-site",expectedRevision:"configuration"}){id state challenge{id prompt}}}`, &reply)
	if len(reply.Errors) != 0 || reply.Data.Report == nil || reply.Data.Report.State != "WAITING" || reply.Data.Report.Challenge.ID != "public-flow" || service.input.InstanceID != "saved-site" || service.input.ExpectedRevision != "configuration" {
		t.Fatal(reply)
	}
}

type expiredIndexerLogin struct{ *fakeIndexerLogin }

func (expiredIndexerLogin) Cancel(context.Context, string, indexerlogin.Input) error {
	return indexer.ErrExpired
}
func TestIndexerLoginCancelAlreadyExpiredIsIdempotent(t *testing.T) {
	r := NewResolver(nil, nil, nil, "test")
	r.IndexerLogins = expiredIndexerLogin{&fakeIndexerLogin{}}
	var reply struct {
		Errors []any
		Data   struct {
			Cancel bool `json:"cancelIndexerLogin"`
		}
	}
	executeDefinitionGraphQLInto(t, r, `mutation{cancelIndexerLogin(id:"expired-flow",input:{instanceId:"saved-site",expectedRevision:"configuration"})}`, &reply)
	if len(reply.Errors) != 0 || !reply.Data.Cancel {
		t.Fatal("expired cancellation prevented a fresh login", reply)
	}
}
