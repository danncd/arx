import { useCallback, useRef, useState } from "react";
import type { PermissionPolicy } from "../../../../contracts/wire.generated";
import type { RunSettings, Snapshot } from "../../../../contracts/wire.generated";
import { useNotifications } from "../../ui/notifications/Notifications";

export function usePreferences(initial: Snapshot) {
    const [settings, setSettings] = useState({
        ...initial.settings,
        directory: initial.settings.directory ?? "",
    });
    const [views, setViews] = useState(initial.views);
    const current = useRef({
        ...initial.settings,
        directory: initial.settings.directory ?? "",
    });
    const queue = useRef(Promise.resolve());
    const { notify } = useNotifications();
    const accept = useCallback((saved: Snapshot["settings"]) => {
        current.current = {
            ...saved,
            directory: saved.directory ?? "",
            chat_permissions: {
                ...current.current.chat_permissions,
                ...saved.chat_permissions,
            },
        };
        setSettings(current.current);
    }, []);
    const saveView = useCallback(
        (key: string, value: string) => {
            setViews((held) => ({ ...held, [key]: value }));
            const saved =
                window.arxDesktop?.request("save_view", { key, value }) ||
                Promise.resolve();
            void saved.catch(() => notify("Couldn’t save changes", "save-view"));
            return saved;
        },
        [notify],
    );
    const configure = useCallback(
        (change: {
            run?: Partial<RunSettings>;
            directory?: string;
            auto_continue?: boolean;
        }) => {
            const task = queue.current.then(async () => {
                const next = {
                    ...current.current,
                    ...change,
                    run: { ...current.current.run, ...change.run },
                };
                const saved = window.arxDesktop
                    ? await window.arxDesktop.request("configure", nextRequest(next))
                    : next;
                accept(saved);
            });
            queue.current = task.catch(() =>
                notify("Couldn’t save settings", "save-settings"),
            );
            return task;
        },
        [notify, accept],
    );
    const configurePermissions = useCallback(
        (conversation: string, policy: PermissionPolicy) => {
            const task = queue.current.then(async () => {
                const saved = window.arxDesktop
                    ? await window.arxDesktop.request("permissions.configure", {
                          conversation,
                          ...policy,
                      })
                    : conversation
                      ? {
                            ...current.current,
                            chat_permissions: {
                                ...current.current.chat_permissions,
                                [conversation]: policy,
                            },
                        }
                      : { ...current.current, permissions: policy };
                accept(saved);
            });
            queue.current = task.catch(() => {});
            return task;
        },
        [accept],
    );
    const configureLocalIdle = useCallback(
        (minutes: number) => {
            const task = queue.current.then(async () => {
                const saved = window.arxDesktop
                    ? await window.arxDesktop.request("local.configure", {
                          idle_minutes: minutes,
                      })
                    : { ...current.current, local_idle_minutes: minutes };
                accept(saved);
            });
            queue.current = task.catch(() =>
                notify("Couldn’t save local model timeout", "local-idle"),
            );
            return task;
        },
        [accept, notify],
    );
    const rememberPermissions = useCallback(
        (conversation: string, policy: PermissionPolicy) => {
            current.current = {
                ...current.current,
                chat_permissions: {
                    ...current.current.chat_permissions,
                    [conversation]: policy,
                },
            };
            setSettings(current.current);
        },
        [],
    );
    return {
        settings,
        views,
        saveView,
        configure,
        configurePermissions,
        configureLocalIdle,
        rememberPermissions,
    };
}

function nextRequest(settings: Snapshot["settings"]) {
    return {
        run: settings.run,
        directory: settings.directory ?? "",
        auto_continue: settings.auto_continue ?? true,
    };
}

export type Preferences = ReturnType<typeof usePreferences>;
