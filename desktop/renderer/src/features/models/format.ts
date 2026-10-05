export function formatSize(bytes: number) {
    return `${(bytes / 1e9).toFixed(1)} GB`;
}
