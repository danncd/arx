import {
    CheckCircleIcon,
    InfoIcon,
    WarningCircleIcon,
    WarningIcon,
    XIcon,
} from "@phosphor-icons/react";
import {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useState,
    type ReactNode,
} from "react";
import "./notifications.css";

export type NotificationKind = "error" | "warning" | "success" | "info";
const lifetime = 8000;
const notificationIcons = {
    error: WarningCircleIcon,
    warning: WarningIcon,
    success: CheckCircleIcon,
    info: InfoIcon,
};
type Notice = {
    id: string;
    message: string;
    kind: NotificationKind;
    expiresAt: number | null;
    remaining: number;
};
type Notifications = {
    items: Notice[];
    notify: (message: string, id?: string, kind?: NotificationKind) => void;
    dismiss: (id: string) => void;
    pause: (id: string, paused: boolean) => void;
};
const Context = createContext<Notifications | null>(null);

export function NotificationsProvider({ children }: { children: ReactNode }) {
    const [items, setItems] = useState<Notice[]>([]);
    const dismiss = useCallback(
        (id: string) => setItems((current) => current.filter((item) => item.id !== id)),
        [],
    );
    const notify = useCallback(
        (
            message: string,
            id: string = crypto.randomUUID(),
            kind: NotificationKind = "error",
        ) => {
            if (!message.trim()) return;
            setItems((current) => {
                if (current.some((item) => item.message === message)) return current;
                return [
                    ...current.filter((item) => item.id !== id),
                    {
                        id,
                        message,
                        kind,
                        expiresAt: Date.now() + lifetime,
                        remaining: lifetime,
                    },
                ].slice(-3);
            });
        },
        [],
    );
    const pause = useCallback((id: string, paused: boolean) => {
        setItems((current) => {
            const item = current.find((item) => item.id === id);
            if (!item || paused === (item.expiresAt === null)) return current;
            const remaining =
                item.expiresAt === null
                    ? item.remaining
                    : Math.max(0, item.expiresAt - Date.now());
            return current.map((item) =>
                item.id === id
                    ? {
                          ...item,
                          remaining,
                          expiresAt: paused ? null : Date.now() + remaining,
                      }
                    : item,
            );
        });
    }, []);

    useEffect(() => {
        const deadlines = items.flatMap((item) =>
            item.expiresAt === null ? [] : [item.expiresAt],
        );
        if (!deadlines.length) return;
        const timer = setTimeout(
            () => {
                const now = Date.now();
                setItems((current) =>
                    current.filter(
                        (item) => item.expiresAt === null || item.expiresAt > now,
                    ),
                );
            },
            Math.max(0, Math.min(...deadlines) - Date.now()),
        );
        return () => clearTimeout(timer);
    }, [items]);

    const value = useMemo(
        () => ({ items, notify, dismiss, pause }),
        [items, notify, dismiss, pause],
    );
    return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useNotifications() {
    const value = useContext(Context);
    if (!value) throw new Error("Notifications require the app provider");
    return value;
}

function Notification({ item }: { item: Notice }) {
    const StatusIcon = notificationIcons[item.kind];
    const { dismiss, pause } = useNotifications();
    const [hovered, setHovered] = useState(false);
    const [focused, setFocused] = useState(false);
    useEffect(() => {
        pause(item.id, hovered || focused);
        return () => pause(item.id, false);
    }, [item.id, hovered, focused, pause]);
    return (
        <div
            className="notification"
            data-kind={item.kind}
            role={item.kind === "error" || item.kind === "warning" ? "alert" : "status"}
            onMouseEnter={() => setHovered(true)}
            onMouseLeave={() => setHovered(false)}
            onFocus={() => setFocused(true)}
            onBlur={(event) => {
                if (!event.currentTarget.contains(event.relatedTarget)) setFocused(false);
            }}
        >
            <span className="notification-icon" aria-hidden="true">
                <StatusIcon size={18} />
            </span>
            <span className="notification-message">{item.message}</span>
            <button
                type="button"
                className="icon-button notification-close"
                aria-label="Dismiss notification"
                onClick={() => dismiss(item.id)}
            >
                <XIcon size={13} />
            </button>
        </div>
    );
}

export function NotificationsViewport() {
    const { items } = useNotifications();
    return (
        <aside className="notifications" aria-label="Notifications">
            {items.map((item) => (
                <Notification key={item.id} item={item} />
            ))}
        </aside>
    );
}
