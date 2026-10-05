import type { NetworkModelsState } from "../models/providers/network/useNetworkModels";
import type { GenerationCategory } from "../../../../contracts/wire.generated";
import {
    GearSixIcon,
    SparkleIcon,
    PlugsConnectedIcon,
    ShieldCheckIcon,
    XIcon,
    PlugsIcon,
    BookOpenIcon,
} from "@phosphor-icons/react";
import { useLayoutEffect, useRef, useState } from "react";
import { NotificationsViewport } from "../../ui/notifications/Notifications";
import type { PermissionPolicy } from "../../../../contracts/wire.generated";
import type { Preferences } from "../../platform/preferences/usePreferences";
import { Permissions } from "../permissions/PermissionSettings";
import { Connections } from "./Connections";
import type { DeepSeekConnection } from "../models/providers/deepseek/useConnection";
import { GenerationSettings } from "../generation/settings/GenerationSettings";
import { Integrations } from "../integrations/Integrations";
import { General } from "./General";
import type { LocalModelsState } from "../models/providers/local/useLocalModels";
import "./settings.css";
import "./general.css";

export function Settings({
    onClose,
    preferences,
    permissions,
    onPermissions,
    network,
    connection,
    local,
    running,
    initialSection = "general",
}: {
    onClose: () => void;
    preferences: Preferences;
    permissions: PermissionPolicy;
    onPermissions: (policy: PermissionPolicy) => Promise<void>;
    network: NetworkModelsState;
    connection: DeepSeekConnection;
    local: LocalModelsState;
    running: boolean;
    initialSection?: "general" | "connections";
}) {
    const [section, setSection] = useState<
        "general" | "connections" | "permissions" | "generation" | "mcps" | "skills"
    >(initialSection);
    const [browsing, setBrowsing] = useState<GenerationCategory | "chat" | null>(null);
    const dialog = useRef<HTMLDialogElement>(null);
    const backdropPress = useRef(false);

    useLayoutEffect(() => {
        const element = dialog.current;
        const opener = document.activeElement;
        element?.showModal();
        return () => {
            element?.close();
            if (opener instanceof HTMLElement && opener.isConnected) {
                opener.focus({ preventScroll: true });
            }
        };
    }, []);

    const outside = (x: number, y: number) => {
        const bounds = dialog.current?.getBoundingClientRect();
        return Boolean(
            bounds &&
            (x < bounds.left || x > bounds.right || y < bounds.top || y > bounds.bottom),
        );
    };

    return (
        <dialog
            ref={dialog}
            className="settings-dialog"
            aria-labelledby="settings-title"
            onCancel={onClose}
            onPointerDown={(event) => {
                backdropPress.current =
                    !(
                        event.target instanceof Element &&
                        event.target.closest(".notifications")
                    ) && outside(event.clientX, event.clientY);
            }}
            onClick={(event) => {
                if (backdropPress.current && outside(event.clientX, event.clientY))
                    onClose();
                backdropPress.current = false;
            }}
        >
            <header className="settings-heading">
                <h2 id="settings-title">Settings</h2>
                <button
                    type="button"
                    className="icon-button"
                    aria-label="Close settings"
                    onClick={onClose}
                >
                    <XIcon size={18} />
                </button>
            </header>
            <form
                noValidate
                className="settings-form"
                onSubmit={(event) => event.preventDefault()}
            >
                <nav className="settings-nav" aria-label="Settings sections">
                    <button
                        type="button"
                        aria-current={section === "general" ? "page" : undefined}
                        onClick={() => setSection("general")}
                    >
                        <GearSixIcon size={15} />
                        General
                    </button>
                    <button
                        type="button"
                        aria-current={section === "connections" ? "page" : undefined}
                        onClick={() => {
                            setBrowsing(null);
                            setSection("connections");
                        }}
                    >
                        <PlugsConnectedIcon size={15} />
                        Connections
                    </button>
                    <button
                        type="button"
                        aria-current={section === "generation" ? "page" : undefined}
                        onClick={() => setSection("generation")}
                    >
                        <SparkleIcon size={15} />
                        Generation
                    </button>
                    <button
                        type="button"
                        aria-current={section === "mcps" ? "page" : undefined}
                        onClick={() => setSection("mcps")}
                    >
                        <PlugsIcon size={15} />
                        MCPs
                    </button>
                    <button
                        type="button"
                        aria-current={section === "skills" ? "page" : undefined}
                        onClick={() => setSection("skills")}
                    >
                        <BookOpenIcon size={15} />
                        Skills
                    </button>
                    <button
                        type="button"
                        aria-current={section === "permissions" ? "page" : undefined}
                        onClick={() => setSection("permissions")}
                    >
                        <ShieldCheckIcon size={15} />
                        Permissions
                    </button>
                </nav>
                <div className="settings-main">
                    <div className="settings-pane">
                        {section === "general" ? (
                            <section aria-label="General settings">
                                <General preferences={preferences} />
                            </section>
                        ) : section === "connections" ? (
                            <Connections
                                preferences={preferences}
                                network={network}
                                connection={connection}
                                browsing={browsing}
                                onBrowse={setBrowsing}
                                local={local}
                                running={running}
                            />
                        ) : section === "generation" ? (
                            <GenerationSettings
                                onBrowse={(category) => {
                                    setBrowsing(category);
                                    setSection("connections");
                                }}
                            />
                        ) : section === "mcps" || section === "skills" ? (
                            <Integrations
                                section={section}
                                running={running}
                                onSection={setSection}
                            />
                        ) : (
                            <Permissions
                                policy={permissions}
                                directory={preferences.settings.directory}
                                onChange={onPermissions}
                            />
                        )}
                    </div>
                </div>
            </form>
            <NotificationsViewport />
        </dialog>
    );
}
