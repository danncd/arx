import { useEffect } from "react";
import type { Dispatch, SetStateAction } from "react";
import type { Preferences } from "../platform/preferences/usePreferences";
import type { SidebarLayout, useSidebar } from "../shell/useSidebar";

export function useShellPreferences(
    layout: SidebarLayout,
    sidebar: ReturnType<typeof useSidebar>,
    preferences: Preferences,
    settings: boolean,
    setSettings: Dispatch<SetStateAction<boolean>>,
    newSession: () => void,
) {
    useEffect(() => window.arxDesktop?.signalReady(), []);
    useEffect(() => {
        if (sidebar.dragging) return;
        const value = JSON.stringify({ open: layout.open, width: layout.savedWidth });
        if (preferences.views.sidebar !== value)
            void preferences.saveView("sidebar", value);
    }, [
        layout.open,
        layout.savedWidth,
        sidebar.dragging,
        preferences.saveView,
        preferences.views.sidebar,
    ]);
    useEffect(() => {
        const keydown = (event: KeyboardEvent) => {
            if (!(event.metaKey || event.ctrlKey)) return;
            if (event.key.toLowerCase() === "n") {
                event.preventDefault();
                if (!settings) newSession();
            } else if (event.key === ",") {
                event.preventDefault();
                setSettings(true);
            }
        };
        window.addEventListener("keydown", keydown);
        return () => window.removeEventListener("keydown", keydown);
    }, [newSession, settings]);
}
