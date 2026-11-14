package tracker

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/fit"
	"github.com/saifkazi/trove/backend/internal/llm"
)

// verdictBatch caps model calls per run (about 1–2¢ each).
const verdictBatch = 10

// Judge makes device verdicts for games that need one (or just
// the given game). An incomplete verdict is only redone once its missing
// evidence can be reached, so a source that's down doesn't cost a model
// call every few hours.
func (t *Tracker) Judge(ctx context.Context, id *uuid.UUID) error {
	if t.Fit == nil || t.LLM == nil {
		return nil
	}
	targets, err := t.Store.GamesNeedingVerdict(ctx, id, verdictBatch)
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err := t.judge(ctx, target); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			t.Log.Warn("verdict failed", "game", target.Title, "err", err)
		}
	}
	return nil
}

func (t *Tracker) judge(ctx context.Context, target dao.VerdictTarget) error {
	ev := t.Fit.Gather(ctx, target.Game())
	if target.HasIncomplete && !ev.Complete() {
		t.Log.Info("verdict evidence still missing", "game", target.Title, "missing", ev.Missing)
		return t.Store.TouchVerdict(ctx, target.ID)
	}
	complete := ev.Complete()
	v, model, err := fit.Decide(ctx, t.LLM, ev)
	if errors.Is(err, llm.ErrRefused) {
		// Asking again would only be refused again: save a placeholder,
		// marked complete so it's retried only when it expires.
		v, model, complete, err = fit.Verdict{Summary: "No verdict: the model declined to answer."}, llm.Model, true, nil
	}
	if err != nil {
		return err
	}
	t.Log.Info("verdict", "game", target.Title, "complete", complete, "summary", v.Summary)
	return t.Store.SaveVerdict(ctx, target.ID, dao.StoredVerdict{
		Verdict: v, Sources: ev.Sources, Model: model, Complete: complete,
	})
}
