package graphqlapi

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/leothevan2444/moji/internal/graphqlapi/model"
	"github.com/leothevan2444/moji/internal/indexer"
	"github.com/leothevan2444/moji/internal/indexerlogin"
	"net/http"
	"time"
)

type IndexerLoginService interface {
	Pending(context.Context, indexerlogin.Input) (*indexerlogin.Report, error)
	Begin(context.Context, indexerlogin.Input) (indexerlogin.Report, error)
	Answer(context.Context, string, indexerlogin.Input, string) (indexerlogin.Report, error)
	Cancel(context.Context, string, indexerlogin.Input) error
}

func loginScope(input model.IndexerLoginInput) indexerlogin.Input {
	return indexerlogin.Input{InstanceID: input.InstanceID, ExpectedRevision: input.ExpectedRevision}
}
func (r *Resolver) beginIndexerLogin(ctx context.Context, input model.IndexerLoginInput) (*model.IndexerLoginReport, error) {
	if r.IndexerLogins == nil {
		return nil, indexer.ErrNotConfigured
	}
	report, err := r.IndexerLogins.Begin(ctx, loginScope(input))
	if err != nil {
		return nil, err
	}
	return r.projectLoginReport(ctx, input, report)
}
func (r *Resolver) projectLoginReport(ctx context.Context, input model.IndexerLoginInput, report indexerlogin.Report) (*model.IndexerLoginReport, error) {
	result, err := publicLoginReport(report)
	if err != nil && report.State == "WAITING" {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		_ = r.IndexerLogins.Cancel(cleanup, report.ID, loginScope(input))
	}
	return result, err
}
func (r *Resolver) answerIndexerChallenge(ctx context.Context, id string, input model.IndexerLoginInput, answer string) (*model.IndexerLoginReport, error) {
	if r.IndexerLogins == nil {
		return nil, indexer.ErrNotConfigured
	}
	report, err := r.IndexerLogins.Answer(ctx, id, loginScope(input), answer)
	if err != nil {
		return nil, err
	}
	return publicLoginReport(report)
}
func (r *Resolver) cancelIndexerLogin(ctx context.Context, id string, input model.IndexerLoginInput) (bool, error) {
	if r.IndexerLogins == nil {
		return false, indexer.ErrNotConfigured
	}
	err := r.IndexerLogins.Cancel(ctx, id, loginScope(input))
	if errors.Is(err, indexer.ErrExpired) {
		return true, nil
	}
	return err == nil, err
}
func publicLoginReport(report indexerlogin.Report) (*model.IndexerLoginReport, error) {
	result := &model.IndexerLoginReport{ID: report.ID, InstanceID: report.InstanceID, ConfigRevision: report.ConfigRevision, RuleRevision: report.RuleRevision, State: model.IndexerLoginState(report.State), ExpiresAt: report.ExpiresAt.UTC().Format(time.RFC3339Nano)}
	if !result.State.IsValid() {
		return nil, indexer.ErrUnsupported
	}
	if report.State != "WAITING" {
		return result, nil
	}
	if report.Challenge == nil {
		return nil, indexer.ErrExpired
	}
	challenge := report.Challenge
	if len(challenge.Prompt) > 4096 || len(challenge.Image) > 8<<20 {
		return nil, indexer.ErrResponseBudget
	}
	// The public capability is the scoped login ID, never the session's raw ID.
	result.Challenge = &model.IndexerLoginChallenge{ID: report.ID, Type: challenge.Type, Prompt: challenge.Prompt}
	if len(challenge.Image) > 0 {
		mime := http.DetectContentType(challenge.Image)
		switch mime {
		case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp":
		default:
			return nil, indexer.ErrUnsupported
		}
		data := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(challenge.Image)
		result.Challenge.ImageData = &data
	}
	return result, nil
}

func (r *Resolver) pendingIndexerLogin(ctx context.Context, input model.IndexerLoginInput) (*model.IndexerLoginReport, error) {
	if r.IndexerLogins == nil {
		return nil, indexer.ErrNotConfigured
	}
	report, err := r.IndexerLogins.Pending(ctx, loginScope(input))
	if err != nil || report == nil {
		return nil, err
	}
	return r.projectLoginReport(ctx, input, *report)
}
