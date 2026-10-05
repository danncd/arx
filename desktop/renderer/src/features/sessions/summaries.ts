import type { Snapshot } from "../../../../contracts/wire.generated";
import { compareTime } from "./merge";

export function summary(item: Snapshot["conversations"][number]) {
    return {
        id: item.id,
        title: item.title,
        updatedAt: item.updated,
        status:
            item.status === "running" || item.status === "stopping"
                ? ("running" as const)
                : ("idle" as const),
    };
}

export function mergeSummaries(
    held: Snapshot["conversations"],
    next: Snapshot["conversations"],
    replaceEqual = true,
) {
    const entries = new Map(held.map((item) => [item.id, item]));
    for (const item of next) {
        const previous = entries.get(item.id);
        if (
            !previous ||
            (!previous.title && !!item.title) ||
            compareTime(item.updated, previous.updated) > 0 ||
            (replaceEqual && compareTime(item.updated, previous.updated) === 0)
        )
            entries.set(item.id, item);
    }
    return [...entries.values()].sort((a, b) => compareTime(b.updated, a.updated));
}
