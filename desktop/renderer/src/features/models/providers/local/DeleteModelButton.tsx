import { CheckIcon, TrashIcon } from "@phosphor-icons/react";
import { useState } from "react";

export function DeleteModelButton({
    disabled,
    onDelete,
}: {
    disabled: boolean;
    onDelete: () => Promise<void>;
}) {
    const [armed, setArmed] = useState(false);
    return (
        <button
            type="button"
            className={`local-button model-delete${armed ? " confirm" : ""}`}
            aria-label={
                armed ? "Confirm delete model and files" : "Delete model and files"
            }
            title={
                armed ? "Click again to delete model and files" : "Delete model and files"
            }
            disabled={disabled}
            onBlur={() => setArmed(false)}
            onKeyDown={(event) => {
                if (event.key === "Escape" && armed) {
                    event.preventDefault();
                    event.stopPropagation();
                    setArmed(false);
                }
            }}
            onClick={() => {
                if (!armed) {
                    setArmed(true);
                    return;
                }
                setArmed(false);
                void onDelete();
            }}
        >
            {armed ? <CheckIcon size={14} /> : <TrashIcon size={14} />}
        </button>
    );
}
