import { CheckIcon, CopyIcon } from "@phosphor-icons/react";
import {
    Fragment,
    lazy,
    Suspense,
    useEffect,
    useLayoutEffect,
    useRef,
    useState,
    type ComponentProps,
} from "react";
import type { ExtraProps } from "react-markdown";
import { usePresentation } from "../platform/desktop/Presentation";
import { useNotifications } from "../ui/notifications/Notifications";

const SyntaxCode = lazy(() =>
    import("./SyntaxCode").then((module) => ({ default: module.SyntaxCode })),
);

const languageNames: Record<string, string> = {
    ts: "TypeScript",
    typescript: "TypeScript",
    tsx: "TSX",
    js: "JavaScript",
    javascript: "JavaScript",
    jsx: "JSX",
    c: "C",
    cpp: "C++",
    java: "Java",
    swift: "Swift",
    xml: "XML",
    json: "JSON",
    jsonc: "JSONC",
    html: "HTML",
    css: "CSS",
    sh: "Shell",
    shell: "Shell",
    bash: "Bash",
    zsh: "Zsh",
    py: "Python",
    python: "Python",
    go: "Go",
    rust: "Rust",
    sql: "SQL",
    yaml: "YAML",
    yml: "YAML",
    md: "Markdown",
    markdown: "Markdown",
    diff: "Diff",
    patch: "Diff",
    text: "Plain text",
    plaintext: "Plain text",
    txt: "Plain text",
};

function Diff({ text }: { text: string }) {
    return text.split("\n").map((line, index, lines) => {
        if (index === lines.length - 1 && line === "") return null;
        const kind =
            line.startsWith("--- ") ||
            line.startsWith("+++ ") ||
            line.startsWith("diff ") ||
            line.startsWith("index ")
                ? "file"
                : line.startsWith("@@")
                  ? "hunk"
                  : line.startsWith("+")
                    ? "addition"
                    : line.startsWith("-")
                      ? "deletion"
                      : "context";
        return (
            <Fragment key={index}>
                <span className={`diff-line ${kind}`}>{line}</span>
                {index < lines.length - 1 ? "\n" : null}
            </Fragment>
        );
    });
}

export function CodeBlock({ node }: ComponentProps<"pre"> & ExtraProps) {
    const code = node?.children[0];
    const languageClass =
        code?.type === "element"
            ? String(code.properties.className || "")
                  .split(/[ ,]+/)
                  .find((value) => value.startsWith("language-"))
            : undefined;
    const language = languageClass?.slice("language-".length).toLowerCase() || "";
    const title = languageNames[language] || language || "Plain text";
    const text =
        code?.type === "element"
            ? code.children
                  .map((child) => (child.type === "text" ? child.value : ""))
                  .join("")
            : "";
    const lineCount =
        text === "" ? 0 : text.split("\n").length - (text.endsWith("\n") ? 1 : 0);
    const sticky = lineCount > 15;
    const blockRef = useRef<HTMLDivElement>(null);
    const headerRef = useRef<HTMLDivElement>(null);
    useLayoutEffect(() => {
        const block = blockRef.current!;
        const header = headerRef.current!;
        if (!sticky) {
            header.style.removeProperty("--code-bottom-radius");
            return;
        }
        let frame = 0;
        const update = () => {
            frame = 0;
            const remaining =
                block.getBoundingClientRect().bottom -
                header.getBoundingClientRect().bottom;

            header.style.setProperty(
                "--code-bottom-radius",
                `${Math.max(0, Math.min(10, 10 - remaining))}px`,
            );
        };
        const schedule = () => {
            if (!frame) frame = requestAnimationFrame(update);
        };
        const observer = new ResizeObserver(schedule);
        observer.observe(block);
        observer.observe(header);
        window.addEventListener("scroll", schedule, true);
        window.addEventListener("resize", schedule);
        update();
        return () => {
            cancelAnimationFrame(frame);
            observer.disconnect();
            window.removeEventListener("scroll", schedule, true);
            window.removeEventListener("resize", schedule);
        };
    }, [sticky]);
    const diff = language === "diff" || language === "patch";
    const [copied, setCopied] = useState<string | null>(null);
    const { notify } = useNotifications();
    const bridge = usePresentation();
    useEffect(() => {
        if (copied === null) return;
        const timer = setTimeout(() => setCopied(null), 1800);
        return () => clearTimeout(timer);
    }, [copied]);

    return (
        <div className="code-block" data-sticky={sticky} ref={blockRef}>
            <div className="code-header" ref={headerRef}>
                <span className="code-title">{title}</span>
                <button
                    type="button"
                    className="code-copy"
                    aria-label={
                        copied === text ? `Copied ${title} code` : `Copy ${title} code`
                    }
                    title={copied === text ? "Copied" : "Copy code"}
                    onClick={() =>
                        void bridge.copy(text).then(
                            () => setCopied(text),
                            () => notify("Could not copy this code block."),
                        )
                    }
                >
                    {copied === text ? <CheckIcon size={15} /> : <CopyIcon size={15} />}
                </button>
            </div>
            <pre tabIndex={0} aria-label={`${title} code`}>
                {diff ? (
                    <code className="diff-code">
                        <Diff text={text} />
                    </code>
                ) : (
                    <Suspense fallback={<code>{text}</code>}>
                        <SyntaxCode text={text} language={language} />
                    </Suspense>
                )}
            </pre>
        </div>
    );
}
