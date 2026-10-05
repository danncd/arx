import type { TranscriptChunk } from "../../../../contracts/wire.generated";
import type { Message } from "../chat/types";

const encoder = new TextEncoder();

export function messagesFromChunks(chunks: TranscriptChunk[]): Message[] {
    const messages = new Map<string, Message>();
    const offsets = new Map<string, { text: number; reasoning: number }>();
    for (const chunk of chunks) {
        if (chunk.role !== "user" && chunk.role !== "assistant") continue;
        const message =
            messages.get(chunk.id) ??
            ({
                id: chunk.id,
                role: chunk.role,
                text: "",
                reasoning: "",
                tools: [],
            } as Message);
        const position = offsets.get(chunk.id) ?? { text: 0, reasoning: 0 };
        if (
            chunk.offset !== position.text ||
            (chunk.reasoning_offset || 0) !== position.reasoning
        )
            throw new Error("Session history is incomplete");
        if (chunk.images) message.images = chunk.images;
        message.text += chunk.text || "";
        message.reasoning += chunk.reasoning || "";
        position.text += encoder.encode(chunk.text || "").length;
        position.reasoning += encoder.encode(chunk.reasoning || "").length;
        for (const tool of chunk.tools || []) {
            const index = message.tools!.findIndex((held) => held.id === tool.id);
            if (index < 0) message.tools!.push(tool);
            else message.tools![index] = tool;
        }
        message.createdAt ||= chunk.at;
        message.at = chunk.at || message.at;
        message.usage = chunk.usage || message.usage;
        message.failed = chunk.failed;
        message.reason = chunk.reason;
        message.status = chunk.status;
        messages.set(chunk.id, message);
        offsets.set(chunk.id, position);
    }
    return [...messages.values()];
}
