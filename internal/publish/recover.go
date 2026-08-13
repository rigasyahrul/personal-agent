package publish

import "context"
import "github.com/rigasyahrul/personal-agent/internal/store"

func (m *Machine) RecoverAll(ctx context.Context) error {
	ops, err := (store.DirectStore{DB: m.DB}).Active(ctx)
	if err != nil {
		return err
	}
	for _, o := range ops {
		if err = m.resume(ctx, o); err != nil {
			return err
		}
	}
	return nil
}
