import { permissionModes, changePermissionMode } from "../permissions/modes";
import { XIcon } from "@phosphor-icons/react";
import { useState } from "react";
import type {
    PermissionMode,
    PermissionPolicy,
} from "../../../../contracts/wire.generated";
import { useNotifications } from "../../ui/notifications/Notifications";
import { Select } from "../../ui/select/Select";
import "../permissions/permissions.css";

export function Permissions({
    policy,
    directory,
    onChange,
}: {
    policy: PermissionPolicy;
    directory: string;
    onChange: (policy: PermissionPolicy) => Promise<void>;
}) {
    const [saving, setSaving] = useState(false);
    const { notify } = useNotifications();
    const save = async (next: PermissionPolicy) => {
        setSaving(true);
        try {
            await onChange(next);
        } catch (error) {
            notify(
                error instanceof Error ? error.message : "Couldn’t save permissions",
                "permissions",
            );
        } finally {
            setSaving(false);
        }
    };
    const changeMode = (mode: PermissionMode) => {
        void save(changePermissionMode(policy, mode, directory));
    };
    const addFolder = async () => {
        try {
            const root = await window.arxDesktop?.chooseDirectory();
            if (root && !policy.roots.includes(root))
                await save({ ...policy, roots: [...policy.roots, root] });
        } catch {
            notify("Couldn’t choose a folder", "permissions");
        }
    };
    return (
        <section aria-label="Permissions">
            <h3>Permissions</h3>
            <h4 className="settings-section-title">Access</h4>
            <div className="general-setting">
                <label>Default permission mode</label>
                <Select
                    label="Permission mode"
                    choices={permissionModes}
                    value={policy.mode}
                    disabled={saving}
                    onChange={changeMode}
                />
            </div>
            <p className="permission-description">
                {permissionModes.find((mode) => mode.value === policy.mode)?.description}
            </p>
            {policy.mode === "folders" && (
                <>
                    <h4 className="settings-section-title">Folders</h4>
                    <ul className="permission-folders">
                        {policy.roots.map((root) => (
                            <li key={root}>
                                <span>{root}</span>
                                <button
                                    type="button"
                                    className="icon-button"
                                    aria-label={`Remove ${root}`}
                                    disabled={saving || policy.roots.length === 1}
                                    onClick={() =>
                                        void save({
                                            ...policy,
                                            roots: policy.roots.filter(
                                                (held) => held !== root,
                                            ),
                                        })
                                    }
                                >
                                    <XIcon size={14} />
                                </button>
                            </li>
                        ))}
                    </ul>
                    <button
                        type="button"
                        className="directory-button"
                        disabled={saving || !window.arxDesktop}
                        onClick={() => void addFolder()}
                    >
                        Add folder…
                    </button>
                </>
            )}
        </section>
    );
}
