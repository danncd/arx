import { ModelLoadIndicator } from "./ModelLoadIndicator";
import { ModelSourceIcon } from "./ModelSourceIcon";
import { CaretDownIcon } from "@phosphor-icons/react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { useModelMenu, type ModelOption } from "./useModelMenu";
import { ModelMenuContent } from "./ModelMenuContent";
import "./model-options.css";
import "./model-picker.css";
import "../../../ui/popover/popover.css";

export function ModelMenu({
    value,
    options,
    disabled,
    onChange,
    children,
}: {
    value: string;
    options: ModelOption[];
    disabled: boolean;
    onChange: (id: string) => Promise<boolean | undefined>;
    children: ReactNode;
}) {
    const state = useModelMenu(value, options, disabled, onChange);
    const {
        id,
        trigger,
        menu,
        open,
        close,
        showSettings,
        selected,
        placement,
        lastFocused,
    } = state;
    return (
        <>
            <button
                ref={trigger}
                type="button"
                className="select-trigger select-compact"
                role="combobox"
                aria-label="Chat model"
                aria-haspopup="dialog"
                aria-expanded={open}
                aria-controls={open ? id : undefined}
                disabled={disabled}
                onClick={() => {
                    if (open) close();
                    else showSettings();
                }}
                onKeyDown={(event) => {
                    if (["ArrowDown", "ArrowUp"].includes(event.key)) {
                        event.preventDefault();
                        showSettings();
                    }
                }}
            >
                {selected && <ModelSourceIcon model={selected.value} />}
                <span>{selected?.label || "Choose model"}</span>
                {selected?.unloaded && <ModelLoadIndicator />}
                <CaretDownIcon aria-hidden="true" />
            </button>
            {open &&
                createPortal(
                    <div
                        ref={menu}
                        id={id}
                        role="dialog"
                        aria-modal="true"
                        aria-label="Model settings"
                        className="popup-surface model-popup"
                        style={placement}
                        onFocusCapture={(event) => {
                            if (event.target instanceof HTMLElement)
                                lastFocused.current = event.target;
                        }}
                    >
                        <ModelMenuContent state={state} value={value} options={options}>
                            {children}
                        </ModelMenuContent>
                    </div>,
                    document.body,
                )}
        </>
    );
}
