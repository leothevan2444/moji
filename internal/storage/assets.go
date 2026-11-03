package storage

import "context"

func (d *Database) acquireAssets(ctx context.Context) error {
	if d.closed.Load() {
		return ErrClosed
	}
	select {
	case d.assetGate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			d.releaseAssets()
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (d *Database) releaseAssets() { <-d.assetGate }

// WithAssetMaintenance serializes removal/replacement of immutable referenced
// files with backups. The callback must not invoke Backup on this database.
// It does not hold a SQL transaction or block ordinary database writes.
func (d *Database) WithAssetMaintenance(ctx context.Context, work func() error) error {
	if err := d.acquireAssets(ctx); err != nil {
		return err
	}
	defer d.releaseAssets()
	return work()
}
