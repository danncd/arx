import { useEffect, useState } from "react";
import type { ContextReport } from "../../../../../contracts/wire.generated";

export function useContextReport({
    conversation,
    model,
    contextWindow,
    directory,
    ready,
    running,
    compacting,
    revision,
}: {
    conversation: string;
    model: string;
    contextWindow?: number;
    directory: string;
    ready: boolean;
    running: boolean;
    compacting: boolean;
    revision: number;
}) {
    const key = JSON.stringify([conversation, model, contextWindow, directory]);
    const [reading, setReading] = useState<{ key: string; report: ContextReport } | null>(
        null,
    );
    const settledRevision = running ? 0 : revision;
    useEffect(() => {
        if (!ready || !window.arxDesktop) return;
        let active = true;
        let pending = false;
        const refresh = async () => {
            if (pending) return;
            pending = true;
            try {
                const report = await window.arxDesktop!.request("chat.context", {
                    conversation,
                    model,
                });
                if (active) setReading({ key, report });
            } catch {
                if (active) setReading(null);
            } finally {
                pending = false;
            }
        };
        void refresh();
        const timer = running
            ? window.setInterval(() => void refresh(), 1000)
            : undefined;
        return () => {
            active = false;
            window.clearInterval(timer);
        };
    }, [key, conversation, model, ready, running, compacting, settledRevision]);
    return ready && reading?.key === key ? reading.report : null;
}
