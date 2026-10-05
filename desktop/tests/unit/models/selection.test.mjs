import { test } from "node:test";
import assert from "node:assert/strict";
import { selectRun } from "../../../renderer/src/features/models/picker/selection.ts";

const models = [
    {
        id: "deepseek-flash",
        thinking: {
            efforts: ["low", "high", "max"],
            defaultEffort: "high",
            canDisable: true,
        },
    },
];

test("replaces obsolete selections with the provider default", () => {
    assert.deepEqual(selectRun(models, { model: "deepseek", effort: "medium" }), {
        provider: "deepseek",
        model: "deepseek-flash",
        effort: "high",
    });
});

test("preserves a supported selection including disabled thinking", () => {
    for (const effort of ["low", "max", "none"]) {
        const run = { model: "deepseek-flash", effort };
        assert.deepEqual(selectRun(models, run), { ...run, provider: "deepseek" });
    }
});

test("does not invent thinking levels when capabilities are missing", () => {
    assert.deepEqual(selectRun([{ id: "other" }], { model: "other", effort: "high" }), {
        provider: "deepseek",
        model: "other",
        effort: "",
    });
});

test("keeps local selection isolated from cloud providers", () => {
    const local = {
        id: "local:one",
        provider: "local",
        thinking: { efforts: ["default"], defaultEffort: "default", canDisable: true },
    };
    assert.deepEqual(selectRun([...models, local], { model: local.id, effort: "high" }), {
        provider: "local",
        model: local.id,
        effort: "default",
    });
});
