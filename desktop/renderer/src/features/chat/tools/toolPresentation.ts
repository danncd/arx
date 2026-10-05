import type { ToolRecord } from "../../../../../contracts/wire.generated";

export function toolLabel(tool: ToolRecord) {
    let input: Record<string, unknown> = {};
    try {
        const parsed = JSON.parse(tool.arguments);
        if (parsed && typeof parsed === "object" && !Array.isArray(parsed))
            input = parsed;
    } catch {
        if (tool.status === "preparing") {
            const fields =
                /"(operation|path|url|query|command)"\s*:\s*("(?:\\.|[^"\\])*")/g;
            for (const match of tool.arguments.slice(0, 4096).matchAll(fields)) {
                const [, field, value] = match;
                if (!field || !value) continue;
                try {
                    input[field] ??= JSON.parse(value);
                } catch {}
            }
        }
    }
    const text = (value: unknown) => (typeof value === "string" ? value : "");
    const operation = text(input.operation);
    const key = ["files", "web", "generate", "media", "memo", "mcp", "skills"].includes(
        tool.name,
    )
        ? `${tool.name}:${operation}`
        : tool.name;
    const labels: Record<string, string> = {
        "files:read": "Read",
        "files:image": "View image",
        "files:list": "List files",
        "files:search": "Search text",
        "files:write": "Write file",
        "files:edit": "Edit file",
        "files:mkdir": "Create folder",
        "web:search": "Search web",
        "web:fetch": "Read page",
        "web:image": "View image",
        bash: "Run command",
        "generate:image": "Generate image",
        "generate:video": "Generate video",
        "generate:speech": "Generate speech",
        "media:combine": "Combine media",
        "mcp:list": "Discover MCP tools",
        "mcp:describe": "Inspect MCP tool",
        "mcp:call": "Call MCP tool",
        "skills:list": "List skills",
        "skills:load": "Load skill",
        "skills:read": "Read skill reference",
        "memo:": "Using Memo",
        "memo:list": "Using Memo",
        "memo:describe": "Using Memo",
        "memo:call": "Using Memo",
    };
    const path = text(input.path);
    const full =
        path ||
        text(input.url) ||
        text(input.query) ||
        text(input.command) ||
        text(input.model) ||
        (tool.name === "mcp"
            ? [text(input.server), text(input.name)].filter(Boolean).join(" · ")
            : tool.name === "skills"
              ? text(input.id)
              : "") ||
        (tool.name === "memo"
            ? text(input.name)
                  .replace(/^memo_/, "")
                  .replaceAll("_", " ")
            : "");
    let target = path ? path.split("/").filter(Boolean).at(-1) || path : full;
    if (key === "web:fetch" || key === "web:image") {
        try {
            target = new URL(full).hostname.replace(/^www\./, "");
        } catch {}
    }
    return { key, action: labels[key] || tool.name, target, full };
}

export function groupCalls(tools: ToolRecord[]) {
    const groups: [ToolRecord, ...ToolRecord[]][] = [];
    for (const tool of tools) {
        const last = groups.at(-1);
        const key = toolLabel(tool).key;
        const preparingOperation = tool.status === "preparing" && key === `${tool.name}:`;
        if (
            last &&
            (toolLabel(last[0]).key === key ||
                (preparingOperation && last[0].name === tool.name))
        )
            last.push(tool);
        else groups.push([tool]);
    }
    return groups;
}

export function toolStatus(tools: ToolRecord[], live: boolean) {
    const pending = tools.some((tool) =>
        ["preparing", "pending", "running"].includes(tool.status),
    );
    const active = live && pending;
    const failed = tools.some((tool) => tool.failed);
    return {
        active,
        failed,
        stopped: pending && !live,
        label: active ? "Working…" : failed ? "Failed" : pending ? "Stopped" : "Done",
    };
}

export function toolOutput(tool: ToolRecord, live: boolean) {
    if (!tool.result)
        return live && tool.status !== "done" ? "Waiting for result…" : "No result";
    try {
        const parsed = JSON.parse(tool.result);
        if (typeof parsed.text === "string") return parsed.text;
    } catch {}
    return tool.result;
}

export function prettyInput(tool: ToolRecord) {
    try {
        return JSON.stringify(JSON.parse(tool.arguments), null, 4);
    } catch {
        return tool.arguments;
    }
}

export function groupLabel(tools: [ToolRecord, ...ToolRecord[]], sourceCounts: number[]) {
    const label = toolLabel(tools[0]);
    if (tools.length === 1) return label;
    if (label.key === "web:search") {
        const results = sourceCounts.reduce((sum, count) => sum + count, 0);
        const completed = tools.some((tool) => tool.status === "done" && !tool.failed);
        return {
            ...label,
            target: `${tools.length} searches${completed ? ` · ${results} ${results === 1 ? "result" : "results"}` : ""}`,
            full: "",
        };
    }
    if (label.key === "web:fetch")
        return {
            ...label,
            action: "Read pages",
            target: `${tools.length} page reads`,
            full: "",
        };
    return { ...label, target: `${tools.length} calls` };
}
