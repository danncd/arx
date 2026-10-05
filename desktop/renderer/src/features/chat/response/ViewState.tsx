import {
    createContext,
    useContext,
    useState,
    type Dispatch,
    type SetStateAction,
    type ReactNode,
} from "react";

export const MessageScope = createContext("");
const ViewState = createContext<{
    values: Record<string, unknown>;
    setValues: Dispatch<SetStateAction<Record<string, unknown>>>;
} | null>(null);

export function ResponseViewProvider({ children }: { children: ReactNode }) {
    const [values, setValues] = useState<Record<string, unknown>>({});
    return (
        <ViewState.Provider value={{ values, setValues }}>{children}</ViewState.Provider>
    );
}

export function useRememberedState<T>(
    key: string,
    initial: T,
): [T, Dispatch<SetStateAction<T>>] {
    const state = useContext(ViewState);
    if (!state) throw new Error("Response view state is unavailable");
    const value = Object.hasOwn(state.values, key) ? (state.values[key] as T) : initial;
    return [
        value,
        (update) =>
            state.setValues((current) => {
                const previous = Object.hasOwn(current, key)
                    ? (current[key] as T)
                    : initial;
                const next =
                    typeof update === "function"
                        ? (update as (value: T) => T)(previous)
                        : update;
                return Object.is(previous, next) ? current : { ...current, [key]: next };
            }),
    ];
}
