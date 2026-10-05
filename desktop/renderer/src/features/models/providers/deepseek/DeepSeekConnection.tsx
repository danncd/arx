import { CaretRightIcon, CheckIcon, EyeIcon, EyeSlashIcon } from "@phosphor-icons/react";
import { useId, useState } from "react";
import type { DeepSeekConnection } from "./useConnection";
import "../providers.css";

const icon = new URL("./deepseek.svg", import.meta.url).href;

export function DeepSeekConnectionSection({
    connection,
}: {
    connection: DeepSeekConnection;
}) {
    const [key, setKey] = useState("");
    const [editing, setEditing] = useState(false);
    const [expanded, setExpanded] = useState(false);
    const [revealed, setRevealed] = useState(false);
    const id = useId();
    const save = async () => {
        if (await connection.connect(key)) {
            setKey("");
            setEditing(false);
            setRevealed(false);
        }
    };
    return (
        <>
            <div className="provider-section" role="group" aria-label="DeepSeek">
                <h4 className="provider-heading">
                    <button
                        type="button"
                        className="provider-toggle"
                        id={`${id}-toggle`}
                        aria-label="DeepSeek"
                        aria-expanded={expanded}
                        aria-controls={`${id}-body`}
                        onClick={() => setExpanded(!expanded)}
                    >
                        <span className="provider-identity">
                            <img className="provider-icon" src={icon} alt="" />
                            <strong>DeepSeek</strong>
                        </span>
                        <span className="provider-tail">
                            {(connection.busy || connection.configured) && (
                                <span className="connection-status">
                                    {connection.connected && !connection.busy && (
                                        <CheckIcon size={12} />
                                    )}
                                    {connection.busy
                                        ? "Checking…"
                                        : connection.connected
                                          ? "Connected"
                                          : "Key saved"}
                                </span>
                            )}
                            <CaretRightIcon className="provider-chevron" size={12} />
                        </span>
                    </button>
                </h4>
                <div
                    className={`provider-body${expanded ? " expanded" : ""}`}
                    id={`${id}-body`}
                    role="region"
                    aria-labelledby={`${id}-toggle`}
                    aria-hidden={!expanded}
                    inert={!expanded}
                >
                    <div>
                        <div className="provider-content">
                            <div className="provider-key-row">
                                {!connection.configured || editing ? (
                                    <>
                                        <div className="provider-key-input">
                                            <input
                                                aria-label="DeepSeek API key"
                                                placeholder="API key"
                                                type={revealed ? "text" : "password"}
                                                autoComplete="off"
                                                spellCheck={false}
                                                value={key}
                                                disabled={connection.busy}
                                                onChange={(event) =>
                                                    setKey(event.target.value)
                                                }
                                                onKeyDown={(event) => {
                                                    if (event.key === "Enter") {
                                                        event.preventDefault();
                                                        if (
                                                            key.trim() &&
                                                            !connection.busy
                                                        )
                                                            void save();
                                                    }
                                                }}
                                            />
                                            <button
                                                className="provider-reveal"
                                                type="button"
                                                aria-label={
                                                    revealed ? "Hide key" : "Show key"
                                                }
                                                onClick={() => setRevealed(!revealed)}
                                            >
                                                {revealed ? (
                                                    <EyeSlashIcon size={14} />
                                                ) : (
                                                    <EyeIcon size={14} />
                                                )}
                                            </button>
                                        </div>
                                        <button
                                            className="provider-connect"
                                            type="button"
                                            disabled={!key.trim() || connection.busy}
                                            onClick={() => void save()}
                                        >
                                            {connection.busy ? "Connecting…" : "Connect"}
                                        </button>
                                    </>
                                ) : (
                                    <>
                                        <span
                                            className="saved-key"
                                            aria-label="API key saved"
                                        >
                                            ••••••••••••••••
                                        </span>
                                        <button
                                            className="provider-disconnect"
                                            type="button"
                                            disabled={connection.busy}
                                            onClick={() => void connection.disconnect()}
                                        >
                                            Disconnect
                                        </button>
                                    </>
                                )}
                            </div>
                            {connection.configured && (
                                <div className="connection-actions">
                                    <button
                                        type="button"
                                        disabled={connection.busy}
                                        onClick={() => void connection.refresh()}
                                    >
                                        Refresh
                                    </button>
                                    <button
                                        type="button"
                                        disabled={connection.busy}
                                        onClick={() => {
                                            setEditing(!editing);
                                            setKey("");
                                            setRevealed(false);
                                        }}
                                    >
                                        {editing ? "Cancel" : "Change key"}
                                    </button>
                                </div>
                            )}
                            {connection.error && (
                                <p className="connection-message" role="alert">
                                    {connection.error}
                                </p>
                            )}
                            {connection.models.length > 0 && (
                                <ul className="connection-models">
                                    {connection.models.map((model) => (
                                        <li key={model.id}>
                                            <span>{model.name}</span>
                                            <small>
                                                {model.contextWindow
                                                    ? `${model.contextWindow.toLocaleString()} context`
                                                    : "Context unavailable"}
                                                {model.maxOutputTokens
                                                    ? ` · ${model.maxOutputTokens.toLocaleString()} output`
                                                    : ""}
                                            </small>
                                        </li>
                                    ))}
                                </ul>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
}
