import { useState } from "react";
import type { SkillDetail, MCPServer } from "../../../../../contracts/wire.generated";
import type { SkillsState } from "./useSkills";
import { SkillForm } from "./SkillForm";
import { Activity, Back, DetailTabs, Remove, Usage } from "../shared/Details";
const tabs = [
    { value: "instructions", label: "Instructions" },
    { value: "files", label: "Files" },
    { value: "activity", label: "Activity" },
] as const;
export function SkillDetails({
    state,
    current,
    disabled,
    back,
    onMCP,
    choose,
    open,
}: {
    state: SkillsState & { servers: MCPServer[] };
    current: SkillDetail;
    disabled: boolean;
    back: () => void;
    onMCP: (id: string) => void;
    choose: (change?: boolean) => Promise<void>;
    open: (id: string) => Promise<void>;
}) {
    const [tab, setTab] = useState<(typeof tabs)[number]["value"]>("instructions"),
        [removing, setRemoving] = useState(false),
        [reference, setReference] = useState<{ path: string; body: string } | null>(null);
    return (
        <>
            <Back name="Skills" onBack={back} />
            <div className="integration-heading">
                <h3>{current.name}</h3>
                <span
                    className={`connection-status ${current.enabled ? "" : "integration-muted"}`}
                >
                    {current.enabled ? "Enabled" : "Disabled"}
                </span>
            </div>
            <div className="general-setting">
                <div>
                    <label>Available to Arx</label>
                    <p>{current.description}</p>
                </div>
                <button
                    type="button"
                    className="directory-button"
                    disabled={disabled}
                    onClick={() =>
                        void state.saveSkill({ ...current, enabled: !current.enabled })
                    }
                >
                    {current.enabled ? "Disable" : "Enable"}
                </button>
            </div>
            <Usage usage={current.usage} kind="times used" />
            <DetailTabs values={tabs} value={tab} onChange={setTab} />
            {tab === "instructions" ? (
                <>
                    <SkillForm
                        key={current.id + current.instructions}
                        value={current}
                        disabled={disabled}
                        servers={state.servers.map((s) => s.id)}
                        onValidate={state.validate}
                        onSave={async (e) => {
                            const result = await state.saveSkill(e);
                            if (result) await open(e.id);
                            return result;
                        }}
                    />
                    <div className="integration-actions">
                        <button
                            type="button"
                            className="text-button"
                            disabled={disabled}
                            onClick={() => setRemoving(!removing)}
                        >
                            {removing ? "Cancel removal" : "Remove skill…"}
                        </button>
                    </div>
                </>
            ) : tab === "files" ? (
                <>
                    <h4 className="settings-section-title">Location</h4>
                    <div className="general-setting">
                        <div className="directory-value">
                            <p>{current.path}</p>
                        </div>
                        <button
                            type="button"
                            className="directory-button"
                            disabled={disabled}
                            onClick={() => void choose(true)}
                        >
                            Change folder…
                        </button>
                    </div>
                    <h4 className="settings-section-title">Files</h4>
                    {current.files.map((path) => (
                        <div className="integration-data-row" key={path}>
                            <div>
                                <strong>{path}</strong>
                                <small>
                                    {path === "SKILL.md"
                                        ? "Instructions and metadata"
                                        : "Supporting file"}
                                </small>
                            </div>
                            <button
                                type="button"
                                className="text-button"
                                disabled={state.busy}
                                onClick={() =>
                                    path === "SKILL.md"
                                        ? setTab("instructions")
                                        : void state
                                              .read(current.id, path)
                                              .then((body) => {
                                                  if (body !== undefined)
                                                      setReference({ path, body });
                                              })
                                }
                            >
                                {path === "SKILL.md" ? "Edit" : "View"}
                            </button>
                        </div>
                    ))}
                    {reference && (
                        <>
                            <h4 className="settings-section-title">{reference.path}</h4>
                            <pre className="integration-code">{reference.body}</pre>
                        </>
                    )}
                    <h4 className="settings-section-title">MCP dependencies</h4>
                    {current.dependencies.length ? (
                        current.dependencies.map((id) => {
                            const server = state.servers.find((s) => s.id === id);
                            return (
                                <div className="general-setting" key={id}>
                                    <div>
                                        <label>{server?.name ?? id}</label>
                                        <p
                                            className={
                                                !server || !server.enabled
                                                    ? "integration-error"
                                                    : ""
                                            }
                                        >
                                            {!server
                                                ? "Required server is missing"
                                                : !server.enabled
                                                  ? "Required server is disabled"
                                                  : server.status}
                                        </p>
                                    </div>
                                    <button
                                        type="button"
                                        className="directory-button"
                                        onClick={() => onMCP(id)}
                                    >
                                        Open MCP settings
                                    </button>
                                </div>
                            );
                        })
                    ) : (
                        <p className="permission-description">No MCP server required.</p>
                    )}
                </>
            ) : (
                <Activity usage={current.usage} kind="skill" />
            )}
            {removing && (
                <Remove
                    name={current.name}
                    description="Remove this skill from Arx. Source files will be kept."
                    busy={disabled}
                    onRemove={() =>
                        void state.removeSkill(current.id).then((result) => {
                            if (result) back();
                        })
                    }
                />
            )}
        </>
    );
}
