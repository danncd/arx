import type { Session } from "./types";
import type { Message } from "../chat/types";
import type { HistoryPage } from "../../../../contracts/wire.generated";
import { messagesFromChunks } from "./history.ts";
import { mergeMessages } from "./merge.ts";

export type LoadedSession = { session: Session; before: string; more: boolean };
type Loaded = Record<string, LoadedSession>;
type Summary = Omit<Session, "messages" | "draft">;
export function applyLive(
    held: Loaded,
    id: string,
    item: Summary,
    live: Message[],
    known: boolean,
): Loaded {
    if (!held[id] && known) return held;
    const existing = held[id];
    return {
        ...held,
        [id]: {
            session: {
                ...item,
                draft: "",
                messages: mergeMessages(existing?.session.messages || [], live),
            },
            before: existing?.before || "",
            more: existing?.more || false,
        },
    };
}
export function applyPage(
    held: Loaded,
    id: string,
    item: Summary,
    page: HistoryPage,
    live: Message[],
    prepend: boolean,
): Loaded {
    const messages = messagesFromChunks(page.chunks);
    const history = prepend
        ? mergeMessages(messages, held[id]?.session.messages || [])
        : messages;
    return {
        ...held,
        [id]: {
            session: { ...item, draft: "", messages: mergeMessages(history, live) },
            before: page.before,
            more: page.more,
        },
    };
}
