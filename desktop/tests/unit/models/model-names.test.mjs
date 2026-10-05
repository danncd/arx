import { test } from "node:test";
import assert from "node:assert/strict";
import { modelDisplayNames } from "../../../renderer/src/features/models/picker/displayNames.ts";

test("local labels keep model versions and sizes without file suffixes", () => {
    const names = [
        "Qwen3-0.6B-Q8_0.gguf",
        "Llama-3.1-8B-Instruct-Q4_K_M-00001-of-00002.gguf",
        "Gemma-3-4B-it-BF16.gguf",
    ];
    assert.deepEqual(
        modelDisplayNames(names.map((name, i) => ({ id: `local:${i}`, name }))).map(
            (model) => model.name,
        ),
        ["Qwen3 0.6B", "Llama 3.1 8B Instruct", "Gemma 3 4B it"],
    );
});

test("cloud names stay unchanged and local quantizations remain distinguishable", () => {
    const models = [
        { id: "cloud", name: "DeepSeek-V4.1-Flash" },
        { id: "local:one", name: "Qwen3-0.6B-Q8_0.gguf" },
        { id: "local:two", name: "Qwen3-0.6B-Q4_K_M.gguf" },
    ];
    assert.deepEqual(
        modelDisplayNames(models).map((model) => model.name),
        ["DeepSeek-V4.1-Flash", "Qwen3 0.6B · Q8_0", "Qwen3 0.6B · Q4_K_M"],
    );
    assert.equal(models[1].name, "Qwen3-0.6B-Q8_0.gguf");
});
