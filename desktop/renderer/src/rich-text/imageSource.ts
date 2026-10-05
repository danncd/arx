export function imageURL(source: string) {
    if (
        /^arx-image:[a-f0-9]{64}$/.test(source) ||
        /^arx-media:[a-f0-9]{32}$/.test(source)
    )
        return source;
    try {
        const url = new URL(source);
        if (
            !["http:", "https:"].includes(url.protocol) ||
            url.username ||
            url.password ||
            source.length > 8192
        )
            return "";
        return url.href;
    } catch {
        return "";
    }
}

export function generatedImageURL(reference: string): string {
    return /^arx-media:[a-f0-9]{32}$/.test(reference)
        ? `arx-media://artifact/${reference.slice(10)}`
        : "";
}
