package datamigration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/leothevan2444/moji/internal/subscription"
)

// Legacy payloads have no independent version column. Refuse unknown fields
// instead of silently dropping a newer producer's persisted business state.
func validateLegacyPayloads(ctx context.Context, source *sqlx.DB) error {
	var reviews []struct {
		Key string `db:"release_key"`
		Raw string `db:"record_json"`
	}
	if err := source.SelectContext(ctx, &reviews, "SELECT release_key,record_json FROM subscription_release_reviews"); err != nil {
		return err
	}
	for _, row := range reviews {
		var review subscription.ReleaseReview
		decoder := json.NewDecoder(strings.NewReader(row.Raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&review); err != nil {
			return fmt.Errorf("migration: unsupported legacy review payload %q: %w", row.Key, err)
		}
		if review.Key != row.Key {
			return errors.New("migration: legacy review key disagrees with payload")
		}
	}
	var runs []struct{ ID, Payload string }
	if err := source.SelectContext(ctx, &runs, "SELECT id,payload FROM subscription_check_runs"); err != nil {
		return err
	}
	for _, row := range runs {
		var run subscription.CheckRun
		decoder := json.NewDecoder(strings.NewReader(row.Payload))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&run); err != nil {
			return fmt.Errorf("migration: unsupported legacy check payload %q: %w", row.ID, err)
		}
		if run.ID != row.ID {
			return errors.New("migration: legacy check ID disagrees with payload")
		}
	}
	return nil
}
