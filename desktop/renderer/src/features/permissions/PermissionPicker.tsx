import { useState } from "react";
import { Select } from "../../ui/select/Select";
import { useNotifications } from "../../ui/notifications/Notifications";
import type {
    PermissionMode,
    PermissionPolicy,
} from "../../../../contracts/wire.generated";
import { permissionModes, changePermissionMode } from "./modes";
import "./permissions.css";

export function PermissionPicker({
    policy,
    directory,
    disabled,
    onChange,
}: {
    policy: PermissionPolicy;
    directory: string;
    disabled: boolean;
    onChange: (policy: PermissionPolicy) => Promise<void>;
}) {
    const [saving, setSaving] = useState(false);
    const { notify } = useNotifications();
    const choose = async (mode: PermissionMode) => {
        setSaving(true);
        try {
            await onChange(changePermissionMode(policy, mode, directory));
        } catch (error) {
            notify(
                error instanceof Error ? error.message : "Couldn’t change permissions",
                "permissions",
            );
        } finally {
            setSaving(false);
        }
    };
    return (
        <div className="permission-picker">
            <Select
                label="Permissions"
                menuTitle="Permissions mode"
                value={policy.mode}
                choices={permissionModes}
                direction="up"
                compact
                disabled={disabled || saving}
                onChange={(mode) => void choose(mode)}
            />
        </div>
    );
}
