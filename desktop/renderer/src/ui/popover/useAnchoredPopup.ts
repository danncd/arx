import { useLayoutEffect, useState, type CSSProperties, type RefObject } from "react";

export function useAnchoredPopup(
    open: boolean,
    trigger: RefObject<HTMLElement | null>,
    width: number,
    height = 360,
    gap = 6,
) {
    const [placement, setPlacement] = useState<CSSProperties>({
        width,
        visibility: "hidden",
    });

    useLayoutEffect(() => {
        if (!open || !trigger.current) return;
        const button = trigger.current;
        const place = () => {
            const bounds = button.getBoundingClientRect();
            const margin = 10;
            const availableWidth = Math.min(width, window.innerWidth - margin * 2);
            const above = bounds.top - gap - margin;
            const below = window.innerHeight - bounds.bottom - gap - margin;
            const upward = above >= below;
            const next: CSSProperties = {
                width: availableWidth,
                maxHeight: Math.max(0, Math.min(height, upward ? above : below)),
                left: Math.max(
                    margin,
                    Math.min(
                        bounds.right - availableWidth,
                        window.innerWidth - availableWidth - margin,
                    ),
                ),
                top: upward ? undefined : Math.max(margin, bounds.bottom + gap),
                bottom: upward
                    ? Math.max(margin, window.innerHeight - bounds.top + gap)
                    : undefined,
            };
            setPlacement((current) =>
                Object.keys({ ...current, ...next }).every(
                    (key) =>
                        current[key as keyof CSSProperties] ===
                        next[key as keyof CSSProperties],
                )
                    ? current
                    : next,
            );
        };
        place();
        const observer = new ResizeObserver(place);
        observer.observe(button);
        window.addEventListener("resize", place);
        window.addEventListener("scroll", place, true);
        return () => {
            observer.disconnect();
            window.removeEventListener("resize", place);
            window.removeEventListener("scroll", place, true);
        };
    }, [open, trigger, width, height, gap]);

    return placement;
}
