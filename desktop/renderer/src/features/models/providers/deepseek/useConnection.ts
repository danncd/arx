import { useCallback, useEffect, useRef, useState } from "react";
import type { BackendStatus } from "../../../../../../contracts/backend";
import type { Preferences } from "../../../../platform/preferences/usePreferences";
import type { Connection } from "../../../../../../contracts/wire.generated";
import { selectRun } from "../../picker/selection";

const empty: Connection = { configured: false, connected: false, models: [], error: "" };

export function useConnection(backend: BackendStatus, preferences: Preferences) {
    const [connection, setConnection] = useState(empty);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");
    const generation = useRef(0);
    const latest = useRef(preferences);
    latest.current = preferences;
    const request = useCallback(
        async (
            method:
                | "deepseek.status"
                | "deepseek.refresh"
                | "deepseek.connect"
                | "deepseek.disconnect",
            key?: string,
        ) => {
            const desktop = window.arxDesktop;
            if (!desktop) return false;
            const attempt = ++generation.current;
            setBusy(true);
            setError("");
            try {
                const result =
                    method === "deepseek.connect"
                        ? await desktop.request(method, { key: key || "" })
                        : await desktop.request(method, undefined);
                if (attempt !== generation.current) return false;
                setConnection(result);
                if (
                    result.models.length &&
                    latest.current.settings.run.provider !== "network" &&
                    !latest.current.settings.run.model.startsWith("network:") &&
                    latest.current.settings.run.provider !== "local" &&
                    !latest.current.settings.run.model.startsWith("local:")
                ) {
                    const run = selectRun(result.models, latest.current.settings.run);
                    if (
                        run.model !== latest.current.settings.run.model ||
                        run.effort !== latest.current.settings.run.effort
                    ) {
                        await latest.current.configure({ run });
                    }
                }
                return result.connected || method === "deepseek.disconnect";
            } catch (error) {
                if (attempt === generation.current)
                    setError(
                        error instanceof Error
                            ? error.message
                            : "Couldn’t update connection",
                    );
                return false;
            } finally {
                if (attempt === generation.current) setBusy(false);
            }
        },
        [],
    );
    useEffect(() => {
        if (backend.state === "ready") void request("deepseek.status");
        return () => {
            generation.current++;
        };
    }, [backend.state, request]);
    return {
        ...connection,
        busy,
        error: error || connection.error,
        refresh: () => request("deepseek.refresh"),
        connect: (key: string) => request("deepseek.connect", key),
        disconnect: () => request("deepseek.disconnect"),
    };
}

export type DeepSeekConnection = ReturnType<typeof useConnection>;
