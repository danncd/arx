import type { ContextReport } from "../../../../../contracts/wire.generated";

export const EMPTY_CONTEXT: ContextReport = {
    usage: { input: 0, cached: 0, output: 0, reasoning: 0 },
    parts: { system: 0, tools: 0, messages: 0, summary: 0 },
    known: false,
    used: 0,
    limit: 0,
    basis: "",
};

export const RING_RADIUS = 5.5;
export const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

export type Segment = {
    key: "system" | "tools" | "messages" | "summary";
    label: string;
    tokens: number;
    share: number;
};

function share(tokens: number, limit: number): number {
    if (!(limit > 0) || !(tokens > 0)) return 0;
    return Math.min(1, tokens / limit);
}

export function usedShare(report: ContextReport): number {
    return share(report.used, report.limit);
}

export function usedPercent(report: ContextReport): number {
    return Math.round(usedShare(report) * 100);
}

export function segments(report: ContextReport): Segment[] {
    const values = report.parts ?? {
        system: 0,
        tools: 0,
        messages: report.used,
        summary: 0,
    };
    const labels = {
        system: "System prompt",
        tools: "Tool definitions",
        messages: "Messages",
        summary: "Summary",
    };
    return (["system", "tools", "messages", "summary"] as const)
        .filter((key) => key !== "summary" || values.summary > 0)
        .map((key) => ({
            key,
            label: labels[key],
            tokens: values[key],
            share: share(values[key], Math.max(report.limit, report.used)),
        }));
}

export function ringDash(report: ContextReport): string {
    const used = usedShare(report) * RING_CIRCUMFERENCE;
    return `${used.toFixed(2)} ${(RING_CIRCUMFERENCE - used).toFixed(2)}`;
}

export function formatTokens(value: number): string {
    const scaled = (candidate: number) =>
        candidate >= 100
            ? String(Math.round(candidate))
            : String(Math.round(candidate * 10) / 10);
    if (value < 1000) return String(value);
    if (value < 1_000_000) {
        const thousands = scaled(value / 1000);
        if (Number(thousands) < 1000) return `${thousands}K`;
    }
    return `${scaled(value / 1_000_000)}M`;
}

export function formatReading(report: ContextReport): string {
    if (!report.known || !(report.limit > 0)) return "—";
    return `~${formatTokens(report.used)} / ${formatTokens(report.limit)}`;
}

export function readingLabel(report: ContextReport): string {
    if (!report.known || !(report.limit > 0)) return "Context unavailable";
    const percent = usedPercent(report);
    const value = report.used > 0 && percent === 0 ? "<1" : String(percent);
    return `${value}% of context used`;
}
