import type { ToolRecord } from "../../../../../contracts/wire.generated";

export type WebSource = {
    url: string;
    domain: string;
    title: string;
    snippet: string;
    text: string;
    complete: boolean;
};

function address(raw: unknown) {
    if (typeof raw !== "string") return "";
    try {
        const url = new URL(raw);
        return ["http:", "https:"].includes(url.protocol) &&
            !url.username &&
            !url.password
            ? url.href
            : "";
    } catch {
        return "";
    }
}

export function webSources(tool: ToolRecord): WebSource[] {
    if (tool.name !== "web" || tool.status !== "done" || tool.failed) return [];
    try {
        const result = JSON.parse(tool.result || "{}");
        const input = JSON.parse(tool.arguments || "{}");
        if (!result || result.failed || !input) return [];
        const fetch = input.operation === "fetch";
        const text: string = typeof result.text === "string" ? result.text : "";
        const source = (
            raw: { url?: unknown; title?: unknown; snippet?: unknown } | null,
        ): WebSource[] => {
            const url = address(raw?.url);
            if (!url) return [];
            const domain = new URL(url).hostname.replace(/^www\./, "");
            return [
                {
                    url,
                    domain,
                    title:
                        typeof raw?.title === "string" && raw.title.trim()
                            ? raw.title
                            : domain,
                    snippet: typeof raw?.snippet === "string" ? raw.snippet : "",
                    text: fetch ? text.replace(/^Source: https?:\/\/[^\n]+\n\n/, "") : "",
                    complete: fetch && result.truncated === false,
                },
            ];
        };
        if (Array.isArray(result.sources))
            return result.sources.flatMap(source).slice(0, 10);
        if (fetch && text) return source({ url: input.url });
        if (input.operation === "search") {
            const lines = text.split("\n");
            return lines
                .flatMap((line, index) =>
                    source({
                        url: line.trim(),
                        title: lines[index - 1],
                        snippet: lines[index + 1],
                    }),
                )
                .slice(0, 10);
        }
    } catch {}
    return [];
}
