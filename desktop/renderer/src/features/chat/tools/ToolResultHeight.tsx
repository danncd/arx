import { useCallback, useLayoutEffect, useRef, type ReactNode } from "react";

export function ToolResultHeight({ children }: { children: ReactNode }) {
    const outer = useRef<HTMLDivElement>(null);
    const content = useRef<HTMLDivElement>(null);
    const measure = useCallback(() => {
        const element = outer.current;
        const node = content.current;
        if (element && node) {
            const height = `${node.getBoundingClientRect().height}px`;
            if (element.style.height !== height) element.style.height = height;
        }
    }, []);

    useLayoutEffect(measure);
    useLayoutEffect(() => {
        const node = content.current;
        if (!node) return;
        let frame = 0;
        const observer = new ResizeObserver(() => {
            cancelAnimationFrame(frame);
            frame = requestAnimationFrame(measure);
        });
        observer.observe(node);
        return () => {
            observer.disconnect();
            cancelAnimationFrame(frame);
        };
    }, [measure]);
    return (
        <div ref={outer} className="tool-result-height">
            <div ref={content}>{children}</div>
        </div>
    );
}
