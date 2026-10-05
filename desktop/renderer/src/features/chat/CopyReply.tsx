import { CheckIcon, CopyIcon } from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { useNotifications } from "../../ui/notifications/Notifications";
import { usePresentation } from "../../platform/desktop/Presentation";

export function CopyReply({ text }: { text: string }) {
    const bridge = usePresentation();
    const copy = () => bridge.copy(text);
    const [status, setStatus] = useState("");
    const { notify } = useNotifications();
    useEffect(() => {
        if (!status) return;
        const timer = setTimeout(() => setStatus(""), 1800);
        return () => clearTimeout(timer);
    }, [status]);
    return (
        <button
            type="button"
            className="icon-button copy-button"
            aria-label={status || "Copy reply"}
            title={status || "Copy reply"}
            onClick={() =>
                void copy().then(
                    () => setStatus("Copied"),
                    () => notify("Could not copy this reply."),
                )
            }
        >
            {status === "Copied" ? <CheckIcon size={15} /> : <CopyIcon size={15} />}
        </button>
    );
}
