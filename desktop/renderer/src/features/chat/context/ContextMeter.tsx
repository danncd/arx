import { useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useAnchoredPopup } from "../../../ui/popover/useAnchoredPopup";

import type { ContextReport } from "../../../../../contracts/wire.generated";
import {
    EMPTY_CONTEXT,
    RING_CIRCUMFERENCE,
    readingLabel,
    ringDash,
} from "./contextReading";
import { ContextMeterPanel } from "./ContextMeterPanel";
import "../../../ui/popover/popover.css";
import "./context-meter.css";

export function ContextMeter({
    report,
    conversation,
}: {
    report: ContextReport | null;
    conversation: string;
}) {
    const id = useId();
    const trigger = useRef<HTMLButtonElement>(null);
    const panel = useRef<HTMLDivElement>(null);
    const [open, setOpen] = useState(false);
    const placement = useAnchoredPopup(open, trigger, 320, Infinity, 8);

    const reading = report ?? EMPTY_CONTEXT;
    useEffect(() => {
        if (!open) return;
        const onPointerDown = (event: PointerEvent) => {
            const target = event.target;
            if (
                target instanceof Node &&
                (trigger.current?.contains(target) || panel.current?.contains(target))
            )
                return;
            setOpen(false);
        };
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                setOpen(false);
                trigger.current?.focus({ preventScroll: true });
            }
        };
        document.addEventListener("pointerdown", onPointerDown);
        document.addEventListener("keydown", onKeyDown);
        return () => {
            document.removeEventListener("pointerdown", onPointerDown);
            document.removeEventListener("keydown", onKeyDown);
        };
    }, [open]);
    const label = readingLabel(reading);
    return (
        <>
            <button
                ref={trigger}
                type="button"
                className="context-meter"
                aria-label={label}
                aria-expanded={open}
                aria-controls={open ? `${id}-panel` : undefined}
                title={label}
                onClick={() => setOpen((current) => !current)}
            >
                <svg width="14" height="14" viewBox="0 0 14 14" aria-hidden="true">
                    <circle
                        cx="7"
                        cy="7"
                        r="5.5"
                        fill="none"
                        stroke="var(--line)"
                        strokeWidth="2"
                    />
                    <circle
                        cx="7"
                        cy="7"
                        r="5.5"
                        fill="none"
                        stroke="var(--jade)"
                        strokeWidth="2"
                        strokeDasharray={ringDash(reading)}
                        strokeDashoffset="0"
                        transform="rotate(-90 7 7)"
                        data-circumference={RING_CIRCUMFERENCE.toFixed(2)}
                    />
                </svg>
            </button>
            {open &&
                createPortal(
                    <div
                        id={`${id}-panel`}
                        ref={panel}
                        className="popup-surface context-panel"
                        style={placement}
                        role="dialog"
                        aria-label={`Context, ${conversation}`}
                    >
                        <ContextMeterPanel report={reading} />
                    </div>,
                    document.body,
                )}
        </>
    );
}
