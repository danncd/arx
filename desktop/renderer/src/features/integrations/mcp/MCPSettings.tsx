import { ServerDetails } from "./ServerDetails";
import { useState } from "react";
import { CaretRightIcon, PlugsIcon } from "@phosphor-icons/react";
import type { MCPConfiguration } from "../../../../../contracts/wire.generated";
import { ServerForm } from "./ServerForm";
import type { MCPState } from "./useMCPServers";
import { Back, when } from "../shared/Details";
const blank: MCPConfiguration = {
    id: "",
    name: "",
    command: "",
    arguments: [],
    environment: {},
    enabled: true,
    policy: "ask",
    readOnlyTools: [],
};

export function MCPSettings({
    state,
    running,
    onSkills,
    initialID,
}: {
    state: MCPState;
    running: boolean;
    onSkills: () => void;
    initialID?: string;
}) {
    const [selected, setSelected] = useState<string | null>(initialID ?? null),
        [adding, setAdding] = useState(false);

    const disabled = state.busy || running,
        server = state.servers.find((s) => s.id === selected);
    const back = () => {
        setSelected(null);
        setAdding(false);

        state.setError("");
    };
    if (adding)
        return (
            <>
                <Back name="MCPs" onBack={back} />
                <h3>Add MCP server</h3>
                <ServerForm
                    key="new"
                    value={blank}
                    adding
                    disabled={disabled}
                    onSave={async (c) => {
                        const saved = await state.saveServer(c);
                        if (saved) {
                            setAdding(false);
                            setSelected(c.id);
                        }
                    }}
                    onTest={(c) =>
                        state.testServer(c.id || "connection-test", {
                            ...c,
                            id: c.id || "connection-test",
                        })
                    }
                    onCancel={back}
                />
            </>
        );
    if (!server)
        return (
            <>
                <div className="integration-heading">
                    <h3>MCPs</h3>
                    <button
                        type="button"
                        className="directory-button"
                        disabled={disabled || state.loading}
                        onClick={() => setAdding(true)}
                    >
                        Add server…
                    </button>
                </div>
                <p className="network-hint">Connect tools and data to Arx.</p>
                {state.loading ? (
                    <p className="network-status">Loading MCP servers…</p>
                ) : state.servers.length ? (
                    state.servers.map((s) => (
                        <div className="provider-section" key={s.id}>
                            <button
                                type="button"
                                className="provider-toggle"
                                onClick={() => {
                                    setSelected(s.id);
                                }}
                            >
                                <span className="provider-identity">
                                    <PlugsIcon size={15} />
                                    <span>
                                        <strong>{s.name}</strong>
                                        <small>
                                            {s.tools.length} tools · {s.usage.count} calls
                                        </small>
                                    </span>
                                </span>
                                <span className="provider-tail">
                                    <span
                                        className={`connection-status ${s.error ? "integration-error" : !s.enabled ? "integration-muted" : ""}`}
                                    >
                                        {s.status}
                                    </span>
                                    <CaretRightIcon size={15} />
                                </span>
                            </button>
                            <p className="integration-row-last">
                                Last used {when(s.usage.lastUsed)}
                            </p>
                        </div>
                    ))
                ) : (
                    <p className="network-status">No MCP servers added.</p>
                )}
            </>
        );
    return (
        <ServerDetails
            key={server.id}
            state={state}
            server={server}
            disabled={disabled}
            back={back}
            onSkills={onSkills}
        />
    );
}
