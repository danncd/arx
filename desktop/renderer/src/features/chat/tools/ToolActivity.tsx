import { useGenerationProgress } from "./useGenerationProgress";
import { WebToolDetails } from "./WebToolDetails";
import { webSources } from "./webSources";
import { useContext } from "react";
import {
    CaretRightIcon,
    PlugsIcon,
    BookOpenIcon,
    CheckIcon,
    CircleNotchIcon,
    FileTextIcon,
    PencilSimpleIcon,
    FolderIcon,
    MagnifyingGlassIcon,
    WarningCircleIcon,
    TerminalIcon,
    GlobeIcon,
    ImageIcon,
    FilmStripIcon,
    WaveformIcon,
} from "@phosphor-icons/react";
import type { ToolRecord } from "../../../../../contracts/wire.generated";
import { MessageScope, useRememberedState } from "../response/ViewState";
import { ToolResultHeight } from "./ToolResultHeight";
import {
    groupCalls,
    groupLabel,
    toolLabel,
    toolStatus,
    toolOutput,
    prettyInput,
} from "./toolPresentation";
import "./tool-activity.css";

type Props = { tools: ToolRecord[]; live: boolean; onExpand: () => void };

function CallDetails({
    tool,
    live,
    onExpand,
}: {
    tool: ToolRecord;
    live: boolean;
    onExpand: () => void;
}) {
    const scope = useContext(MessageScope);
    const [panel, setPanel] = useRememberedState<"in" | "out">(
        `tool:${scope}:${tool.id}:panel`,
        "out",
    );
    const [open, setOpen] = useRememberedState(`tool:${scope}:${tool.id}:details`, false);
    const label = toolLabel(tool);
    return (
        <details
            className="tool-file"
            open={open}
            onToggle={(event) => {
                if (event.currentTarget.open !== open) {
                    setOpen(event.currentTarget.open);
                    onExpand();
                }
            }}
        >
            <summary>
                <CaretRightIcon className="tool-chevron" size={10} />
                <span className="tool-filename">{label.action}</span>
                <span className="tool-path" title={label.full}>
                    {label.target}
                </span>
            </summary>
            <div className="tool-io">
                <div className="tool-io-tabs" aria-label="Tool details">
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
                    <pre
                        tabIndex={0}
                        aria-label={panel === "in" ? "Tool input" : "Tool output"}
                    >
                        {panel === "in" ? prettyInput(tool) : toolOutput(tool, live)}
                    </pre>
                </ToolResultHeight>
            </div>
        </details>
    );
}

function Group({
    tools,
    live,
    onExpand,
}: Omit<Props, "tools"> & { tools: [ToolRecord, ...ToolRecord[]] }) {
    const first = tools[0];
    const scope = useContext(MessageScope);
    const [open, setOpen] = useRememberedState(`tool:${scope}:${first.id}:group`, false);
    const label = groupLabel(
        tools,
        tools.map((tool) => webSources(tool).length),
    );
    const status = toolStatus(tools, live);
    const statusLabel =
        status.active && ["generate", "media"].includes(first.name) && first.summary
            ? `${status.label} · ${first.summary}`
            : status.label;
    const Icon =
        label.key === "generate:image"
            ? ImageIcon
            : label.key === "generate:speech"
              ? WaveformIcon
              : first.name === "media" || label.key === "generate:video"
                ? FilmStripIcon
                : first.name === "mcp"
                  ? PlugsIcon
                  : first.name === "skills"
                    ? BookOpenIcon
                    : first.name === "bash"
                      ? TerminalIcon
                      : first.name === "web"
                        ? GlobeIcon
                        : label.key === "files:read"
                          ? FileTextIcon
                          : ["files:write", "files:edit"].includes(label.key)
                            ? PencilSimpleIcon
                            : ["files:list", "files:mkdir"].includes(label.key)
                              ? FolderIcon
                              : MagnifyingGlassIcon;
    return (
        <details
            className="tool-group"
            open={open}
            onToggle={(event) => {
                if (event.currentTarget.open !== open) {
                    setOpen(event.currentTarget.open);
                    onExpand();
                }
            }}
        >
            <summary className="tool-group-summary">
                <Icon size={15} className="tool-kind" />
                <span className="tool-title">
                    <span className="tool-action">{label.action}</span>
                    {(label.target || tools.length > 1) && (
                        <>
                            <span className="tool-separator" aria-hidden="true">
                                {" "}
                                ·{" "}
                            </span>
                            <span className="tool-target" title={label.full}>
                                {label.target}
                            </span>
                        </>
                    )}
                </span>
                <span
                    className={`tool-status${status.active ? " live" : status.failed ? " warning" : ""}`}
                    aria-label={statusLabel}
                    title={statusLabel}
                >
                    {status.active ? (
                        <CircleNotchIcon size={13} className="tool-spinner" />
                    ) : status.failed ? (
                        <WarningCircleIcon size={13} />
                    ) : status.stopped ? null : (
                        <CheckIcon size={13} />
                    )}
                    {status.label !== "Done" && statusLabel}
                </span>
                <CaretRightIcon size={12} className="tool-chevron" />
            </summary>
            <div className="tool-files">
                {tools.map((tool) =>
                    webSources(tool).length > 0 ? (
                        <WebToolDetails key={tool.id} tool={tool} onExpand={onExpand} />
                    ) : (
                        <CallDetails
                            key={tool.id}
                            tool={tool}
                            live={live}
                            onExpand={onExpand}
                        />
                    ),
                )}
            </div>
        </details>
    );
}

export function ToolActivity({ tools, live, onExpand }: Props) {
    const displayed = useGenerationProgress(tools, live);
    return (
        <div className="tool-activity" aria-label="Tool activity">
            {groupCalls(displayed).map((group) => (
                <Group key={group[0].id} tools={group} live={live} onExpand={onExpand} />
            ))}
        </div>
    );
}
