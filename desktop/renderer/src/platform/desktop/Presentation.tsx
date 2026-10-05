import { createContext, useContext } from "react";

export const Presentation = createContext({
    copy: async (text: string) => {
        if (window.arxDesktop) return window.arxDesktop.copy(text);
        return navigator.clipboard.writeText(text);
    },
    openExternal: async (raw: string) => {
        if (window.arxDesktop) return window.arxDesktop.openExternal(raw);
        const url = new URL(raw.startsWith("www.") ? `https://${raw}` : raw);
        if (!["https:", "http:", "mailto:"].includes(url.protocol))
            throw new Error("Unsupported link");
        window.open(url.href, "_blank", "noopener,noreferrer");
    },
});

export function usePresentation() {
    return useContext(Presentation);
}
