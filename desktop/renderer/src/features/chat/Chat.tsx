import type { ChatEvent } from "../../../../contracts/wire.generated";
import { Attachments } from "./attachments/Attachments";
import { useLayoutEffect, useMemo, useRef } from "react";
import { ArrowDownIcon } from "@phosphor-icons/react";
import type { Session } from "../sessions/types";
import { AssistantMessage } from "./response/AssistantMessage";
import { responseTurns } from "./response/turns";
import { WorkingLine } from "./response/WorkingLine";
import { useAutoScroll } from "./useAutoScroll";
import "./response/response.css";

export function Chat({
    session,
    activeMessage,
    stopping,
    compacting,
    recovery,
    more,
    loading,
    onLoadOlder,
}: {
    session?: Session;
    activeMessage?: string;
    stopping: boolean;
    compacting?: boolean;
    recovery?: ChatEvent["recovery"];
    more: boolean;
    loading: boolean;
    onLoadOlder: () => Promise<void>;
}) {
    if (!session)
        return (
            <div className="empty-chat">
                <img
                    className="welcome-mark"
                    src="./arx.png"
                    width={88}
                    height={88}
                    alt=""
                />
                <p>Send a message to start this session.</p>
                <span className="welcome-version">0.1.0</span>
            </div>
        );
    return (
        <Messages
            session={session}
            activeMessage={activeMessage}
            stopping={stopping}
            compacting={compacting}
            recovery={recovery}
            more={more}
            loading={loading}
            onLoadOlder={onLoadOlder}
        />
    );
}

function Messages({
    session,
    activeMessage,
    stopping,
    compacting,
    recovery,
    more,
    loading,
    onLoadOlder,
}: {
    session: Session;
    activeMessage?: string;
    stopping: boolean;
    compacting?: boolean;
    recovery?: ChatEvent["recovery"];
    more: boolean;
    loading: boolean;
    onLoadOlder: () => Promise<void>;
}) {
    const scroll = useAutoScroll();
    const anchor = useRef<{ height: number; top: number } | null>(null);
    const turns = useMemo(() => responseTurns(session.messages), [session.messages]);
    const active = turns.find(
        (turn) => activeMessage && turn.ids.includes(activeMessage),
    );
    useLayoutEffect(() => {
        const element = scroll.container.current;
        if (!element || !anchor.current || loading) return;
        element.scrollTop =
            anchor.current.top + element.scrollHeight - anchor.current.height;
        anchor.current = null;
    }, [session.messages, loading, scroll.container]);
    return (
        <div className="stream-wrap">
            <div
                className="message-scroll"
                ref={scroll.container}
                onScroll={scroll.onScroll}
                aria-busy={loading}
            >
                <div
                    className="messages"
                    ref={scroll.content}
                    data-live={Boolean(active)}
                >
                    {more && (
                        <button
                            className="history-button"
                            disabled={loading}
                            onClick={() => {
                                const element = scroll.container.current;
                                if (element)
                                    anchor.current = {
                                        height: element.scrollHeight,
                                        top: element.scrollTop,
                                    };
                                scroll.release();
                                void onLoadOlder();
                            }}
                        >
                            Earlier messages
                        </button>
                    )}
                    {turns.map(({ message, ids }) => (
                        <article
                            className={`message ${message.role}`}
                            key={message.id}
                            aria-label={message.role === "user" ? "You" : "Arx"}
                        >
                            {message.role === "user" ? (
                                <>
                                    <Attachments images={message.images || []} />
                                    {message.text && (
                                        <div className="user-message">{message.text}</div>
                                    )}
                                </>
                            ) : (
                                <AssistantMessage
                                    message={message}
                                    streaming={Boolean(
                                        activeMessage && ids.includes(activeMessage),
                                    )}
                                    onExpand={scroll.release}
                                />
                            )}
                        </article>
                    ))}
                    <WorkingLine
                        recovery={recovery}
                        status={
                            !activeMessage
                                ? "done"
                                : stopping
                                  ? "cancelling"
                                  : compacting
                                    ? "compacting"
                                    : "running"
                        }
                        startedAt={active?.message.createdAt || active?.message.at || ""}
                    />
                </div>
            </div>
            {scroll.away && (
                <button
                    type="button"
                    className="jump-button"
                    aria-label="Jump to latest"
                    onClick={scroll.pin}
                >
                    <ArrowDownIcon size={16} />
                </button>
            )}
        </div>
    );
}
