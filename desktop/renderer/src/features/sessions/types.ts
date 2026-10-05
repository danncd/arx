import type { Snapshot } from "../../../../contracts/wire.generated";
import type { SidebarLayout } from "../../shell/useSidebar";
import type { Message } from "../chat/types";

export type ConversationSummary = {
    id: string;
    title: string;
    updatedAt: string;
    status: "idle" | "running";
};

export type Session = ConversationSummary & {
    draft: string;
    messages: Message[];
};

export type InitialState = {
    snapshot: Snapshot;
    selected: string;
    session?: Session;
    before: string;
    more: boolean;
    layout: SidebarLayout;
};
