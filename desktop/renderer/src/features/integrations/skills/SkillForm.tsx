import { useState } from "react";
import { Select } from "../../../ui/select/Select";
import type { SkillDetail, SkillEdit } from "../../../../../contracts/wire.generated";
const activations = [
    { value: "auto", label: "Automatic and explicit" },
    { value: "explicit", label: "Explicit only" },
] as const;
const initial =
    "---\nname: my-skill\ndescription: Describe when Arx should use this skill.\n---\n\n# Instructions\n\nDescribe the workflow.\n";
export function SkillForm({
    value,
    adding = false,
    disabled,
    servers,
    onSave,
    onValidate,
}: {
    value?: SkillDetail;
    adding?: boolean;
    disabled: boolean;
    servers: string[];
    onSave: (e: SkillEdit) => Promise<unknown>;
    onValidate: (body: string) => Promise<unknown>;
}) {
    const [instructions, setInstructions] = useState(value?.instructions ?? initial),
        [id, setID] = useState(value?.id ?? ""),
        [activation, setActivation] = useState(value?.activation ?? "auto"),
        [dependencies, setDependencies] = useState(value?.dependencies.join("\n") ?? ""),
        [message, setMessage] = useState("");
    const edit = (): SkillEdit => ({
        id,
        path: value?.path ?? "",
        instructions,
        enabled: value?.enabled ?? true,
        activation,
        dependencies: dependencies
            .split("\n")
            .map((s) => s.trim())
            .filter(Boolean),
    });
    return (
        <div className="network-content">
            <div className="network-form">
                {adding && (
                    <label>
                        Skill ID
                        <input
                            value={id}
                            onChange={(e) => setID(e.target.value)}
                            disabled={disabled}
                            spellCheck={false}
                        />
                    </label>
                )}
                <div className="integration-select-label">
                    Activation
                    <Select
                        label="Activation"
                        value={activation}
                        choices={activations}
                        disabled={disabled}
                        onChange={setActivation}
                    />
                </div>
                <label>
                    SKILL.md
                    <textarea
                        aria-label="SKILL.md"
                        className="integration-instructions"
                        rows={14}
                        value={instructions}
                        onChange={(e) => {
                            setInstructions(e.target.value);
                            setMessage("");
                        }}
                        disabled={disabled}
                        spellCheck={false}
                    />
                    <span>
                        Edit the name and description in the front matter. Invoke
                        explicitly with $skill-name.
                    </span>
                </label>
                <label>
                    MCP dependencies · server IDs, one per line
                    <textarea
                        rows={2}
                        value={dependencies}
                        onChange={(e) => setDependencies(e.target.value)}
                        disabled={disabled}
                        spellCheck={false}
                    />
                    <span>
                        Available: {servers.join(", ") || "No servers configured"}
                    </span>
                </label>
            </div>
            {message && (
                <p className="connection-message" role="status">
                    {message}
                </p>
            )}
            <div className="integration-actions">
                <button
                    type="button"
                    className="primary-button"
                    disabled={disabled}
                    onClick={() => {
                        setMessage("");
                        void onSave(edit()).then((result) => {
                            if (result) setMessage("Skill saved.");
                        });
                    }}
                >
                    {adding ? "Create skill" : "Save changes"}
                </button>
                <button
                    type="button"
                    className="directory-button"
                    disabled={disabled}
                    onClick={() => {
                        setMessage("");
                        void onValidate(instructions).then((result) => {
                            if (result)
                                setMessage(
                                    "Valid skill. Metadata and instructions found.",
                                );
                        });
                    }}
                >
                    Validate
                </button>
            </div>
        </div>
    );
}
