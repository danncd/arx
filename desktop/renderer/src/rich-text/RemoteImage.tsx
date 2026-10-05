import { usePresentation } from "../platform/desktop/Presentation";
import { imageURL, generatedImageURL } from "./imageSource";
import { useEffect, useRef, useState } from "react";
import { ImagePreview } from "../ui/image-preview/ImagePreview";
import { useResourcePolicy } from "./ResourcePolicy";
import "./remote-image.css";

const savedImages = new Map<string, Promise<string>>();
function savedImage(url: string): Promise<string> {
    const generated = generatedImageURL(url);
    if (generated) return Promise.resolve(generated);
    let pending = savedImages.get(url);
    if (!pending) {
        if (savedImages.size >= 12) savedImages.delete(savedImages.keys().next().value!);
        pending = window.arxDesktop?.readImage(url.slice(10)) || Promise.resolve("");
        savedImages.set(url, pending);
        void pending.catch(() => savedImages.delete(url));
    }
    return pending;
}
function validPixels(data: string) {
    return (
        /^data:image\/(png|jpeg|gif|webp);base64,/.test(data) ||
        /^arx-media:\/\/artifact\/[a-f0-9]{32}$/.test(data)
    );
}
export function RemoteImage({ source, alt }: { source: string; alt: string }) {
    const bridge = usePresentation();
    const url = imageURL(source);
    const root = useRef<HTMLSpanElement>(null);
    const { conversation, mode } = useResourcePolicy();
    const [data, setData] = useState("");
    const [error, setError] = useState("");
    const [busy, setBusy] = useState(false);
    const [expanded, setExpanded] = useState(false);
    const operation = useRef("");
    const lifetime = useRef(0);
    const remote = Boolean(url && !url.startsWith("arx-"));
    const needsApproval = remote && mode === "ask";
    useEffect(() => {
        const version = ++lifetime.current;
        setData("");
        setError("");
        setBusy(false);
        setExpanded(false);
        if (!url || !root.current) return;
        const observer = new IntersectionObserver(
            (entries) => {
                if (!entries.some((entry) => entry.isIntersecting)) return;
                observer.disconnect();
                if (needsApproval || (remote && !conversation)) return;
                setBusy(true);
                const pending = remote
                    ? window.arxDesktop!.request("web.image", { url, conversation })
                    : savedImage(url);
                void pending
                    .then((value) => {
                        if (version !== lifetime.current) return;
                        if (!validPixels(value)) throw Error("Image unavailable");
                        setData(value);
                    })
                    .catch(() => {
                        if (version === lifetime.current) setError("Image unavailable");
                    })
                    .finally(() => {
                        if (version === lifetime.current) setBusy(false);
                    });
            },
            { rootMargin: "200px" },
        );
        observer.observe(root.current);
        return () => {
            lifetime.current++;
            observer.disconnect();
            if (operation.current)
                void window.arxDesktop
                    ?.request("tools.cancel", { id: operation.current })
                    .catch(() => {});
            operation.current = "";
        };
    }, [url, conversation, needsApproval, remote]);
    const load = async () => {
        if (!url || !conversation || operation.current || !window.arxDesktop) return;
        const id = crypto.randomUUID();
        const version = lifetime.current;
        operation.current = id;
        setBusy(true);
        setError("");
        try {
            const result = await window.arxDesktop.request("tools.run", {
                id,
                conversation,
                name: "web",
                arguments: { operation: "image", url },
            });
            if (version !== lifetime.current) return;
            if (result.failed || !result.images?.length)
                throw Error(result.text || "Image unavailable");
            const value = await window.arxDesktop.readImage(result.images[0]!.id);
            if (version !== lifetime.current) return;
            if (!validPixels(value)) throw Error("Image unavailable");
            setData(value);
        } catch (failure) {
            if (version === lifetime.current)
                setError(
                    failure instanceof Error ? failure.message : "Image unavailable",
                );
        } finally {
            if (operation.current === id) operation.current = "";
            if (version === lifetime.current) setBusy(false);
        }
    };
    if (!url) return <span>{alt || "Image"}</span>;
    return (
        <span className="response-image" ref={root}>
            {data ? (
                <button
                    type="button"
                    className="response-image-button"
                    aria-label={`Preview ${alt || "image"}`}
                    onClick={() => setExpanded(true)}
                >
                    <img
                        src={data}
                        alt={alt}
                        onError={() => {
                            setData("");
                            setError("Image unavailable");
                        }}
                    />
                </button>
            ) : busy ? (
                <span className="response-image-loading" role="status">
                    Loading image…
                </span>
            ) : (
                <span>
                    {error && remote ? (
                        <a
                            href={url}
                            onClick={(event) => {
                                event.preventDefault();
                                void bridge.openExternal(url).catch(() => {});
                            }}
                        >
                            {alt || "View image"}
                        </a>
                    ) : (
                        <span>{error || alt || "Remote image"}</span>
                    )}
                    {remote && (
                        <button
                            type="button"
                            disabled={!conversation}
                            onClick={() => void load()}
                        >
                            {error ? "Retry image" : "Load image"}
                        </button>
                    )}
                </span>
            )}
            {expanded && data && (
                <ImagePreview
                    source={data}
                    name={alt || "image"}
                    onClose={() => setExpanded(false)}
                />
            )}
        </span>
    );
}
