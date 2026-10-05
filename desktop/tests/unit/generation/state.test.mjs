import { test } from "node:test";
import assert from "node:assert/strict";
import {
    applyGeneration,
    mergeGeneration,
} from "../../../renderer/src/features/generation/state/merge.ts";
const library = (revision) => ({ revision, models: [], defaults: {} });
const job = (updated, state) => ({ id: "job", updated, state, toolCall: "tool" });
test("late snapshots and events preserve newer generation data", () => {
    const held = {
        library: library(5),
        catalog: [],
        jobs: [job("2026-10-04T18:00:01Z", "done")],
    };
    const next = mergeGeneration(held, {
        library: library(4),
        catalog: [{ id: "catalog" }],
        jobs: [job("2026-10-04T18:00:00Z", "running")],
    });
    assert.equal(next.library.revision, 5);
    assert.equal(next.jobs[0].state, "done");
    assert.equal(next.catalog[0].id, "catalog");
    const stale = applyGeneration(next, {
        library: library(3),
        job: job("2026-10-04T18:00:00Z", "running"),
    });
    assert.equal(stale.library.revision, 5);
    assert.equal(stale.jobs[0].state, "done");
    const update = applyGeneration(stale, {
        job: { ...job("2026-10-04T18:00:02Z", "interrupted"), id: "other" },
    });
    assert.equal(update.jobs.length, 2);
});
