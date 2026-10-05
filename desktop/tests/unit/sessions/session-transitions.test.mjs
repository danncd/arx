import { test } from "node:test";
import assert from "node:assert/strict";
import {
    applyPage,
    applyLive,
} from "../../../renderer/src/features/sessions/transitions.ts";

const item = { id: "chat", title: "Chat", updatedAt: "", status: "idle" };
const chunk = (text, at) => ({
    id: "reply",
    conversation: "chat",
    role: "assistant",
    text,
    at,
    offset: 0,
    reasoning_offset: 0,
    status: "done",
});
const live = (text, at) => ({
    id: "reply",
    role: "assistant",
    text,
    at,
    reasoning: "",
    tools: [],
});
test("late history cannot replace newer live text, and older live cannot replace newer history", () => {
    const before = "2026-10-02T00:00:00.100000001Z";
    const after = "2026-10-02T00:00:00.100000002Z";
    let state = applyPage(
        {},
        "chat",
        item,
        { chunks: [chunk("old page", before)], before: "", more: false },
        [live("new live", after)],
        false,
    );
    assert.equal(state.chat.session.messages[0].text, "new live");
    state = applyPage(
        state,
        "chat",
        item,
        { chunks: [chunk("new page", after)], before: "cursor", more: true },
        [live("old live", before)],
        false,
    );
    assert.equal(state.chat.session.messages[0].text, "new page");
    state = applyLive(state, "chat", item, [live("older event", before)], true);
    assert.equal(state.chat.session.messages[0].text, "new page");
    assert.equal(state.chat.before, "cursor");
    assert.equal(state.chat.more, true);
});
