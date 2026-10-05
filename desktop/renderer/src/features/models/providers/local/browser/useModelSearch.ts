import { useEffect, useRef, useState } from "react";
import type { SearchModel } from "../../../../../../../contracts/wire.generated";

export function useModelSearch(query: string, sort: string) {
    const [models, setModels] = useState<SearchModel[]>([]);
    const [busy, setBusy] = useState(true);
    const [moreBusy, setMoreBusy] = useState(false);
    const [next, setNext] = useState("");
    const [error, setError] = useState("");
    const generation = useRef(0);
    const loading = useRef(false);
    useEffect(() => {
        const version = ++generation.current;
        setBusy(true);
        setMoreBusy(false);
        setModels([]);
        setNext("");
        setError("");
        loading.current = false;
        const timer = setTimeout(() => {
            void window.arxDesktop
                ?.request("local.search", { query, sort })
                .then((result) => {
                    if (version !== generation.current) return;
                    setModels(result.models);
                    setNext(result.next);
                })
                .catch((error) => {
                    if (version === generation.current) setError(error.message);
                })
                .finally(() => {
                    if (version === generation.current) setBusy(false);
                });
        }, 300);
        return () => {
            ++generation.current;
            clearTimeout(timer);
        };
    }, [query, sort]);
    const loadMore = async () => {
        if (!next || busy || loading.current) return;
        const version = generation.current;
        loading.current = true;
        setMoreBusy(true);
        setError("");
        try {
            const result = await window.arxDesktop?.request("local.search", {
                query,
                sort,
                cursor: next,
            });
            if (!result || version !== generation.current) return;
            setModels((previous) => [
                ...new Map(
                    [...previous, ...result.models].map((model) => [model.id, model]),
                ).values(),
            ]);
            setNext(result.next === next ? "" : result.next);
        } catch (error) {
            if (version === generation.current)
                setError(
                    error instanceof Error ? error.message : "Could not load models",
                );
        } finally {
            if (version === generation.current) {
                loading.current = false;
                setMoreBusy(false);
            }
        }
    };
    return { models, busy, moreBusy, error, next, loadMore };
}
