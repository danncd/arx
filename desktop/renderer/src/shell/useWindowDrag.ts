import { useCallback, useEffect, useRef } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

import { useNotifications } from "../ui/notifications/Notifications";

interface ScreenPoint {
    readonly x: number;
    readonly y: number;
}

function isInteractiveTarget(target: EventTarget | null): boolean {
    return (
        target instanceof Element &&
        Boolean(
            target.closest(
                'button, input, textarea, a, [role="button"], [role="menu"], [role="dialog"], [contenteditable="true"]',
            ),
        )
    );
}

export function useWindowDrag() {
    const { notify } = useNotifications();
    const pointerIdRef = useRef<number | null>(null);
    const latestPointRef = useRef<ScreenPoint | null>(null);
    const frameRef = useRef<number | null>(null);

    const flushMove = useCallback(() => {
        frameRef.current = null;
        const point = latestPointRef.current;
        if (point) window.arxDesktop?.moveWindowDrag?.(point);
    }, []);

    const finishDrag = useCallback(() => {
        if (frameRef.current !== null) {
            window.cancelAnimationFrame(frameRef.current);
            flushMove();
        }
        pointerIdRef.current = null;
        latestPointRef.current = null;
        window.arxDesktop?.endWindowDrag?.();
    }, [flushMove]);

    useEffect(() => {
        window.addEventListener("blur", finishDrag);
        return () => {
            window.removeEventListener("blur", finishDrag);
            finishDrag();
        };
    }, [finishDrag]);

    const onPointerDown = useCallback((event: ReactPointerEvent<HTMLElement>) => {
        if (!window.arxDesktop?.beginWindowDrag || event.button !== 0) return;
        if (isInteractiveTarget(event.target)) return;

        pointerIdRef.current = event.pointerId;
        event.currentTarget.setPointerCapture(event.pointerId);
        window.arxDesktop.beginWindowDrag({ x: event.screenX, y: event.screenY });
        event.preventDefault();
    }, []);

    const onPointerMove = useCallback(
        (event: ReactPointerEvent<HTMLElement>) => {
            if (pointerIdRef.current !== event.pointerId) return;
            if (event.pointerType === "mouse" && !(event.buttons & 1)) {
                finishDrag();
                return;
            }
            latestPointRef.current = { x: event.screenX, y: event.screenY };
            if (frameRef.current === null) {
                frameRef.current = window.requestAnimationFrame(flushMove);
            }
        },
        [flushMove, finishDrag],
    );

    const onPointerUp = useCallback(
        (event: ReactPointerEvent<HTMLElement>) => {
            if (pointerIdRef.current !== event.pointerId) return;
            if (event.currentTarget.hasPointerCapture(event.pointerId)) {
                event.currentTarget.releasePointerCapture(event.pointerId);
            }
            finishDrag();
        },
        [finishDrag],
    );

    const onDoubleClick = useCallback(
        (event: React.MouseEvent<HTMLElement>) => {
            if (isInteractiveTarget(event.target)) return;
            void window.arxDesktop
                ?.doubleClickTitleBar?.()
                .catch(() => notify("Could not resize the window. Try again."));
        },
        [notify],
    );

    return {
        onPointerDown,
        onPointerMove,
        onPointerUp,
        onPointerCancel: onPointerUp,
        onLostPointerCapture: () => {
            if (pointerIdRef.current !== null) finishDrag();
        },
        onDoubleClick,
    };
}
