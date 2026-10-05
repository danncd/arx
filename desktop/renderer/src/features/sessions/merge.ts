import type { Message } from "../chat/types";

export function mergeMessages(older: Message[], newer: Message[]): Message[] {
    const messages = new Map(older.map((message) => [message.id, message]));
    for (const message of newer) {
        const held = messages.get(message.id);
        if (!held || compareTime(message.at || "", held.at || "") >= 0)
            messages.set(message.id, {
                ...message,
                ...(held?.createdAt ? { createdAt: held.createdAt } : {}),
            });
    }
    return [...messages.values()];
}

export function compareTime(left: string, right: string): number {
    const difference = Date.parse(left) - Date.parse(right);
    if (Number.isFinite(difference) && difference !== 0) return difference;
    const fraction = (value: string) =>
        (value.match(/\.(\d+)/)?.[1] || "").padEnd(9, "0");
    return fraction(left).localeCompare(fraction(right));
}
