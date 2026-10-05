import {
    createContext,
    useContext,
    useCallback,
    useEffect,
    useState,
    type ReactNode,
} from "react";
import type {
    GenerationCategory,
    GenerationState,
} from "../../../../../contracts/wire.generated";
import { applyGeneration, mergeGeneration } from "./merge";
const empty: GenerationState = {
    library: { models: [], defaults: {}, revision: 0 },
    jobs: [],
    catalog: [],
};
const Context = createContext<Generation | null>(null);
export function GenerationProvider({ children }: { children: ReactNode }) {
    const [state, setState] = useState(empty),
        [error, setError] = useState(""),
        [loading, setLoading] = useState(true);
    useEffect(() => {
        const bridge = window.arxDesktop;
        if (!bridge) {
            setLoading(false);
            return;
        }
        let active = true,
            epoch = 0,
            ready = false;
        const fetchState = () => {
            const version = ++epoch;
            setLoading(true);
            setError("");
            void bridge
                .request("generation.state", undefined)
                .then((next) => {
                    if (active && version === epoch)
                        setState((held) => mergeGeneration(held, next));
                })
                .catch((e) => {
                    if (active && version === epoch) setError(e.message);
                })
                .finally(() => {
                    if (active && version === epoch) setLoading(false);
                });
        };
        const stopEvents = bridge.onGeneration((event) => {
            if (active && ready) setState((held) => applyGeneration(held, event));
        });
        const stopStatus = bridge.onBackendStatus((status) => {
            if (!active) return;
            if (status.state === "ready") {
                if (!ready) {
                    ready = true;
                    fetchState();
                }
            } else {
                ready = false;
                epoch++;
                setState(empty);
                setLoading(status.state === "starting");
            }
        });
        return () => {
            active = false;
            epoch++;
            stopEvents();
            stopStatus();
        };
    }, []);
    const action = useCallback(async (action: string, id: string) => {
        setError("");
        try {
            await window.arxDesktop?.request("generation.action", { action, id });
        } catch (error) {
            setError(error instanceof Error ? error.message : "Could not update model");
        }
    }, []);
    const configure = useCallback(async (category: GenerationCategory, id: string) => {
        setError("");
        try {
            await window.arxDesktop?.request("generation.configure", { category, id });
        } catch (error) {
            setError(error instanceof Error ? error.message : "Could not update default");
        }
    }, []);
    const value = { ...state, error, loading, action, configure };
    return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useGeneration() {
    const state = useContext(Context);
    if (!state) throw new Error("GenerationProvider is required");
    return state;
}
export type Generation = GenerationState & {
    error: string;
    loading: boolean;
    action: (action: string, id: string) => Promise<void>;
    configure: (category: GenerationCategory, id: string) => Promise<void>;
};
