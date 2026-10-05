import { memo, useId, useRef, useMemo } from "react";
import ReactMarkdown, { defaultUrlTransform, type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import "./markdown.css";
import "./code-block.css";
import "./syntax.css";
import { imageRows } from "./imageRows";
import { RemoteImage } from "./RemoteImage";
import { CodeBlock } from "./CodeBlock";
import { useNotifications } from "../ui/notifications/Notifications";
import { usePresentation } from "../platform/desktop/Presentation";

export const Markdown = memo(function Markdown({ text }: { text: string }) {
    const { notify } = useNotifications();
    const bridge = usePresentation();
    const root = useRef<HTMLDivElement>(null);
    const id = useId();
    const renderers = useMemo<Components>(
        () => ({
            pre: CodeBlock,
            img: ({ src, alt }) => <RemoteImage source={src || ""} alt={alt || ""} />,
            h2: ({ node: _node, id: headingId, ...props }) => (
                <h2
                    {...props}
                    id={
                        headingId === "footnote-label"
                            ? `footnote-label-${id}`
                            : headingId
                    }
                />
            ),
            table: ({ children }) => (
                <div
                    className="markdown-table"
                    tabIndex={0}
                    role="region"
                    aria-label="Table"
                >
                    <table>{children}</table>
                </div>
            ),
            a: ({ node, href, children, ...props }) =>
                node?.children.some(
                    (child) => child.type === "element" && child.tagName === "img",
                ) ? (
                    <span>{children}</span>
                ) : (
                    <a
                        {...props}
                        href={href}
                        aria-describedby={
                            props["aria-describedby"] === "footnote-label"
                                ? `footnote-label-${id}`
                                : props["aria-describedby"]
                        }
                        onClick={(event) => {
                            event.preventDefault();
                            if (href?.startsWith("#")) {
                                let targetId: string;
                                try {
                                    targetId = decodeURIComponent(href.slice(1));
                                } catch {
                                    return;
                                }
                                const target = root.current?.querySelector<HTMLElement>(
                                    `#${CSS.escape(targetId)}`,
                                );
                                if (target) {
                                    target.tabIndex = -1;
                                    target.focus({ preventScroll: true });
                                    target.scrollIntoView({ block: "nearest" });
                                }
                                return;
                            }
                            if (href)
                                void bridge.openExternal(href).catch((error) => {
                                    const message =
                                        error instanceof Error
                                            ? error.message.replace(
                                                  /^Error invoking remote method [^:]+: Error: /,
                                                  "",
                                              )
                                            : "This link could not be opened.";
                                    notify(message);
                                });
                        }}
                    >
                        {children}
                    </a>
                ),
        }),
        [bridge, notify, id],
    );
    return (
        <div className="markdown" ref={root}>
            <ReactMarkdown
                urlTransform={(url) =>
                    /^arx-image:[a-f0-9]{64}$/.test(url) ||
                    /^arx-media:[a-f0-9]{32}$/.test(url) ||
                    url.startsWith("file:") ||
                    (url.startsWith("/") && !url.startsWith("//"))
                        ? url
                        : defaultUrlTransform(url)
                }
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[imageRows]}
                remarkRehypeOptions={{ clobberPrefix: `markdown-${id}-` }}
                components={renderers}
            >
                {text}
            </ReactMarkdown>
        </div>
    );
});
