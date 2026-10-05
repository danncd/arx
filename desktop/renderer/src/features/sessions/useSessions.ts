import { summary, mergeSummaries } from "./summaries";
import { useCallback, useRef, useState } from "react";
import type { InitialState } from "./types";
import type { ChatEvent, HistoryPage } from "../../../../contracts/wire.generated";
import type { Message } from "../chat/types";
import type { Preferences } from "../../platform/preferences/usePreferences";
import { useNotifications } from "../../ui/notifications/Notifications";
import { messagesFromChunks } from "./history";
import { mergeMessages } from "./merge";
import { applyPage, applyLive, type LoadedSession } from "./transitions";

export function useSessions(initial: InitialState, preferences: Preferences) {
    const [selected, setSelected] = useState(initial.selected);
    const [summaries, setSummaries] = useState(initial.snapshot.conversations);
    const [loaded, setLoaded] = useState<Record<string, LoadedSession>>(() =>
        initial.session
            ? {
                  [initial.selected]: {
                      session: initial.session,
                      before: initial.before,
                      more: initial.more,
                  },
              }
            : {},
    );
    const [loading, setLoading] = useState(false);

    const selection = useRef(0);
    const live = useRef<Record<string, Message[]>>({});
    const state = useRef({ loaded, summaries, selected, views: preferences.views });
    state.current = { loaded, summaries, selected, views: preferences.views };
    const { notify } = useNotifications();
    const current = loaded[selected]?.session;
    const draft = preferences.views[`draft:${selected || "new"}`] || "";
    const sessions = summaries.filter((item) => item.title.trim()).map(summary);
    const newSession = useCallback(() => {
        selection.current++;
        setSelected("");
        setLoading(false);
    }, []);
    const receive = useCallback((event: ChatEvent) => {
        if (event.conversation) {
            const next = event.conversation;
            setSummaries((held) => mergeSummaries(held, [next]));
        }
        if (!event.message || !event.conversation) return;
        const id = event.message.conversation;
        const messages = messagesFromChunks([event.message]);
        live.current[id] = mergeMessages(live.current[id] || [], messages);
        const item = summary(event.conversation);
        const known = state.current.summaries.some((held) => held.id === id);
        setLoaded((held) => applyLive(held, id, item, live.current[id] || [], known));
    }, []);
    const savePage = useCallback(
        (
            id: string,
            item: ReturnType<typeof summary>,
            page: HistoryPage,
            prepend = false,
        ) => {
            setLoaded((held) =>
                applyPage(held, id, item, page, live.current[id] || [], prepend),
            );
        },
        [],
    );
    const refresh = useCallback(async () => {
        if (!window.arxDesktop) return;
        const snapshot = await window.arxDesktop.request("snapshot", undefined);
        setSummaries((held) => mergeSummaries(held, snapshot.conversations, false));
        const ids = Object.keys(state.current.loaded);
        await Promise.all(
            ids.map(async (id) => {
                const item = snapshot.conversations.find((entry) => entry.id === id);
                if (!item) return;
                const page = await window.arxDesktop!.request("history", {
                    conversation: id,
                });
                savePage(id, summary(item), page);
            }),
        );
    }, [savePage]);
    const select = async (id: string) => {
        const request = ++selection.current;
        if (loaded[id]) {
            setSelected(id);
            setLoading(false);
            return;
        }
        const item = sessions.find((entry) => entry.id === id);
        if (!item) return;
        setLoading(true);
        try {
            const page = await window.arxDesktop?.request("history", {
                conversation: id,
            });
            if (page) savePage(id, item, page);
            else
                setLoaded((held) => ({
                    ...held,
                    [id]: {
                        session: {
                            ...item,
                            draft: "",
                            messages: [],
                        },
                        before: "",
                        more: false,
                    },
                }));
            if (request === selection.current) setSelected(id);
        } catch {
            notify("Couldn’t load session", "session-history");
        } finally {
            if (request === selection.current) setLoading(false);
        }
    };
    const loadOlder = async () => {
        const held = loaded[selected];
        if (!held?.more || loading || !window.arxDesktop) return;
        const request = ++selection.current;
        const id = selected;
        setLoading(true);
        try {
            const page = await window.arxDesktop.request("history", {
                conversation: id,
                before: held.before,
            });
            savePage(id, held.session, page, true);
        } catch {
            notify("Couldn’t load earlier messages", "session-history");
        } finally {
            if (request === selection.current) setLoading(false);
        }
    };
    const sent = async (id: string, from: string, text: string) => {
        const key = `draft:${from || "new"}`;
        if (state.current.views[key] === text) await preferences.saveView(key, "");
        if (state.current.selected === from) {
            selection.current++;
            setSelected(id);
            setLoading(false);
        }
    };
    return {
        sessions,
        selected,
        current,
        draft,
        loading,
        select,
        newSession,
        loadOlder,
        receive,
        refresh,
        sent,
        more: loaded[selected]?.more || false,
        updateDraft: (value: string) => {
            void preferences.saveView(`draft:${selected || "new"}`, value);
        },
    };
}
