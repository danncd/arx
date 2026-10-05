import { useId, useLayoutEffect, useRef, useState, type KeyboardEvent } from "react";
import { useAnchoredPopup } from "../../../ui/popover/useAnchoredPopup";

export interface ModelOption {
    value: string;
    label: string;
    provider: string;
    unavailable?: string;
    unloaded?: boolean;
}

export function useModelMenu(
    value: string,
    options: ModelOption[],
    disabled: boolean,
    onChange: (id: string) => Promise<boolean | undefined>,
) {
    const id = useId();
    const trigger = useRef<HTMLButtonElement>(null);
    const menu = useRef<HTMLDivElement>(null);
    const modelRow = useRef<HTMLButtonElement>(null);
    const list = useRef<HTMLDivElement>(null);
    const lastFocused = useRef<HTMLElement | null>(null);
    const search = useRef({ text: "", time: 0 });
    const [stage, setStage] = useState<"closed" | "settings" | "models">("closed");
    const navigation = useRef(0);
    const open = stage !== "closed";
    const models = stage === "models";
    const showSettings = () => {
        navigation.current++;
        setStage("settings");
    };
    const [pending, setPending] = useState(false);
    const [active, setActive] = useState(0);
    const placement = useAnchoredPopup(open, trigger, 280);
    const selected = options.find((option) => option.value === value);
    const busy = disabled || pending;
    const close = (focus = false) => {
        navigation.current++;
        setStage("closed");
        if (focus) trigger.current?.focus({ preventScroll: true });
    };
    const showModels = () => {
        setActive(
            Math.max(
                0,
                options.findIndex((option) => option.value === value),
            ),
        );
        navigation.current++;
        setStage("models");
    };
    const choose = async (index: number) => {
        const option = options[index];
        if (!option || option.unavailable || busy) return;
        const version = navigation.current;
        setPending(true);
        try {
            if (option.value === value || (await onChange(option.value))) {
                if (version === navigation.current) showSettings();
            }
        } catch {
            // Keep the current page and focus after a failed save.
        } finally {
            setPending(false);
        }
    };

    useLayoutEffect(() => {
        setActive((current) => Math.max(0, Math.min(current, options.length - 1)));
    }, [options.length]);

    useLayoutEffect(() => {
        if (!open || busy) return;
        if (models) list.current?.focus({ preventScroll: true });
        else modelRow.current?.focus({ preventScroll: true });
        search.current = { text: "", time: 0 };
    }, [open, models, busy]);

    useLayoutEffect(() => {
        if (!open || !menu.current || !trigger.current) return;
        const popup = menu.current;
        const button = trigger.current;
        const outside = (event: Event) => {
            if (
                event.target instanceof Node &&
                !button.contains(event.target) &&
                !popup.contains(event.target)
            )
                close();
        };
        const escape = (event: globalThis.KeyboardEvent) => {
            if (event.key !== "Escape") return;
            event.preventDefault();
            event.stopPropagation();
            if (models) showSettings();
            else close(true);
        };

        const contain = (event: globalThis.KeyboardEvent) => {
            if (event.key !== "Tab") return;
            const reachable = [
                ...popup.querySelectorAll<HTMLElement>(
                    'button:not([disabled]), [href], input:not([disabled]), select, textarea, [tabindex]:not([tabindex="-1"])',
                ),
            ].filter((node) => node.offsetParent !== null);
            if (reachable.length === 0) return;
            const first = reachable[0]!;
            const last = reachable[reachable.length - 1]!;
            const inside = event.target instanceof Node && popup.contains(event.target);
            if (!inside) {
                event.preventDefault();
                first.focus({ preventScroll: true });
                return;
            }
            if (event.shiftKey && event.target === first) {
                event.preventDefault();
                last.focus({ preventScroll: true });
            } else if (!event.shiftKey && event.target === last) {
                event.preventDefault();
                first.focus({ preventScroll: true });
            }
        };
        window.addEventListener("keydown", contain, true);
        window.addEventListener("keydown", escape, true);
        window.addEventListener("pointerdown", outside, true);
        window.addEventListener("focusin", outside);
        return () => {
            window.removeEventListener("keydown", contain, true);
            window.removeEventListener("keydown", escape, true);
            window.removeEventListener("pointerdown", outside, true);
            window.removeEventListener("focusin", outside);
        };
    }, [open, models, options.length]);

    useLayoutEffect(() => {
        const option = document.getElementById(`${id}-${active}`);
        const viewport = list.current;
        if (!models || !option || !viewport) return;
        const bounds = option.getBoundingClientRect(),
            area = viewport.getBoundingClientRect();
        if (bounds.top < area.top) viewport.scrollTop -= area.top - bounds.top;
        if (bounds.bottom > area.bottom)
            viewport.scrollTop += bounds.bottom - area.bottom;
    }, [models, active]);

    useLayoutEffect(() => {
        if (
            open &&
            !busy &&
            document.activeElement === document.body &&
            lastFocused.current?.isConnected
        )
            lastFocused.current.focus({ preventScroll: true });
    }, [open, busy]);

    const handleListKey = (event: KeyboardEvent) => {
        if (busy || event.nativeEvent.isComposing) return;
        if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
            event.preventDefault();
            setActive((current) =>
                event.key === "Home"
                    ? 0
                    : event.key === "End"
                      ? options.length - 1
                      : (current + (event.key === "ArrowUp" ? -1 : 1) + options.length) %
                        options.length,
            );
        } else if (["Enter", " "].includes(event.key)) {
            event.preventDefault();
            void choose(active);
        } else if (
            event.key.length === 1 &&
            !event.metaKey &&
            !event.ctrlKey &&
            !event.altKey
        ) {
            event.preventDefault();
            const now = Date.now();
            const text =
                (now - search.current.time < 700 ? search.current.text : "") +
                event.key.toLocaleLowerCase();
            search.current = { text, time: now };
            const query = [...text].every((char) => char === text[0]) ? text[0]! : text;
            for (let step = 1; step <= options.length; step++) {
                const index = (active + step) % options.length;
                if (options[index]?.label.toLocaleLowerCase().startsWith(query)) {
                    setActive(index);
                    break;
                }
            }
        }
    };

    return {
        id,
        trigger,
        menu,
        modelRow,
        list,
        lastFocused,
        placement,
        selected,
        busy,
        open,
        models,
        pending,
        active,
        close,
        showModels,
        choose,
        handleListKey,
        showSettings,
        highlight: setActive,
    };
}
