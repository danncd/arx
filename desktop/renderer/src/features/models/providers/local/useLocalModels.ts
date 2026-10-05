import { useCallback, useEffect, useRef, useState } from "react";
import type { BackendStatus } from "../../../../../../contracts/backend";
import type { LocalState } from "../../../../../../contracts/wire.generated";

const empty: LocalState = {
    revision: 0,
    hardware: { name: "", memory: 0, supported: false },
    models: [],
    runtime: { state: "stopped", received: 0, total: 0 },
};

export function useLocalModels(backend: BackendStatus) {
    const [state, setState] = useState(empty);
    const [error, setError] = useState("");
    const current = useRef(state);
    current.current = state;
    const accept = useCallback((next: LocalState) => {
        setState((held) => (next.revision >= held.revision ? next : held));
    }, []);
    useEffect(() => {
        if (backend.state !== "ready" || !window.arxDesktop) return;
        let active = true;
        setState(empty);
        const dispose = window.arxDesktop.onLocal?.((next) => {
            if (active) accept(next);
        });
        void window.arxDesktop
            .request("local.state", undefined)
            .then((next) => {
                if (active) {
                    accept(next);
                    setError("");
                }
            })
            .catch((error) => {
                if (active) setError(error.message);
            });
        return () => {
            active = false;
            dispose?.();
        };
    }, [backend.state, accept]);
    const action = useCallback(
        async (action: string, id = "", deleteFiles = false) => {
            setError("");
            try {
                const next = await window.arxDesktop?.request("local.action", {
                    action,
                    id,
                    deleteFiles,
                });
                if (next) accept(next);
            } catch (error) {
                const message =
                    error instanceof Error
                        ? error.message
                        : "Could not update local model";
                setError(message);
                throw new Error(message);
            }
        },
        [accept],
    );
    const ensure = useCallback(
        async (id: string) => {
            if (
                current.current.runtime.model !== id &&
                current.current.runtime.state !== "stopped"
            )
                await action("unload");
            await action("load", id);
            const started = Date.now();
            while (Date.now() - started < 240000) {
                const next = await window.arxDesktop?.request("local.state", undefined);
                if (!next) throw new Error("Backend is unavailable");
                accept(next);
                if (next.runtime.state === "ready" && next.runtime.model === id)
                    return next.models.find((model) => model.id === id)?.info;
                if (next.runtime.state === "failed")
                    throw new Error(next.runtime.error || "Model could not load");
                if (next.runtime.state === "stopped")
                    throw new Error("Model loading was cancelled");
                await new Promise((resolve) => setTimeout(resolve, 300));
            }
            throw new Error("Model loading timed out");
        },
        [action, accept],
    );
    return {
        ...state,
        error: error || state.error || state.runtime.error || "",
        action,
        ensure,
    };
}
export type LocalModelsState = ReturnType<typeof useLocalModels>;
