import { useEffect, useState } from "react";
import { XIcon } from "@phosphor-icons/react";
import { ImagePreview } from "../../../ui/image-preview/ImagePreview";
import type { ImageAttachment } from "../../../../../contracts/wire.generated";
import "./attachments.css";
import "../../../styles/icon-pill.css";

export function Attachments({
    images,
    onRemove,
}: {
    images: ImageAttachment[];
    onRemove?: (id: string) => void;
}) {
    if (!images.length) return null;
    return (
        <div className="image-attachments">
            {images.map((image) => (
                <div className="image-attachment" key={image.id}>
                    <AttachmentImage image={image} />
                    {onRemove && (
                        <button
                            type="button"
                            className="icon-pill remove-attachment"
                            aria-label={`Remove ${image.name}`}
                            onClick={() => onRemove(image.id)}
                        >
                            <XIcon size={15} />
                        </button>
                    )}
                </div>
            ))}
        </div>
    );
}

function AttachmentImage({ image }: { image: ImageAttachment }) {
    const [source, setSource] = useState("");
    const [failed, setFailed] = useState(false);
    const [expanded, setExpanded] = useState(false);
    useEffect(() => {
        let active = true;
        setSource("");
        setFailed(false);
        void window.arxDesktop
            ?.readImage(image.id)
            .then((value) => {
                if (active) setSource(value);
            })
            .catch(() => {
                if (active) setFailed(true);
            });
        return () => {
            active = false;
        };
    }, [image.id]);
    return source ? (
        <>
            <button
                type="button"
                className="attachment-preview-button"
                aria-label={`Preview ${image.name}`}
                onClick={() => setExpanded(true)}
            >
                <img src={source} alt={image.name} title={image.name} />
            </button>
            {expanded && (
                <ImagePreview
                    source={source}
                    name={image.name}
                    onClose={() => setExpanded(false)}
                />
            )}
        </>
    ) : (
        <span className="image-placeholder">
            {failed ? "Image unavailable" : image.name}
        </span>
    );
}
