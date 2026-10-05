import { CaretDownIcon } from "@phosphor-icons/react";
import { useContext, useId, useState } from "react";
import { MessageScope, useRememberedState } from "./ViewState";
import { Markdown } from "../../../rich-text/Markdown";
import { useSmoothText } from "./useSmoothText";

export function Thinking({
    text,
    persistenceKey = "tail",
    streaming,
    onExpand,
}: {
    text: string;
    persistenceKey?: string;
    streaming: boolean;
    onExpand: () => void;
}) {
    const revealed = useSmoothText(text, streaming);

    const flat = revealed.replace(/\s+/g, " ").trim();
    const preview = (() => {
        if (!flat) return "";

        return streaming && flat.length > 200 ? flat.slice(-200) : flat.slice(0, 400);
    })();
    const bodyId = useId();
    const scope = useContext(MessageScope);
    const [open, setOpen] = useRememberedState(
        `thought:${scope}:${persistenceKey}`,
        false,
    );

    const [liveOpen, setLiveOpen] = useState(false);
    const expanded = streaming ? liveOpen : open;

    const label = "Think";
    return (
        <div className="thinking">
            <button
                type="button"
                className="thinking-header"
                aria-expanded={expanded}
                aria-controls={bodyId}
                onClick={() => {
                    if (!expanded) onExpand();
                    if (streaming) setLiveOpen(!liveOpen);
                    else setOpen(!open);
                }}
            >
                <span className={`thinking-chevron${expanded ? " open" : ""}`}>
                    <CaretDownIcon size={10} weight="bold" />
                </span>
                <span className="thinking-label">
                    {streaming ? <span className="shimmer-text">{label}</span> : label}
                </span>

                {!expanded && preview && (
                    <span className="thinking-preview">{preview}</span>
                )}
            </button>
            <div
                className={`thinking-reveal${expanded ? " expanded" : ""}`}
                aria-hidden={!expanded}
                inert={!expanded}
            >
                <div className="thinking-clip">
                    <div id={bodyId} className="thinking-body">
                        <Markdown text={revealed.trimEnd()} />
                    </div>
                </div>
            </div>
        </div>
    );
}
