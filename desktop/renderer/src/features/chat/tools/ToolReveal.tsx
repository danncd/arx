import { useLayoutEffect, useRef, type ReactNode } from "react";

export function ToolReveal({ live, children }: { live: boolean; children: ReactNode }) {
    const outer = useRef<HTMLDivElement>(null);
    useLayoutEffect(() => {
        const element = outer.current;
        if (
            !element ||
            !live ||
            window.matchMedia("(prefers-reduced-motion: reduce)").matches
        )
            return;
        const style = getComputedStyle(element);
        const animation = element.animate(
            [
                { height: "0px" },
                { height: `${element.getBoundingClientRect().height}px` },
            ],
            {
                duration: parseFloat(style.getPropertyValue("--expand-duration")),
                easing: style.getPropertyValue("--expand-easing").trim(),
            },
        );
        return () => animation.cancel();
    }, [live]);
    return (
        <div ref={outer} className="tool-reveal">
            <div className="tool-reveal-content">{children}</div>
        </div>
    );
}
