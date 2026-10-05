import { useLayoutEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { XIcon } from "@phosphor-icons/react";
import "../../styles/icon-pill.css";
import "./image-preview.css";

export function ImagePreview({
    source,
    name,
    onClose,
}: {
    source: string;
    name: string;
    onClose: () => void;
}) {
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
    return createPortal(
        <dialog
            ref={dialog}
            className="image-preview-dialog"
            aria-label={`Preview ${name}`}
            onCancel={onClose}
            onPointerDown={(event) => {
                backdropPress.current = outside(event.clientX, event.clientY);
            }}
            onClick={(event) => {
                if (backdropPress.current && outside(event.clientX, event.clientY))
                    onClose();
                backdropPress.current = false;
            }}
        >
            <button
                type="button"
                className="icon-pill image-preview-close"
                aria-label="Close image preview"
                onClick={onClose}
                autoFocus
            >
                <XIcon size={15} />
            </button>
            <img className="image-preview-full" src={source} alt={name} />
        </dialog>,
        document.body,
    );
}
