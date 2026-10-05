import { CaretDownIcon, CheckIcon, type Icon } from "@phosphor-icons/react";
import { useEffect, useLayoutEffect, useId, useRef, useState } from "react";
import "./select.css";

export function Select<Value extends string>({
    label,
    value,
    choices,
    disabled = false,
    direction = "down",
    compact = false,
    menuTitle,
    onChange,
}: {
    label: string;
    value: Value;
    choices: readonly {
        value: Value;
        label: string;
        description?: string;
        icon?: Icon;
        tone?: "danger";
    }[];
    direction?: "up" | "down";
    compact?: boolean;
    menuTitle?: string;
    disabled?: boolean;
    onChange: (value: Value) => void;
}) {
    const [open, setOpen] = useState(false);
    const [focused, setFocused] = useState(0);
    const root = useRef<HTMLDivElement>(null);
    const trigger = useRef<HTMLButtonElement>(null);
    const options = useRef<(HTMLButtonElement | null)[]>([]);
    const id = useId();
    useLayoutEffect(() => {
        if (!open) return;
        const option = options.current[focused];
        const menu = option?.parentElement;
        if (!option || !menu) return;
        option.focus({ preventScroll: true });
        const top = option.offsetTop;
        const bottom = top + option.offsetHeight;
        if (top < menu.scrollTop) menu.scrollTop = top;
        if (bottom > menu.scrollTop + menu.clientHeight)
            menu.scrollTop = bottom - menu.clientHeight;
    }, [open, focused]);
    useEffect(() => {
        if (!open) return;
        const outside = (event: PointerEvent) => {
            if (event.target instanceof Node && !root.current?.contains(event.target))
                setOpen(false);
        };
        document.addEventListener("pointerdown", outside);
        return () => document.removeEventListener("pointerdown", outside);
    }, [open]);
    const selected = choices.find((choice) => choice.value === value);
    return (
        <div
            className="select-control"
            data-direction={direction}
            data-compact={compact}
            ref={root}
            onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget as Node | null))
                    setOpen(false);
            }}
        >
            <button
                ref={trigger}
                id={`${id}-trigger`}
                type="button"
                role="combobox"
                data-tone={selected?.tone}
                aria-label={label}
                aria-expanded={open}
                aria-controls={`${id}-options`}
                aria-haspopup="listbox"
                disabled={disabled}
                onClick={() => {
                    setFocused(choices.findIndex((choice) => choice.value === value));
                    setOpen(!open);
                }}
                onKeyDown={(event) => {
                    if (["ArrowDown", "ArrowUp"].includes(event.key)) {
                        event.preventDefault();
                        setFocused(choices.findIndex((choice) => choice.value === value));
                        setOpen(true);
                    }
                }}
            >
                <span className="select-option-title">
                    {selected?.icon && <selected.icon size={15} aria-hidden="true" />}
                    <span className="select-trigger-label" title={selected?.label}>
                        {selected?.label}
                    </span>
                </span>
                <CaretDownIcon size={13} />
            </button>
            {open && (
                <div
                    className="select-options"
                    id={`${id}-options`}
                    role="listbox"
                    aria-label={label}
                    onKeyDown={(event) => {
                        if (event.key === "Escape") {
                            event.preventDefault();
                            event.stopPropagation();
                            setOpen(false);
                            trigger.current?.focus();
                        }
                        if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
                            event.preventDefault();
                            setFocused(
                                event.key === "Home"
                                    ? 0
                                    : event.key === "End"
                                      ? choices.length - 1
                                      : (focused +
                                            (event.key === "ArrowDown" ? 1 : -1) +
                                            choices.length) %
                                        choices.length,
                            );
                        }
                    }}
                >
                    {menuTitle && <div className="select-menu-title">{menuTitle}</div>}
                    {choices.map((choice, index) => (
                        <button
                            key={choice.value}
                            ref={(node) => {
                                options.current[index] = node;
                            }}
                            type="button"
                            role="option"
                            data-tone={choice.tone}
                            aria-label={choice.label}
                            aria-description={choice.description}
                            aria-selected={value === choice.value}
                            tabIndex={focused === index ? 0 : -1}
                            onClick={() => {
                                onChange(choice.value);
                                setOpen(false);
                                trigger.current?.focus();
                            }}
                        >
                            <span className="select-option-label">
                                <span className="select-option-title">
                                    {choice.icon && (
                                        <choice.icon size={15} aria-hidden="true" />
                                    )}
                                    {choice.label}
                                </span>
                                {choice.description && (
                                    <small>{choice.description}</small>
                                )}
                            </span>
                            {value === choice.value && <CheckIcon size={14} />}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
}
