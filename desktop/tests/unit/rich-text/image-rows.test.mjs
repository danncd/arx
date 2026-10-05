import { test } from "node:test";
import assert from "node:assert/strict";
import { imageRows } from "../../../renderer/src/rich-text/imageRows.ts";

const element = (tagName, children = []) => ({
    type: "element",
    tagName,
    properties: {},
    children,
});
const image = () => element("img");
const text = (value) => ({ type: "text", value });

test("groups adjacent images across paragraphs and line breaks", () => {
    const root = {
        type: "root",
        children: [
            element("p", [image(), element("br"), image()]),
            text("\n"),
            element("p", [element("a", [image()])]),
        ],
    };
    imageRows()(root);
    assert.equal(root.children.length, 1);
    assert.deepEqual(root.children[0].properties.className, ["image-row"]);
    assert.equal(root.children[0].children.length, 3);
});

test("keeps prose between separate image rows", () => {
    const prose = element("p", [text("Explanation")]);
    const root = {
        type: "root",
        children: [element("p", [image()]), prose, element("p", [image()])],
    };
    imageRows()(root);
    assert.equal(root.children.length, 3);
    assert.deepEqual(root.children[1], prose);
});

test("groups inline images without dropping surrounding text", () => {
    const root = {
        type: "root",
        children: [
            element("p", [text("Before "), image(), text(" "), image(), text(" after")]),
        ],
    };
    imageRows()(root);
    assert.equal(root.children[0].children[0].value, "Before ");
    assert.equal(root.children[0].children[1].children.length, 2);
    assert.equal(root.children[0].children[2].value, " after");
});
