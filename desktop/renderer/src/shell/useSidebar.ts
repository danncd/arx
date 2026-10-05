import {
    useEffect,
    useRef,
    useState,
    type KeyboardEvent,
    type PointerEvent,
    type Dispatch,
    type SetStateAction,
} from "react";

export type SidebarLayout = { open: boolean; width: number; savedWidth: number };
export const initialSidebarLayout: SidebarLayout = {
    open: true,
    width: 252,
    savedWidth: 252,
};

const minimumWidth = 200;

const collapseThreshold = 150;

export function useSidebar({
    layout,
    setLayout,
    side = "left",
    maximumWidth = 340,
}: {
    layout: SidebarLayout;
    setLayout: Dispatch<SetStateAction<SidebarLayout>>;
    side?: "left" | "right";
    maximumWidth?: number;
}) {
    const direction = side === "right" ? -1 : 1;
    const [dragging, setDragging] = useState(false);
    const endDrag = useRef<((commit?: boolean) => void) | null>(null);

    useEffect(() => () => endDrag.current?.(), []);

    const toggle = () => {
        endDrag.current?.(true);
        setLayout((current) => ({
            ...current,
            open: !current.open,
            width: current.open ? current.width : current.savedWidth,
        }));
    };

    const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
        if (event.key === "Enter") {
            event.preventDefault();
            toggle();
        } else if (["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) {
            event.preventDefault();
            setLayout((current) => {
                const next =
                    event.key === "Home"
                        ? minimumWidth
                        : event.key === "End"
                          ? maximumWidth
                          : current.width +
                            (event.key === "ArrowLeft" ? -10 : 10) * direction;
                const width = Math.max(minimumWidth, Math.min(maximumWidth, next));
                return { open: true, width, savedWidth: width };
            });
        }
    };

    const onPointerDown = (event: PointerEvent<HTMLDivElement>) => {
        if (event.button !== 0 || !event.isPrimary) return;
        event.preventDefault();
        endDrag.current?.(true);
        const element = event.currentTarget;
        const pointerId = event.pointerId;
        const startX = event.clientX;
        const startLayout = layout;
        const startWidth = layout.open ? layout.width : 0;
        const resistanceStart = Math.min(210, startWidth || 210);
        let latestX = startX;
        let frame: number | null = null;
        let finished = false;
        element.setPointerCapture(pointerId);
        setDragging(true);

        const apply = () => {
            frame = null;
            const requested = startWidth + (latestX - startX) * direction;
            if (requested < collapseThreshold) {
                setLayout((current) => ({ ...current, open: false }));
                return;
            }
            const width = Math.min(
                maximumWidth,
                requested < resistanceStart
                    ? resistanceStart - (resistanceStart - requested) * 0.25
                    : requested,
            );
            setLayout((current) => ({ ...current, open: true, width }));
        };

        const move = (next: globalThis.PointerEvent) => {
            if (next.pointerId !== pointerId) return;
            if (next.pointerType === "mouse" && !(next.buttons & 1)) {
                finish(true);
                return;
            }
            latestX = next.clientX;
            if (frame === null) frame = requestAnimationFrame(apply);
        };
        const up = (next: globalThis.PointerEvent) => {
            if (next.pointerId !== pointerId) return;
            latestX = next.clientX;
            finish(true);
        };
        const cancel = (next: globalThis.PointerEvent) => {
            if (next.pointerId === pointerId) finish(false);
        };
        const blur = () => finish(false);
        const finish = (commit?: boolean) => {
            if (finished) return;
            finished = true;
            if (frame !== null) cancelAnimationFrame(frame);
            window.removeEventListener("pointermove", move);
            window.removeEventListener("pointerup", up);
            window.removeEventListener("pointercancel", cancel);
            window.removeEventListener("blur", blur);
            endDrag.current = null;
            if (element.hasPointerCapture(pointerId))
                element.releasePointerCapture(pointerId);
            if (commit === undefined) return;
            if (commit) {
                apply();
                setLayout((current) => {
                    const width = current.open
                        ? Math.max(minimumWidth, Math.min(maximumWidth, current.width))
                        : startLayout.savedWidth;
                    return { open: current.open, width, savedWidth: width };
                });
            } else setLayout(startLayout);
            setDragging(false);
        };

        endDrag.current = finish;
        window.addEventListener("pointermove", move);
        window.addEventListener("pointerup", up);
        window.addEventListener("pointercancel", cancel);
        window.addEventListener("blur", blur);
    };

    return {
        open: layout.open,
        width: layout.width,

        dragging,
        resizing: dragging && layout.open,
        maximumWidth,
        toggle,
        onKeyDown,
        onPointerDown,
    };
}
