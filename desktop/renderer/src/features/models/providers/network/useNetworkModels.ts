import { useCallback, useEffect, useRef, useState } from "react";
import type { BackendStatus } from "../../../../../../contracts/backend";
import type { NetworkServer } from "../../../../../../contracts/wire.generated";

export function useNetworkModels(backend: BackendStatus) {
    const [servers, setServers] = useState<NetworkServer[]>([]);
    const [found, setFound] = useState<NetworkServer[]>([]);
    const [scanning, setScanning] = useState(false);
    const [busy, setBusy] = useState(false);
    const [searched, setSearched] = useState(false);
    const [error, setError] = useState("");
    const scanID = useRef(0);
    const refreshing = useRef(false);
    const revision = useRef(0);
    const refresh = useCallback(async () => {
        if (refreshing.current || !window.arxDesktop) return;
        refreshing.current = true;
        const version = revision.current;
        try {
            const result = await window.arxDesktop.request("network.state", undefined);
            if (version === revision.current) setServers(result);
        } finally {
            refreshing.current = false;
        }
    }, []);
    useEffect(() => {
        if (backend.state !== "ready") return;
        void refresh().catch(() => {});
        const timer = setInterval(() => void refresh().catch(() => {}), 30000);
        return () => clearInterval(timer);
    }, [backend.state, refresh]);
    const connect = async (url: string, name: string, token = "") => {
        if (!window.arxDesktop) return false;
        revision.current++;
        setBusy(true);
        setError("");
        try {
            const server = await window.arxDesktop.request("network.connect", {
                url,
                name,
                token,
            });
            setServers((previous) => [
                ...previous.filter((item) => item.id !== server.id),
                server,
            ]);
            setFound((previous) => previous.filter((item) => item.id !== server.id));
            return true;
        } catch (error) {
            setError(error instanceof Error ? error.message : "Could not connect");
            return false;
        } finally {
            revision.current++;
            setBusy(false);
        }
    };
    const scan = async () => {
        if (!window.arxDesktop) return;
        const id = ++scanID.current;
        setScanning(true);
        setError("");
        setFound([]);
        setSearched(false);
        try {
            const result = await window.arxDesktop.request("network.scan", undefined);
            if (id !== scanID.current) return;
            const fresh = result.filter(
                (item) => !servers.some((saved) => saved.id === item.id),
            );
            setFound(fresh);
            setSearched(true);
            const only = fresh[0];
            if (fresh.length === 1 && only?.connected && !only.requiresToken)
                await connect(only.url, only.name);
            const version = ++revision.current;
            const updated = await window.arxDesktop.request("network.state", undefined);
            if (id === scanID.current && version === revision.current)
                setServers(updated);
        } catch (error) {
            if (id === scanID.current)
                setError(error instanceof Error ? error.message : "Scan failed");
        } finally {
            if (id === scanID.current) setScanning(false);
        }
    };
    const cancel = () => {
        scanID.current++;
        setScanning(false);
        void window.arxDesktop?.request("network.cancel", undefined).catch(() => {});
    };
    const remove = async (id: string) => {
        revision.current++;
        setBusy(true);
        setError("");
        try {
            await window.arxDesktop?.request("network.remove", { id });
            setServers((previous) => previous.filter((item) => item.id !== id));
        } catch (error) {
            setError(error instanceof Error ? error.message : "Could not disconnect");
        } finally {
            revision.current++;
            setBusy(false);
        }
    };
    return {
        servers,
        found,
        scanning,
        busy,
        searched,
        error,
        refresh,
        connect,
        scan,
        cancel,
        remove,
    };
}
export type NetworkModelsState = ReturnType<typeof useNetworkModels>;
