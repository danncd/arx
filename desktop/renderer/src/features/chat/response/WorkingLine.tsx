import type { ChatEvent } from "../../../../../contracts/wire.generated";
import { useEffect, useState } from "react";

function elapsedTime(startedAt: string) {
    const start = Date.parse(startedAt);
    return Number.isFinite(start)
        ? Math.max(0, Math.round((Date.now() - start) / 1000))
        : null;
}

export function formatElapsed(seconds: number) {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m ${seconds % 60}s`;
}

export function WorkingLine({
    status,
    startedAt,
    recovery,
}: {
    status: string;
    startedAt: string;
    recovery?: ChatEvent["recovery"];
}) {
    const running = ["running", "cancelling", "compacting"].includes(status);
    const [elapsed, setElapsed] = useState(() => elapsedTime(startedAt));
    useEffect(() => {
        if (!running) return;
        setElapsed(elapsedTime(startedAt));
        const timer = setInterval(() => setElapsed(elapsedTime(startedAt)), 1000);
        return () => clearInterval(timer);
    }, [startedAt, running]);

    return (
        <div className={`working-line${running ? "" : " done"}`} aria-hidden={!running}>
            <span className="working-glyph" aria-hidden="true">
                ✳
            </span>
            <span className="shimmer-text" role="status">
                {status === "cancelling"
                    ? "Stopping"
                    : recovery && running
                      ? `${recovery.kind === "tool_limit" ? "Continuing" : "Recovering"} (${recovery.attempt}/${recovery.limit})`
                      : status === "compacting"
                        ? "Compacting"
                        : "Working"}
            </span>
            <span className="live-dots" aria-hidden="true">
                <span>.</span>
                <span>.</span>
                <span>.</span>
            </span>
            {elapsed !== null && (
                <span className="working-time">{formatElapsed(elapsed)}</span>
            )}
        </div>
    );
}
