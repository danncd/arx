import { SkillDetails } from "./SkillDetails";
import { useState } from "react";
import { BookOpenIcon, CaretRightIcon } from "@phosphor-icons/react";
import type { SkillDetail } from "../../../../../contracts/wire.generated";
import { SkillForm } from "./SkillForm";
import type { SkillsState } from "./useSkills";
import type { MCPServer } from "../../../../../contracts/wire.generated";
import { Back, when } from "../shared/Details";
export function SkillsSettings({
    state,
    running,
    onMCP,
}: {
    state: SkillsState & { servers: MCPServer[] };
    running: boolean;
    onMCP: (id: string) => void;
}) {
    const [detail, setDetail] = useState<SkillDetail | null>(null),
        [adding, setAdding] = useState(false),
        [revision, setRevision] = useState(0);
    const disabled = state.busy || running,
        back = () => {
            setDetail(null);
            setAdding(false);

            state.setError("");
        };
    const saved = detail && state.skills.find((s) => s.id === detail.id),
        current = detail && saved ? { ...detail, ...saved } : detail;
    const open = async (id: string) => {
        const result = await state.detail(id);
        if (result) {
            setDetail(result);

            setRevision((value) => value + 1);
        }
    };
    const choose = async (change = false) => {
        if (!window.arxDesktop) return;
        try {
            const path = await window.arxDesktop.chooseDirectory();
            if (!path) return;
            const id = change && current ? current.id : crypto.randomUUID();
            const result = await state.saveSkill({
                id,
                path,
                instructions: "",
                enabled: current?.enabled ?? true,
                activation: current?.activation ?? "auto",
                dependencies: current?.dependencies ?? [],
            });
            if (result) {
                await open(id);
                setAdding(false);
            }
        } catch (e) {
            state.setError(e instanceof Error ? e.message : "Could not import skill");
        }
    };
    if (adding)
        return (
            <>
                <Back name="Skills" onBack={back} />
                <h3>Add skill</h3>
                <div className="general-setting">
                    <div>
                        <label>Import an existing skill</label>
                        <p>Choose a folder containing SKILL.md.</p>
                    </div>
                    <button
                        type="button"
                        className="directory-button"
                        disabled={disabled}
                        onClick={() => void choose()}
                    >
                        Choose folder…
                    </button>
                </div>
                <h4 className="settings-section-title">Create a skill</h4>
                <SkillForm
                    key="new"
                    adding
                    disabled={disabled}
                    servers={state.servers.map((s) => s.id)}
                    onValidate={state.validate}
                    onSave={async (e) => {
                        const result = await state.saveSkill(e);
                        if (result) {
                            await open(e.id);
                            setAdding(false);
                        }
                        return result;
                    }}
                />
            </>
        );
    if (!current)
        return (
            <>
                <div className="integration-heading">
                    <h3>Skills</h3>
                    <button
                        type="button"
                        className="directory-button"
                        disabled={disabled || state.loading}
                        onClick={() => setAdding(true)}
                    >
                        Add skill…
                    </button>
                </div>
                <p className="network-hint">
                    Reusable instructions Arx can load when needed.
                </p>
                {state.loading ? (
                    <p className="network-status">Loading skills…</p>
                ) : state.skills.length ? (
                    state.skills.map((s) => (
                        <div className="provider-section" key={s.id}>
                            <button
                                type="button"
                                className="provider-toggle"
                                disabled={state.busy}
                                onClick={() => void open(s.id)}
                            >
                                <span className="provider-identity">
                                    <BookOpenIcon size={15} />
                                    <span>
                                        <strong>{s.name}</strong>
                                        <small>
                                            {s.usage.count} times used · {s.description}
                                        </small>
                                    </span>
                                </span>
                                <span className="provider-tail">
                                    <span
                                        className={`connection-status ${s.error ? "integration-error" : !s.enabled ? "integration-muted" : ""}`}
                                    >
                                        {s.error
                                            ? "Needs attention"
                                            : s.enabled
                                              ? "Enabled"
                                              : "Disabled"}
                                    </span>
                                    <CaretRightIcon size={15} />
                                </span>
                            </button>
                            <p className="integration-row-last">
                                Last used {when(s.usage.lastUsed)}
                            </p>
                            {s.error && (
                                <>
                                    <p className="network-hint integration-error">
                                        {s.error}
                                    </p>
                                    <button
                                        type="button"
                                        className="text-button"
                                        disabled={disabled}
                                        onClick={() => void state.removeSkill(s.id)}
                                    >
                                        Remove registration
                                    </button>
                                </>
                            )}
                        </div>
                    ))
                ) : (
                    <p className="network-status">No skills added.</p>
                )}
            </>
        );
    return (
        <SkillDetails
            key={current.id + ":" + revision}
            state={state}
            current={current}
            disabled={disabled}
            back={back}
            onMCP={onMCP}
            choose={choose}
            open={open}
        />
    );
}
