import { ModelLoadIndicator } from "./ModelLoadIndicator";
import { ModelSourceIcon } from "./ModelSourceIcon";
import { CaretLeftIcon, CaretRightIcon, CheckIcon } from "@phosphor-icons/react";
import type { ReactNode } from "react";
import type { ModelOption, useModelMenu } from "./useModelMenu";

export function ModelMenuContent({
    state,
    value,
    options,
    children,
}: {
    state: ReturnType<typeof useModelMenu>;
    value: string;
    options: ModelOption[];
    children: ReactNode;
}) {
    const {
        models,
        showSettings,
        list,
        id,
        active,
        pending,
        handleListKey,
        busy,
        highlight,
        choose,
        modelRow,
        showModels,
        selected,
    } = state;
    return (
        <>
            {models ? (
                <>
                    <button
                        type="button"
                        className="model-back"
                        onClick={() => showSettings()}
                    >
                        <CaretLeftIcon aria-hidden="true" />
                        Models
                    </button>
                    <div
                        ref={list}
                        role="listbox"
                        aria-label="Models"
                        aria-activedescendant={`${id}-${active}`}
                        aria-busy={pending}
                        tabIndex={0}
                        className="model-options"
                        onKeyDown={handleListKey}
                    >
                        {options.map((option, index) => (
                            <div key={option.value}>
                                {(index === 0 ||
                                    options[index - 1]?.provider !== option.provider) && (
                                    <div className="model-section">{option.provider}</div>
                                )}
                                <div
                                    id={`${id}-${index}`}
                                    role="option"
                                    aria-label={option.label}
                                    aria-selected={option.value === value}
                                    aria-disabled={Boolean(option.unavailable) || busy}
                                    title={
                                        option.unavailable ||
                                        (option.unloaded
                                            ? "Not loaded — loads when you send a message"
                                            : undefined)
                                    }
                                    className={`select-option has-description${active === index ? " active" : ""}`}
                                    onPointerMove={() => highlight(index)}
                                    onPointerDown={(event) => event.preventDefault()}
                                    onClick={() => void choose(index)}
                                >
                                    <ModelSourceIcon model={option.value} />
                                    <span className="select-option-copy">
                                        <span>{option.label}</span>
                                        <small>{option.provider}</small>
                                    </span>
                                    {option.unloaded && <ModelLoadIndicator />}
                                    {option.value === value && (
                                        <CheckIcon aria-hidden="true" />
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                </>
            ) : (
                <>
                    <button
                        ref={modelRow}
                        type="button"
                        className="model-current"
                        onClick={showModels}
                        disabled={pending}
                        aria-label="Choose model"
                    >
                        {selected && <ModelSourceIcon model={selected.value} />}
                        <span>
                            <strong>{selected?.label || "Choose model"}</strong>
                            <small>{selected?.provider}</small>
                        </span>
                        {selected?.unloaded && <ModelLoadIndicator />}
                        <CaretRightIcon aria-hidden="true" />
                    </button>
                    {children}
                </>
            )}
        </>
    );
}
