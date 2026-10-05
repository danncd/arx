import { CaretRightIcon } from "@phosphor-icons/react";
import { useId, useState } from "react";
import type { PermissionRequest } from "../../../../contracts/wire.generated";
import "./permissions.css";

export function Approval({
    request,
    responding,
    onRespond,
}: {
    request: PermissionRequest;
    responding: boolean;
    onRespond: (allow: boolean) => Promise<void>;
}) {
    const { action } = request;
    const [expanded, setExpanded] = useState(false);
    const id = useId();
    const command = action.tool === "bash";
    const memo = action.tool === "memo";
    const mcp = action.tool === "mcp";
    const web = action.tool === "web";
    const read = ["read", "image", "list", "search"].includes(action.operation);
    const title = approvalTitle(action.tool, action.operation);
    const target =
        action.query || action.url || action.command || action.path || action.directory;
    return (
        <section className="approval" aria-label="Permission request" role="region">
            <div className="approval-header">
                <button
                    type="button"
                    className="approval-review"
                    aria-label={
                        command
                            ? "Review command"
                            : web || read
                              ? "Review request"
                              : "Review changes"
                    }
                    aria-expanded={expanded}
                    aria-controls={id}
                    onClick={() => setExpanded(!expanded)}
                >
                    <CaretRightIcon size={12} />
                </button>
                <div className="approval-label">
                    <h3>{title}</h3>
                    <p title={target}>
                        {memo
                            ? (target || "").replace(/^memo_/, "").replaceAll("_", " ")
                            : target}
                    </p>
                </div>
                <div className="approval-actions">
                    <button
                        type="button"
                        disabled={responding}
                        onClick={() => void onRespond(false)}
                    >
                        Deny
                    </button>
                    <button
                        type="button"
                        disabled={responding}
                        onClick={() => void onRespond(true)}
                    >
                        Allow once
                    </button>
                </div>
            </div>
            {expanded && (
                <div id={id} className="approval-details">
                    <p>
                        {mcp
                            ? `MCP server: ${action.server} · Tool: ${action.path}`
                            : memo
                              ? `Memo tool: ${action.path}`
                              : web
                                ? target
                                : action.path || action.directory}
                    </p>
                    {memo || mcp ? (
                        <>
                            <h4>Arguments</h4>
                            <pre>
                                {JSON.stringify(
                                    maskCredentials(action.arguments ?? {}),
                                    null,
                                    2,
                                )}
                            </pre>
                        </>
                    ) : command ? (
                        <pre>{action.command}</pre>
                    ) : (
                        !web &&
                        !memo &&
                        !mcp &&
                        !read &&
                        action.operation !== "mkdir" && (
                            <>
                                {action.before && (
                                    <>
                                        <h4>Before</h4>
                                        <pre>{action.before}</pre>
                                    </>
                                )}
                                <h4>After</h4>
                                <pre>{action.after || "(empty file)"}</pre>
                            </>
                        )
                    )}
                </div>
            )}
        </section>
    );
}

function approvalTitle(tool: string, operation: string) {
    if (tool === "mcp") return "Call MCP tool?";
    if (tool === "memo") return "Change Memo?";
    if (tool === "generate" || tool === "media") return "Generate media?";
    if (tool === "bash") return "Run command?";
    if (tool === "web")
        return operation === "search"
            ? "Search web?"
            : operation === "image"
              ? "Load remote image?"
              : "Read page?";
    const titles: Record<string, string> = {
        image: "View image?",
        read: "Read file?",
        list: "List directory?",
        search: "Search files?",
        mkdir: "Create directory?",
        edit: "Edit file?",
    };
    return titles[operation] || "Write file?";
}
function maskCredentials(value: unknown): unknown {
    if (Array.isArray(value)) return value.map(maskCredentials);
    if (value && typeof value === "object")
        return Object.fromEntries(
            Object.entries(value).map(([key, item]) => [
                key,
                /^(password|secret|token|api_?key|authorization)$/i.test(key)
                    ? "[hidden credential]"
                    : maskCredentials(item),
            ]),
        );
    return value;
}
