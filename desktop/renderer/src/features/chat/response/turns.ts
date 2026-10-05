import type { Message } from "../types";

const encoder = new TextEncoder();
export type Turn = { message: Message; ids: string[] };

export function responseTurns(messages: Message[]): Turn[] {
    const turns: Turn[] = [];
    for (const message of messages) {
        const last = turns.at(-1);
        if (message.role !== "assistant" || last?.message.role !== "assistant") {
            turns.push({
                ids: [message.id],
                message: { ...message, tools: [...(message.tools || [])] },
            });
            continue;
        }
        const previous = last.message;
        const text = previous.text + (previous.text && message.text ? "\n\n" : "");
        const thought =
            (previous.reasoning || "") +
            (previous.reasoning && message.reasoning ? "\n\n" : "");
        const textOffset = encoder.encode(text).length;
        const reasoningOffset = encoder.encode(thought).length;
        last.ids.push(message.id);
        last.message = {
            ...message,
            id: previous.id,
            createdAt: previous.createdAt || previous.at,
            text: text + message.text,
            reasoning: thought + (message.reasoning || ""),
            tools: [
                ...(previous.tools || []),
                ...(message.tools || []).map((tool) => ({
                    ...tool,
                    offset: textOffset + (tool.offset || 0),
                    reasoning_offset: reasoningOffset + (tool.reasoning_offset || 0),
                })),
            ],
            usage:
                previous.usage || message.usage
                    ? {
                          generation:
                              previous.usage?.generation && message.usage?.generation
                                  ? {
                                        tokens:
                                            previous.usage.generation.tokens +
                                            message.usage.generation.tokens,
                                        seconds:
                                            previous.usage.generation.seconds +
                                            message.usage.generation.seconds,
                                    }
                                  : undefined,
                          input:
                              (previous.usage?.input || 0) + (message.usage?.input || 0),
                          output:
                              (previous.usage?.output || 0) +
                              (message.usage?.output || 0),
                          cached:
                              (previous.usage?.cached || 0) +
                              (message.usage?.cached || 0),
                          reasoning:
                              (previous.usage?.reasoning || 0) +
                              (message.usage?.reasoning || 0),
                      }
                    : null,
        };
    }
    return turns;
}
