import { test } from "node:test";
import assert from "node:assert/strict";
import {
    modelAvailability,
    selectRun,
} from "../../../renderer/src/features/models/picker/selection.ts";
const stopped = { state: "stopped" };
test("network selection is ready without DeepSeek and restores legacy provider", () => {
    const model = { id: "network:server:model", name: "Network" };
    assert.equal(modelAvailability(model, false, stopped), "ready");
    assert.equal(selectRun([model], { model: model.id, effort: "" }).provider, "network");
    assert.equal(
        modelAvailability({ id: "deepseek-chat" }, false, stopped),
        "unavailable",
    );
    assert.equal(modelAvailability(undefined, true, stopped), "unavailable");
});
test("installed local selection remains sendable and loads when necessary", () => {
    const model = { id: "local:installed" };
    assert.equal(modelAvailability(model, false, stopped), "requiresLoad");
    assert.equal(
        modelAvailability(model, false, { state: "ready", model: model.id }),
        "ready",
    );
});
