import { useState } from "react";
import type { MCPServer } from "../../../../../contracts/wire.generated";
import type { MCPState } from "./useMCPServers";
import { ServerForm } from "./ServerForm";
import { Activity, Back, DetailTabs, Remove, Usage } from "../shared/Details";
const tabs = [
    { value: "configuration", label: "Configuration" },
    { value: "tools", label: "Tools" },
    { value: "activity", label: "Activity" },
] as const;
export function ServerDetails({
    state,
    server,
    disabled,
    back,
    onSkills,
}: {
    state: MCPState;
    server: MCPServer;
    disabled: boolean;
    back: () => void;
    onSkills: () => void;
}) {
    const [tab, setTab] = useState<(typeof tabs)[number]["value"]>("configuration"),
        [removing, setRemoving] = useState(false),
        [schema, setSchema] = useState("");
    return (
        <>
            <Back name="MCPs" onBack={back} />
            <div className="integration-heading">
                <h3>{server.name}</h3>
                <span
                    className={`connection-status ${server.error ? "integration-error" : !server.enabled ? "integration-muted" : ""}`}
                >
                    {server.status}
                </span>
            </div>
            <div className="general-setting">
                <div>
                    <label>Available to Arx</label>
                    <p>
                        {server.enabled
                            ? "Arx can discover and call this server’s tools."
                            : "This server is disabled."}
                    </p>
                </div>
                <button
                    type="button"
                    className="directory-button"
                    disabled={disabled}
                    onClick={() =>
                        void state.saveServer({ ...server, enabled: !server.enabled })
                    }
                >
                    {server.enabled ? "Disable" : "Enable"}
                </button>
            </div>
            <Usage usage={server.usage} kind="calls" />
            <DetailTabs values={tabs} value={tab} onChange={setTab} />
            {tab === "configuration" ? (
                <>
                    <ServerForm
                        key={
                            server.id +
                            JSON.stringify([
                                server.command,
                                server.arguments,
                                server.environment,
                                server.name,
                                server.policy,
                            ])
                        }
                        value={server}
                        disabled={disabled}
                        onSave={state.saveServer}
                        onTest={(c) => state.testServer(c.id, c)}
                    />
                    <div className="integration-actions">
                        <button
                            type="button"
                            className="text-button"
                            disabled={disabled || !server.enabled}
                            onClick={() => void state.testServer(server.id)}
                        >
                            Reconnect
                        </button>
                        <button
                            type="button"
                            className="text-button"
                            disabled={disabled}
                            onClick={() => setRemoving(!removing)}
                        >
                            {removing ? "Cancel removal" : "Remove server…"}
                        </button>
                    </div>
                    {server.error && (
                        <p className="integration-error network-hint" role="alert">
                            {server.error}
                        </p>
                    )}
                </>
            ) : tab === "tools" ? (
                <>
                    <p className="network-hint">
                        Mark tools you have reviewed as read only. Server hints alone
                        never grant access.
                    </p>
                    {server.tools.length ? (
                        server.tools.map((t) => (
                            <div className="integration-data-row" key={t.name}>
                                <div>
                                    <button
                                        className="text-button"
                                        type="button"
                                        onClick={() =>
                                            setSchema(schema === t.name ? "" : t.name)
                                        }
                                    >
                                        {t.name}
                                    </button>
                                    <small>{t.description}</small>
                                    <label className="integration-check">
                                        <input
                                            type="checkbox"
                                            checked={server.readOnlyTools.includes(
                                                t.name,
                                            )}
                                            disabled={disabled}
                                            onChange={(e) =>
                                                void state.saveServer({
                                                    ...server,
                                                    readOnlyTools: e.target.checked
                                                        ? [
                                                              ...server.readOnlyTools,
                                                              t.name,
                                                          ]
                                                        : server.readOnlyTools.filter(
                                                              (n) => n !== t.name,
                                                          ),
                                                })
                                            }
                                        />
                                        Allow as read only
                                    </label>
                                    {t.readOnlyHint && (
                                        <small>Server suggests read only</small>
                                    )}
                                    {schema === t.name && (
                                        <pre className="integration-code">
                                            {JSON.stringify(t.inputSchema, null, 2)}
                                        </pre>
                                    )}
                                </div>
                                <span className="integration-tail">{t.calls} calls</span>
                            </div>
                        ))
                    ) : (
                        <p className="network-status">
                            Connect this server to discover its tools.
                        </p>
                    )}
                </>
            ) : (
                <Activity usage={server.usage} kind="mcp" />
            )}
            {removing && (
                <Remove
                    name={server.name}
                    description="Disconnect this server and remove its configuration. Related skills will show a missing dependency."
                    busy={disabled}
                    onRemove={() =>
                        void state.removeServer(server.id).then((result) => {
                            if (result) back();
                        })
                    }
                />
            )}
            <h4 className="settings-section-title">Skills</h4>
            <button type="button" className="text-button" onClick={onSkills}>
                Manage skills using this server
            </button>
        </>
    );
}
