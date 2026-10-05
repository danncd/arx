import { memo, type ReactNode } from "react";
import type { RootContent } from "hast";
import { createLowlight } from "lowlight";
import bash from "highlight.js/lib/languages/bash";
import c from "highlight.js/lib/languages/c";
import cpp from "highlight.js/lib/languages/cpp";
import css from "highlight.js/lib/languages/css";
import go from "highlight.js/lib/languages/go";
import java from "highlight.js/lib/languages/java";
import javascript from "highlight.js/lib/languages/javascript";
import json from "highlight.js/lib/languages/json";
import markdown from "highlight.js/lib/languages/markdown";
import python from "highlight.js/lib/languages/python";
import rust from "highlight.js/lib/languages/rust";
import sql from "highlight.js/lib/languages/sql";
import swift from "highlight.js/lib/languages/swift";
import typescript from "highlight.js/lib/languages/typescript";
import xml from "highlight.js/lib/languages/xml";
import yaml from "highlight.js/lib/languages/yaml";

const highlighter = createLowlight({
    bash,
    c,
    cpp,
    css,
    go,
    java,
    javascript,
    json,
    markdown,
    python,
    rust,
    sql,
    swift,
    typescript,
    xml,
    yaml,
});
highlighter.registerAlias({ bash: ["shell", "zsh"], json: ["jsonc"] });

function renderTokens(nodes: RootContent[]): ReactNode {
    return nodes.map((node, index) => {
        if (node.type === "text") return node.value;
        if (node.type !== "element") return null;
        const classes = node.properties.className;
        return (
            <span
                key={index}
                className={Array.isArray(classes) ? classes.join(" ") : undefined}
            >
                {renderTokens(node.children)}
            </span>
        );
    });
}

export const SyntaxCode = memo(function SyntaxCode({
    text,
    language,
}: {
    text: string;
    language: string;
}) {
    let content: ReactNode = text;
    if (highlighter.registered(language)) {
        try {
            content = renderTokens(highlighter.highlight(language, text).children);
        } catch {
            content = text;
        }
    }
    return <code>{content}</code>;
});
