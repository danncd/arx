import { useState } from "react";
import { Select } from "../../../ui/select/Select";
import type { MCPConfiguration } from "../../../../../contracts/wire.generated";
const policyChoices = [
    { value: "ask", label: "Ask for every call" },
    { value: "changes", label: "Ask before changes" },
] as const;
function configOnly(c: MCPConfiguration): MCPConfiguration {
    return {
        id: c.id,
        name: c.name,
        command: c.command,
        arguments: c.arguments,
        environment: c.environment,
        enabled: c.enabled,
        policy: c.policy,
        readOnlyTools: c.readOnlyTools,
    };
}
export function ServerForm({
    value,
    adding = false,
    disabled,
    onSave,
    onTest,
    onCancel,
}: {
    value: MCPConfiguration;
    adding?: boolean;
    disabled: boolean;
    onSave: (c: MCPConfiguration) => Promise<unknown>;
    onTest: (c: MCPConfiguration) => Promise<unknown>;
    onCancel?: () => void;
}) {
    const [draft, setDraft] = useState(configOnly(value)),
        [args, setArgs] = useState(value.arguments.join("\n")),
        [env, setEnv] = useState(
            Object.entries(value.environment)
                .map(([k, v]) => `${k}=${v}`)
                .join("\n"),
        ),
        [message, setMessage] = useState(""),
        [localError, setLocalError] = useState("");
    const put = (name: "id" | "name" | "command", v: string) =>
        setDraft({ ...draft, [name]: v });
    const configuration = () => {
        const environment: Record<string, string> = {};
        for (const line of env.split("\n").filter((s) => s.trim())) {
            const at = line.indexOf("=");
            if (at < 1) throw new Error("Environment variables must use KEY=value.");
            environment[line.slice(0, at).trim()] = line.slice(at + 1);
        }
        return {
            ...draft,
            arguments: args.split("\n").filter((v) => v !== ""),
            environment,
        };
    };
    const run = async (test: boolean) => {
        setMessage("");
        setLocalError("");
        try {
            const c = configuration(),
                result = await (test ? onTest(c) : onSave(c));
            if (result)
                setMessage(test ? "Connection test passed." : "Configuration saved.");
        } catch (e) {
            setLocalError(
                e instanceof Error ? e.message : "Could not save configuration",
            );
        }
    };
    return (
        <div className="network-content">
            <div className="network-form">
                <label>
                    Name
                    <input
                        value={draft.name}
                        onChange={(e) => put("name", e.target.value)}
                        disabled={disabled}
                    />
                </label>
                <label>
                    Server ID
                    {adding ? (
                        <input
                            value={draft.id}
                            onChange={(e) => put("id", e.target.value)}
                            disabled={disabled}
                        />
                    ) : (
                        <span>{draft.id}</span>
                    )}
                </label>
                <label>
                    Connection<span>Local process · stdio</span>
                </label>
                <label>
                    Executable
                    <input
                        value={draft.command}
                        onChange={(e) => put("command", e.target.value)}
                        disabled={disabled}
                        spellCheck={false}
                    />
                </label>
                <label>
                    Arguments · one per line
                    <textarea
                        rows={3}
                        value={args}
                        onChange={(e) => setArgs(e.target.value)}
                        disabled={disabled}
                        spellCheck={false}
                    />
                </label>
                <label>
                    Environment · KEY=value, one per line
                    <textarea
                        rows={3}
                        value={env}
                        onChange={(e) => setEnv(e.target.value)}
                        disabled={disabled}
                        spellCheck={false}
                    />
                </label>
                <div className="integration-select-label">
                    Call permissions
                    <Select
                        label="Call permissions"
                        value={draft.policy}
                        choices={policyChoices}
                        disabled={disabled}
                        onChange={(policy) => setDraft({ ...draft, policy })}
                    />
                </div>
            </div>
            {localError && (
                <p className="network-hint integration-error" role="alert">
                    {localError}
                </p>
            )}
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
                    onClick={() => void run(false)}
                >
                    {adding ? "Add server" : "Save changes"}
                </button>
                <button
                    type="button"
                    className="directory-button"
                    disabled={disabled}
                    onClick={() => void run(true)}
                >
                    Test connection
                </button>
                {onCancel && (
                    <button type="button" className="text-button" onClick={onCancel}>
                        Cancel
                    </button>
                )}
            </div>
        </div>
    );
}
