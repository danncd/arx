import { useState } from "react";
import {
    CaretRightIcon,
    CheckIcon,
    GlobeIcon,
    MagnifyingGlassIcon,
    PlusIcon,
    SpinnerGapIcon,
} from "@phosphor-icons/react";
import type { NetworkModelsState } from "./useNetworkModels";
import { ServerForm } from "./ServerForm";
import { ServerRow } from "./ServerRow";
import "./network.css";
export function NetworkModels({ network }: { network: NetworkModelsState }) {
    const [expanded, setExpanded] = useState(false);
    const [form, setForm] = useState<{ url: string; name: string } | null>(null);
    return (
        <div className="provider-section">
            <button
                type="button"
                className="provider-toggle"
                aria-expanded={expanded}
                onClick={() => {
                    setExpanded(!expanded);
                    if (!expanded) void network.refresh().catch(() => {});
                }}
            >
                <span className="provider-identity">
                    <GlobeIcon className="provider-icon" />
                    <strong>Network models</strong>
                </span>
                <span className="provider-tail">
                    {network.servers.some((server) => server.connected) && (
                        <span className="connection-status">
                            <CheckIcon size={12} />
                            Connected
                        </span>
                    )}
                    <CaretRightIcon className="provider-chevron" size={12} />
                </span>
            </button>
            <div className={`provider-body${expanded ? " expanded" : ""}`}>
                <div>
                    <div className="network-content">
                        <p className="network-hint">
                            Use models running on another computer.
                        </p>
                        <div className="network-actions">
                            <button
                                type="button"
                                className="local-button"
                                disabled={network.busy}
                                onClick={() =>
                                    network.scanning
                                        ? network.cancel()
                                        : void network.scan()
                                }
                            >
                                {network.scanning ? (
                                    <SpinnerGapIcon
                                        className="network-spinner"
                                        size={14}
                                    />
                                ) : (
                                    <MagnifyingGlassIcon size={14} />
                                )}
                                {network.scanning ? "Cancel scan" : "Scan network"}
                            </button>
                            <button
                                type="button"
                                className="local-button"
                                onClick={() => setForm({ url: "", name: "My PC" })}
                            >
                                <PlusIcon size={14} />
                                Add manually
                            </button>
                        </div>
                        {form && (
                            <ServerForm
                                key={form.url}
                                address={form.url}
                                name={form.name}
                                busy={network.busy}
                                onConnect={network.connect}
                                onCancel={() => setForm(null)}
                            />
                        )}
                        {network.error && (
                            <p className="network-hint" role="alert">
                                {network.error}
                            </p>
                        )}
                        {network.scanning && (
                            <p className="network-status" role="status">
                                Looking for model servers…
                            </p>
                        )}
                        {network.servers.map((server) => (
                            <ServerRow
                                key={server.id}
                                server={server}
                                saved
                                busy={network.busy}
                                onConnect={() => setForm(server)}
                                onRemove={() => void network.remove(server.id)}
                            />
                        ))}
                        {network.found.map((server) => (
                            <ServerRow
                                key={server.id}
                                server={server}
                                saved={false}
                                busy={network.busy}
                                onConnect={() =>
                                    server.requiresToken
                                        ? setForm(server)
                                        : void network.connect(server.url, server.name)
                                }
                                onRemove={() => {}}
                            />
                        ))}
                        {!network.servers.length &&
                            !network.found.length &&
                            !network.scanning && (
                                <p className="network-status">
                                    {network.searched
                                        ? "No servers found. Enable Serve on Local Network in LM Studio, or add an address manually."
                                        : "No servers connected."}
                                </p>
                            )}
                    </div>
                </div>
            </div>
        </div>
    );
}
