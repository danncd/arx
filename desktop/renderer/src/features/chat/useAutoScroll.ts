import { useCallback, useLayoutEffect, useRef, useState } from "react";

export function useAutoScroll() {
    const container = useRef<HTMLDivElement>(null);
    const content = useRef<HTMLDivElement>(null);
    const pinned = useRef(true);
    const lastTop = useRef(0);
    const [away, setAway] = useState(false);
    const onScroll = useCallback(() => {
        const element = container.current;
        if (!element) return;
        const distance = element.scrollHeight - element.scrollTop - element.clientHeight;
        if (distance <= 32) pinned.current = true;
        else if (element.scrollTop < lastTop.current) pinned.current = false;
        lastTop.current = element.scrollTop;
        setAway(!pinned.current && distance > 32);
    }, []);
    useLayoutEffect(() => {
        const element = container.current;
        const body = content.current;
        if (!element || !body) return;
        const follow = () => {
            if (pinned.current) {
                element.scrollTop = element.scrollHeight;
                lastTop.current = element.scrollTop;
            }
            setAway(
                !pinned.current &&
                    element.scrollHeight - element.scrollTop - element.clientHeight > 32,
            );
        };
        const wheel = (event: WheelEvent) => {
            if (event.deltaY < 0) pinned.current = false;
        };
        follow();
        const observer = new ResizeObserver(follow);
        observer.observe(body);
        observer.observe(element);
        element.addEventListener("wheel", wheel, { passive: true });
        return () => {
            observer.disconnect();
            element.removeEventListener("wheel", wheel);
        };
    }, []);
    const release = useCallback(() => {
        pinned.current = false;
    }, []);
    const pin = () => {
        pinned.current = true;
        setAway(false);
        const element = container.current;
        if (element) {
            element.scrollTop = element.scrollHeight;
            lastTop.current = element.scrollTop;
        }
    };
    return { container, content, onScroll, away, release, pin };
}
