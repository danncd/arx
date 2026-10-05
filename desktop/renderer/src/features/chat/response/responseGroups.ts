import type { Message } from "../types";
import type { ToolRecord } from "../../../../../contracts/wire.generated";

const encoder = new TextEncoder();
const decoder = new TextDecoder();

export function responseGroups(message: Message) {
    const reasoning = message.reasoning || "";
    const textBytes = encoder.encode(message.text);
    const thoughtBytes = encoder.encode(reasoning);
    let textStart = 0;
    let reasoningStart = 0;
    const groups: {
        text: string;
        thought: string;
        tools: [ToolRecord, ...ToolRecord[]];
    }[] = [];
    for (const tool of message.tools || []) {
        const textEnd = Math.max(textStart, Math.min(textBytes.length, tool.offset || 0));
        const thoughtEnd = Math.max(
            reasoningStart,
            Math.min(thoughtBytes.length, tool.reasoning_offset || 0),
        );
        const text = decoder.decode(textBytes.slice(textStart, textEnd));
        const thought = decoder.decode(thoughtBytes.slice(reasoningStart, thoughtEnd));
        const previous = groups.at(-1);
        if (previous && !text.trim() && !thought.trim()) previous.tools.push(tool);
        else groups.push({ text, thought, tools: [tool] });
        textStart = textEnd;
        reasoningStart = thoughtEnd;
    }
    return {
        groups,
        text: decoder.decode(textBytes.slice(textStart)),
        reasoning: decoder.decode(thoughtBytes.slice(reasoningStart)),
    };
}
