import type { Snapshot } from "../../../contracts/wire.generated";
import { initialSidebarLayout } from "../shell/useSidebar";
import { messagesFromChunks } from "../features/sessions/history";

export type { InitialState } from "../features/sessions/types";
import type { InitialState } from "../features/sessions/types";

export async function loadState(): Promise<InitialState> {
    const desktop = window.arxDesktop;
    if (!desktop) throw new Error("Open the desktop application or the preview entry.");
    const snapshot: Snapshot = await desktop.request("snapshot", undefined);
    const selected =
        snapshot.views["startup-mode"] === "new"
            ? ""
            : snapshot.conversations[0]?.id || "";
    const summary = snapshot.conversations.find((item) => item.id === selected);
    const history =
        desktop && selected
            ? await desktop.request("history", { conversation: selected })
            : null;
    let layout = initialSidebarLayout;
    try {
        const saved = JSON.parse(snapshot.views["sidebar"] || "null");
        if (saved && typeof saved.open === "boolean" && Number.isFinite(saved.width)) {
            const width = Math.max(200, Math.min(340, saved.width));
            layout = { open: saved.open, width, savedWidth: width };
        }
    } catch {}
    return {
        snapshot,
        selected,
        layout,
        session: summary
            ? {
                  id: selected,
                  title: summary.title,
                  updatedAt: summary.updated,
                  status: "idle",
                  draft: snapshot.views[`draft:${selected}`] || "",
                  messages: history ? messagesFromChunks(history.chunks) : [],
              }
            : undefined,
        before: history?.before || "",
        more: history?.more || false,
    };
}
