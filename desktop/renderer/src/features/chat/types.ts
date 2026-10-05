import type { ToolRecord, TranscriptChunk } from "../../../../contracts/wire.generated";

export interface Message {
    images?: import("../../../../contracts/wire.generated").ImageAttachment[];
    id: string;
    role: "user" | "assistant";
    text: string;
    reasoning?: string;
    tools?: ToolRecord[];
    at?: string;
    createdAt?: string;
    usage?: TranscriptChunk["usage"];
    status?: string;
    failed?: boolean;
    reason?: string;
}
