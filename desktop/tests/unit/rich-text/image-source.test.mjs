import { test } from "node:test";
import assert from "node:assert/strict";
import {
    imageURL,
    generatedImageURL,
} from "../../../renderer/src/rich-text/imageSource.ts";

test("generated image references preserve the artifact ID without inspection", () => {
    const id = "7005ac0117b3ca902422b83cd3db8f05";
    assert.equal(imageURL(`arx-media:${id}`), `arx-media:${id}`);
    assert.equal(generatedImageURL(`arx-media:${id}`), `arx-media://artifact/${id}`);
    assert.equal(imageURL(`arx-image:${id}`), "");
});

test("image references reject paths and arbitrary protocols", () => {
    for (const source of [
        "arx-media:../../secret",
        "arx-media://other/file",
        "javascript:alert(1)",
        "file:///private/file.png",
        "https://user:pass@example.com/a.png",
    ]) {
        assert.equal(imageURL(source), "");
        assert.equal(generatedImageURL(source), "");
    }
    assert.equal(imageURL("arx-image:" + "a".repeat(64)), "arx-image:" + "a".repeat(64));
});
