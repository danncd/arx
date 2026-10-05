import { formatGenerationSpeed } from "../usage/format";
import type { Message } from "../types";
import { CopyReply } from "../CopyReply";
import { MessageScope } from "./ViewState";
import { ResponseContent } from "./ResponseContent";
import { formatElapsed } from "./WorkingLine";

export function AssistantMessage({
    message,
    streaming,
    onExpand,
}: {
    message: Message;
    streaming: boolean;
    onExpand: () => void;
}) {
    const started = message.createdAt || message.at;
    const elapsed = Date.parse(message.at || "") - Date.parse(started || "");
    const interrupted =
        !streaming && ["running", "streaming", "working"].includes(message.status || "");
    const stopLabel =
        message.status === "cancelled"
            ? "Stopped"
            : message.failed
              ? "Failed"
              : interrupted
                ? "Interrupted"
                : "";
    const metadata = [
        Number.isFinite(elapsed) && elapsed >= 0
            ? formatElapsed(Math.round(elapsed / 1000))
            : "",
        message.usage
            ? `${message.usage.input.toLocaleString()} in · ${message.usage.output.toLocaleString()} out`
            : "",
        formatGenerationSpeed(message.usage?.generation),
    ]
        .filter(Boolean)
        .join(" · ");
    return (
        <MessageScope.Provider value={message.id}>
            <div className="agent-label">
                <img src="./arx.png" width={19} height={19} alt="" />
                {started && (
                    <time className="agent-time" dateTime={started}>
                        {new Date(started).toLocaleTimeString([], {
                            hour: "numeric",
                            minute: "2-digit",
                        })}
                    </time>
                )}
            </div>
            <ResponseContent
                message={message}
                streaming={streaming}
                onExpand={onExpand}
            />
            {message.failed && (
                <div className="reply-failure">
                    <p>{message.reason || "Reply failed"}</p>
                </div>
            )}
            {(message.text || message.reasoning || stopLabel) && (
                <div
                    className={`message-footer${streaming ? " reserved" : ""}`}
                    aria-hidden={streaming}
                    inert={streaming}
                >
                    {message.text && <CopyReply text={message.text} />}
                    {(metadata || stopLabel) && (
                        <span className="turn-meta">
                            {metadata}
                            {stopLabel && (
                                <>
                                    {metadata && " · "}
                                    <span className="turn-stop-status">{stopLabel}</span>
                                </>
                            )}
                        </span>
                    )}
                </div>
            )}
        </MessageScope.Provider>
    );
}
