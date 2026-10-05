import { useState } from "react";
export function ServerForm({
    address = "",
    name = "My PC",
    busy,
    onConnect,
    onCancel,
}: {
    address?: string;
    name?: string;
    busy: boolean;
    onConnect: (url: string, name: string, token: string) => Promise<boolean>;
    onCancel: () => void;
}) {
    const [url, setURL] = useState(address);
    const [label, setLabel] = useState(name);
    const [token, setToken] = useState("");
    return (
        <form
            className="network-form"
            onSubmit={async (event) => {
                event.preventDefault();
                if (await onConnect(url, label, token)) onCancel();
            }}
        >
            <label>
                Server name
                <input
                    required
                    maxLength={80}
                    value={label}
                    onChange={(event) => setLabel(event.target.value)}
                />
            </label>
            <label>
                Server address
                <input
                    required
                    type="url"
                    placeholder="http://192.168.1.50:1234/v1"
                    value={url}
                    onChange={(event) => setURL(event.target.value)}
                />
            </label>
            <label>
                API token (optional)
                <input
                    type="password"
                    autoComplete="off"
                    value={token}
                    onChange={(event) => setToken(event.target.value)}
                />
            </label>
            <div className="network-actions">
                <button className="local-button" disabled={busy}>
                    Connect
                </button>
                <button className="text-button" type="button" onClick={onCancel}>
                    Cancel
                </button>
            </div>
        </form>
    );
}
