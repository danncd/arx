import { useContext } from "react";
import { CaretRightIcon } from "@phosphor-icons/react";
import { MessageScope, useRememberedState } from "../response/ViewState";
import { usePresentation } from "../../../platform/desktop/Presentation";
import { useNotifications } from "../../../ui/notifications/Notifications";
import type { ToolRecord } from "../../../../../contracts/wire.generated";
import { ToolResultHeight } from "./ToolResultHeight";
import { SiteIcon } from "./SiteIcon";
import { webSources, type WebSource } from "./webSources";
import { prettyInput, toolLabel } from "./toolPresentation";
import "./web-tool.css";

export function WebToolDetails({
    tool,
    onExpand,
}: {
    tool: ToolRecord;
    onExpand: () => void;
}) {
    return (
        <>
            {webSources(tool).map((source, index) => (
                <SiteDetails
                    key={source.url + index}
                    tool={tool}
                    source={source}
                    index={index}
                    onExpand={onExpand}
                />
            ))}
        </>
    );
}
function SiteDetails({
    tool,
    source,
    index,
    onExpand,
}: {
    tool: ToolRecord;
    source: WebSource;
    index: number;
    onExpand: () => void;
}) {
    const scope = useContext(MessageScope);
    const bridge = usePresentation();
    const { notify } = useNotifications();
    const key = `tool:${scope}:${tool.id}:site${index}`;
    const [open, setOpen] = useRememberedState(`${key}:open`, false);
    const [panel, setPanel] = useRememberedState<"in" | "out">(`${key}:panel`, "out");
    const input = prettyInput(tool);
    return (
        <details
            className="tool-file web-site"
            open={open}
            onToggle={(e) => {
                if (e.currentTarget.open !== open) {
                    setOpen(e.currentTarget.open);
                    onExpand();
                }
            }}
        >
            <summary>
                <CaretRightIcon className="tool-chevron" size={10} />
                <SiteIcon url={source.url} />
                <span className="tool-filename">{source.domain}</span>
                <span className="tool-path" title={source.title}>
                    {source.title}
                </span>
            </summary>
            <div className="tool-io">
                <div className="tool-io-tabs">
                    {(["in", "out"] as const).map((value) => (
                        <button
                            type="button"
                            key={value}
                            aria-pressed={panel === value}
                            onClick={() => {
                                setPanel(value);
                                onExpand();
                            }}
                        >
                            {value === "in" ? "In" : "Out"}
                        </button>
                    ))}
                </div>
                <ToolResultHeight>
                    {panel === "in" ? (
                        <pre tabIndex={0} aria-label="Tool input">
                            {input}
                        </pre>
                    ) : (
                        <div className="web-page-output">
                            <a
                                href={source.url}
                                onClick={(e) => {
                                    e.preventDefault();
                                    void bridge
                                        .openExternal(source.url)
                                        .catch(() =>
                                            notify(
                                                "Couldn’t open this page",
                                                "web-source",
                                            ),
                                        );
                                }}
                            >
                                {source.title}
                            </a>
                            <div className="web-source-url">{source.url}</div>
                            <pre tabIndex={0} aria-label="Tool output">
                                {source.text ||
                                    source.snippet ||
                                    "Search result. Open the page to read its content."}
                            </pre>
                            <div className="web-source-note">
                                {toolLabel(tool).key === "web:search"
                                    ? "Search excerpt"
                                    : source.complete
                                      ? "End of retained page"
                                      : "Page excerpt"}
                            </div>
                        </div>
                    )}
                </ToolResultHeight>
            </div>
        </details>
    );
}
