import { useCallback, useEffect, useRef, useState } from "react";

export function useRequests<T>(load: () => Promise<T>, empty: T) {
    const [data, setData] = useState(empty);
    const [busy, setBusy] = useState(false),
        [loading, setLoading] = useState(true),
        [error, setError] = useState("");
    const mounted = useRef(false),
        revision = useRef(0),
        active = useRef(false);
    const refresh = useCallback(async () => {
        const version = ++revision.current;
        try {
            const next = await load();
            if (mounted.current && version === revision.current) setData(next);
        } catch (e) {
            if (mounted.current && version === revision.current)
                setError(e instanceof Error ? e.message : "Could not load integrations");
        } finally {
            if (mounted.current && version === revision.current) setLoading(false);
        }
    }, [load]);
    useEffect(() => {
        mounted.current = true;
        void refresh();
        return () => {
            mounted.current = false;
            revision.current++;
        };
    }, [refresh]);
    async function act<R>(work: () => Promise<R>, reload = true): Promise<R | undefined> {
        if (active.current) return;
        active.current = true;
        revision.current++;
        setBusy(true);
        setError("");
        try {
            const result = await work();
            return mounted.current ? result : undefined;
        } catch (e) {
            if (mounted.current)
                setError(e instanceof Error ? e.message : "Integration request failed");
        } finally {
            active.current = false;
            if (mounted.current) {
                setBusy(false);
                if (reload) await refresh();
                else setLoading(false);
            }
        }
    }
    return { data, busy, loading, error, setError, refresh, act };
}
export function api() {
    if (!window.arxDesktop) throw new Error("Desktop backend is unavailable");
    return window.arxDesktop;
}
