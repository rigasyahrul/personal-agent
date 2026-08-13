package publish

import (
	"context"
	"fmt"

	"github.com/rigasyahrul/personal-agent/internal/store"
)

func (m *Machine) RecoverAll(ctx context.Context) error {
	ops, err := (store.DirectStore{DB: m.DB}).Active(ctx)
	if err != nil {
		return err
	}
	for _, o := range ops {
		if err = m.recover(ctx, o); err != nil {
			current, lookupErr := (store.DirectStore{DB: m.DB}).ByID(ctx, o.ID)
			if lookupErr != nil {
				return lookupErr
			}
			if current.Status != "failed" {
				return err
			}
		}
	}
	promotes, err := (store.PromoteStore{DB: m.DB}).Active(ctx)
	if err != nil {
		return fmt.Errorf("list active promote publications: %w", err)
	}
	for _, o := range promotes {
		if err := m.recover(ctx, o); err != nil {
			return fmt.Errorf("recover promote publication %s: %w", o.ID, err)
		}
	}
	return nil
}

func (m *Machine) recover(ctx context.Context, o store.DirectOperation) error {
	if o.Kind == "promote" && (o.Status == "published_fs" || o.Status == "finalized" || o.Status == "review_enqueued") {
		state, err := m.inspectDestination(ctx, o)
		if err != nil {
			return err
		}
		if state != destinationMatches {
			return fmt.Errorf("published destination is missing, mismatched, or unsafe")
		}
	}
	return m.resume(ctx, o)
}
