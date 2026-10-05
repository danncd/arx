import type { Session } from "../../features/sessions/types";
import { previewPrompt, previewResponse } from "./markdown";

export const previews: Session[] = [
    {
        id: "markdown-preview",
        title: "Markdown preview",
        updatedAt: "2026-09-26T12:00:00Z",
        status: "idle",
        draft: "",
        messages: [
            { id: "question", role: "user", text: previewPrompt },
            { id: "answer", role: "assistant", text: previewResponse },
        ],
    },
    {
        id: "project-notes",
        title: "Project notes",
        updatedAt: "2026-09-26T11:00:00Z",
        status: "idle",
        draft: "",
        messages: [
            { id: "question", role: "user", text: "Help me outline a small project." },
            {
                id: "answer",
                role: "assistant",
                text: "## Project outline\n\nLorem ipsum dolor sit amet, consectetur adipiscing elit. Integer vitae justo quis neque tincidunt facilisis.\n\n### First steps\n\n1. Define the scope.\n2. Build a small working version.\n3. Review and refine.\n\n> Start with the essentials.",
            },
        ],
    },
];
