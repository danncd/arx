import { useEffect, useState } from "react";
import { useBackend } from "../platform/backend/useBackend";
import { ConnectionError } from "../shell/ConnectionError";
import { App } from "./App";
import { loadState, type InitialState } from "./loadState";
import "../styles/theme.css";
import "../shell/shell.css";

export function Startup() {
    const backend = useBackend();
    const [initial, setInitial] = useState<InitialState>();
    const [error, setError] = useState("");
    const [attempt, setAttempt] = useState(0);
    useEffect(() => {
        if (window.arxDesktop && backend.status.state !== "ready") return;
        if (initial) return;
        let active = true;
        void loadState()
            .then((value) => {
                if (active) {
                    setInitial(value);
                    setError("");
                }
            })
            .catch(() => {
                if (active) setError("Couldn’t load saved sessions");
            });
        return () => {
            active = false;
        };
    }, [backend.status.state, initial, attempt]);
    useEffect(() => {
        if (error || backend.status.state === "error") window.arxDesktop?.signalReady();
    }, [error, backend.status.state]);
    if (initial) return <App initial={initial} />;
    if (error)
        return (
            <div className="connection-error">
                <p>{error}</p>
                <button
                    onClick={() => {
                        setError("");
                        setAttempt((value) => value + 1);
                    }}
                >
                    Retry
                </button>
            </div>
        );
    if (
        backend.status.state === "error" ||
        (window.arxDesktop && backend.status.state === "stopped")
    )
        return <ConnectionError onRetry={() => void backend.restart()} />;
    return <div className="startup-placeholder" role="status" aria-label="Loading Arx" />;
}
