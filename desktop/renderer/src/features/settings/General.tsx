import { useState } from "react";
import type { Preferences } from "../../platform/preferences/usePreferences";
import { useNotifications } from "../../ui/notifications/Notifications";
import { Select } from "../../ui/select/Select";
const continuationChoices = [
    { value: "on", label: "On" },
    { value: "off", label: "Off" },
] as const;
const choices = [
    { value: "new", label: "New session" },
    { value: "recent", label: "Most recent session" },
] as const;
export function General({ preferences }: { preferences: Preferences }) {
    const [choosing, setChoosing] = useState(false);
    const [savingContinuation, setSavingContinuation] = useState(false);
    const { notify } = useNotifications();
    const value = preferences.views["startup-mode"] === "new" ? "new" : "recent";
    const chooseDirectory = async () => {
        setChoosing(true);
        try {
            const directory = await window.arxDesktop?.chooseDirectory();
            if (directory) await preferences.configure({ directory });
        } catch {
            notify("Couldn’t change working directory", "directory");
        } finally {
            setChoosing(false);
        }
    };
    return (
        <>
            <h3>General</h3>
            <h4 className="settings-section-title">Startup</h4>
            <div className="general-setting">
                <label>On startup</label>
                <Select
                    label="On startup"
                    value={value}
                    choices={choices}
                    onChange={(mode) => {
                        void preferences.saveView("startup-mode", mode);
                    }}
                />
            </div>
            <h4 className="settings-section-title">Responses</h4>
            <div className="general-setting">
                <div className="general-value">
                    <label>Auto continue</label>
                    <p>
                        Continue after tool limits and recover from context or reply
                        limits, with up to three attempts.
                    </p>
                </div>
                <Select
                    label="Auto continue"
                    choices={continuationChoices}
                    value={preferences.settings.auto_continue === false ? "off" : "on"}
                    disabled={savingContinuation}
                    onChange={(mode) => {
                        setSavingContinuation(true);
                        void preferences
                            .configure({ auto_continue: mode === "on" })
                            .catch(() => {})
                            .finally(() => setSavingContinuation(false));
                    }}
                />
            </div>
            <h4 className="settings-section-title">Environment</h4>
            <div className="general-setting">
                <div className="directory-value">
                    <label>Working directory</label>
                    <p title={preferences.settings.directory}>
                        {preferences.settings.directory || "Not selected"}
                    </p>
                </div>
                <button
                    className="directory-button"
                    type="button"
                    disabled={choosing || !window.arxDesktop}
                    onClick={() => void chooseDirectory()}
                >
                    Choose…
                </button>
            </div>
        </>
    );
}
