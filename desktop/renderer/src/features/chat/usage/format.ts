export function formatUsage(value: number): string {
    const count = Math.max(0, Math.round(value));
    if (count < 1000) return String(count);
    const thousands = Math.round(count / 100) / 10;
    if (thousands < 1000) return `${thousands}k`;
    return `${Math.round(count / 100_000) / 10}M`;
}

export function cacheHitRate(input: number, cached: number): string {
    if (input <= 0) return "—";
    const rate = Math.min(100, Math.max(0, (cached / input) * 100));
    const rounded = Math.round(rate * 10) / 10;
    return rounded === 100 && cached < input ? "<100%" : `${rounded}%`;
}

export function formatGenerationSpeed(generation?: {
    tokens: number;
    seconds: number;
}): string {
    if (!generation || generation.tokens <= 0 || generation.seconds <= 0) return "";
    const speed = generation.tokens / generation.seconds;
    return Number.isFinite(speed)
        ? `${speed.toLocaleString(undefined, { maximumFractionDigits: 1 })} t/s`
        : "";
}
