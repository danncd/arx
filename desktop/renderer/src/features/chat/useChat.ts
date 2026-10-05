import type { useAttachments } from "./attachments/useAttachments";
import { useEffect, useRef, useState } from "react";
import type { PermissionPolicy } from "../../../../contracts/wire.generated";
import type { BackendStatus } from "../../../../contracts/backend";
import type { ChatEvent } from "../../../../contracts/wire.generated";
import type { useSessions } from "../sessions/useSessions";
import { useNotifications } from "../../ui/notifications/Notifications";

type Sessions = ReturnType<typeof useSessions>;
const idle: ChatEvent = { revision: 0, run: { id: "", conversation: "", state: "idle" } };

export function useChat(
    status: BackendStatus,
    sessions: Sessions,
    attachments: ReturnType<typeof useAttachments>,
    permissions: {
        policy: PermissionPolicy;
        remember: (conversation: string, policy: PermissionPolicy) => void;
    },
) {
    const [event, setEvent] = useState(idle);
    const [starting, setStarting] = useState(false);
    const sending = useRef(false);
    const revision = useRef(-1);
    const { notify } = useNotifications();
    const { receive, refresh } = sessions;
    useEffect(() => {
        if (!window.arxDesktop || status.state !== "ready") {
            setEvent(idle);
            return;
        }
        let active = true;
        revision.current = -1;
        const apply = (next: ChatEvent) => {
            if (!active || next.revision <= revision.current) return;
            revision.current = next.revision;
            setEvent(next);
            receive(next);
            if (next.error && !next.message?.failed) notify(next.error, "chat-run");
        };
        const unsubscribe = window.arxDesktop.onChat(apply);
        void refresh()
            .then(() => window.arxDesktop!.request("chat.state", undefined))
            .then(apply)
            .catch(() => {
                if (active) notify("Couldn’t restore chat", "chat-state");
            });
        return () => {
            active = false;
            unsubscribe();
        };
    }, [status.state, receive, refresh, notify]);
    const send = async () => {
        if (
            !window.arxDesktop ||
            sending.current ||
            event.run.state !== "idle" ||
            attachments.busy ||
            (!sessions.draft.trim() && !attachments.images.length)
        )
            return;
        sending.current = true;
        setStarting(true);
        const text = sessions.draft;
        const images = attachments.images;
        const conversation = sessions.selected;
        try {
            const run = await window.arxDesktop.request("chat.send", {
                id: crypto.randomUUID(),
                conversation,
                text,
                images,
                permissions: conversation ? undefined : permissions.policy,
            });
            if (!conversation) permissions.remember(run.conversation, permissions.policy);
            await attachments.sent(conversation, images);
            await sessions.sent(run.conversation, conversation, text);
        } catch (error) {
            notify(
                error instanceof Error ? error.message : "Couldn’t send message",
                "chat-send",
            );
        } finally {
            sending.current = false;
            setStarting(false);
        }
    };
    const stop = async () => {
        try {
            await window.arxDesktop?.request("chat.stop", { id: event.run.id });
        } catch {
            notify("Couldn’t stop reply", "chat-stop");
        }
    };
    return {
        event,
        starting,
        send,
        stop,
        running: event.run.state !== "idle",
        cancelling: event.run.state === "stopping",
    };
}
