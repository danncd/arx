import { useRef, useState } from "react";
import type { useChat } from "../features/chat/useChat";
import type { LocalModelsState } from "../features/models/providers/local/useLocalModels";
import { useNotifications } from "../ui/notifications/Notifications";

export function useChatActions(
    model: string,
    availability: "ready" | "requiresLoad" | "unavailable",
    chat: ReturnType<typeof useChat>,
    local: LocalModelsState,
) {
    const [loadingLocal, setLoadingLocal] = useState(false);
    const pending = useRef(false);
    const cancelled = useRef(false);
    const { notify } = useNotifications();
    const send = () => {
        if (
            pending.current ||
            chat.running ||
            chat.starting ||
            availability === "unavailable"
        )
            return;
        if (availability === "ready") {
            void chat.send();
            return;
        }
        pending.current = true;
        cancelled.current = false;
        setLoadingLocal(true);
        void local
            .ensure(model)
            .then(async () => {
                if (!cancelled.current) await chat.send();
            })
            .catch((error) => {
                if (!cancelled.current) notify(error.message, "local-send");
            })
            .finally(() => {
                pending.current = false;
                setLoadingLocal(false);
            });
    };
    const stop = () => {
        if (chat.running) void chat.stop();
        else if (pending.current) {
            cancelled.current = true;
            void local.action("unload").catch(() => {});
        }
    };
    return { send, stop, loadingLocal };
}
