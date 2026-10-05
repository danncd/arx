import { StackSimpleIcon } from "@phosphor-icons/react";
import { useEffect, useId, useRef, useState } from "react";
import type { ContextReport } from "../../../../../contracts/wire.generated";
import { cacheHitRate, formatUsage } from "./format";
import "../../../ui/popover/popover.css";
import "./session-usage.css";

const zeroUsage = { input: 0, output: 0, cached: 0, reasoning: 0, cacheUnknown: false };

export function SessionUsage({
    usage,
    empty,
}: {
    usage: ContextReport["usage"] | undefined;
    empty: boolean;
}) {
    const [open, setOpen] = useState(false);
    const container = useRef<HTMLDivElement>(null);
    const trigger = useRef<HTMLButtonElement>(null);
    const id = useId();
    const reading = empty ? zeroUsage : usage;
    const total = reading ? reading.input + reading.output : null;
    useEffect(() => {
        if (!open) return;
        const outside = (event: PointerEvent) => {
            if (
                event.target instanceof Node &&
                !container.current?.contains(event.target)
            )
                setOpen(false);
        };
        const escape = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                setOpen(false);
                trigger.current?.focus({ preventScroll: true });
            }
        };
        document.addEventListener("pointerdown", outside);
        document.addEventListener("keydown", escape);
        return () => {
            document.removeEventListener("pointerdown", outside);
            document.removeEventListener("keydown", escape);
        };
    }, [open]);
    return (
        <div
            className="session-usage"
            ref={container}
            onBlur={(event) => {
                if (
                    event.relatedTarget &&
                    !event.currentTarget.contains(event.relatedTarget)
                )
                    setOpen(false);
            }}
        >
            <button
                ref={trigger}
                type="button"
                className="session-usage-trigger"
                aria-label={
                    total === null
                        ? "Session token usage"
                        : `Session token usage: ${total.toLocaleString()} tokens`
                }
                aria-expanded={open}
                aria-controls={open ? id : undefined}
                aria-haspopup="dialog"
                title="Session token usage"
                onClick={() => setOpen((value) => !value)}
            >
                <StackSimpleIcon size={15} aria-hidden="true" />
                <span>{total === null ? "—" : formatUsage(total)}</span>
            </button>
            {open && (
                <div
                    id={id}
                    className="popup-surface session-usage-panel"
                    role="dialog"
                    aria-label="Session token usage"
                >
                    <div className="session-usage-heading">Token usage</div>
                    <dl>
                        <Row
                            label="In"
                            value={reading?.input}
                            description="Includes cached input"
                        />
                        <Row label="Out" value={reading?.output} />
                        <Row
                            label="Cache"
                            value={reading?.cacheUnknown ? undefined : reading?.cached}
                            description="Cached tokens included in input"
                        />
                        <div className="session-usage-row">
                            <dt>Hit rate</dt>
                            <dd>
                                {reading && !reading.cacheUnknown
                                    ? cacheHitRate(reading.input, reading.cached)
                                    : "—"}
                            </dd>
                        </div>
                    </dl>
                </div>
            )}
        </div>
    );
}

function Row({
    label,
    value,
    description,
}: {
    label: string;
    value?: number;
    description?: string;
}) {
    return (
        <div className="session-usage-row">
            <dt title={description}>{label}</dt>
            <dd>{value === undefined ? "—" : value.toLocaleString()}</dd>
        </div>
    );
}
