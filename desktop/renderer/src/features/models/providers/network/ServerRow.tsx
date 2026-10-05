import { CheckIcon, DesktopTowerIcon, XIcon } from "@phosphor-icons/react";
import type { NetworkServer } from "../../../../../../contracts/wire.generated";
export function ServerRow({
    server,
    saved,
    busy,
    onConnect,
    onRemove,
}: {
    server: NetworkServer;
    saved: boolean;
    busy: boolean;
    onConnect: () => void;
    onRemove: () => void;
}) {
    return (
        <div className="network-server">
            <div className="network-server-top">
                <DesktopTowerIcon size={19} />
                <div className="network-server-name">
                    <strong>{server.name}</strong>
                    <span>{server.url}</span>
                </div>
                {saved ? (
                    <>
                        <span
                            className={server.connected ? "connection-status" : "muted"}
                        >
                            {server.connected && <CheckIcon size={12} />}
                            {server.connected ? "Connected" : "Offline"}
                        </span>
                        <button
                            type="button"
                            className="local-button local-import"
                            aria-label={`Disconnect ${server.name}`}
                            disabled={busy}
                            onClick={onRemove}
                        >
                            <XIcon size={13} />
                        </button>
                    </>
                ) : (
                    <button
                        type="button"
                        className="local-button"
                        disabled={busy}
                        onClick={onConnect}
                    >
                        {server.requiresToken ? "Enter token" : "Connect"}
                    </button>
                )}
            </div>
            {server.error && (
                <p className="network-hint" role="status">
                    {server.error}
                </p>
            )}
            {server.connected && saved && (
                <div className="network-models">
                    <span>Models · load in LM Studio to use in chat</span>
                    {server.models.map(({ info, loaded }) => (
                        <div key={info.id}>
                            <span>{info.name}</span>
                            <small>
                                {loaded ? "Loaded · " : ""}
                                {info.vision ? "Vision · " : ""}
                                {info.tools ? "Tools" : "Text"}
                            </small>
                        </div>
                    ))}
                    {!server.models.length && (
                        <p className="network-hint">No chat models available.</p>
                    )}
                </div>
            )}
        </div>
    );
}
