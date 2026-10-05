import { readdir, readFile, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { resolve, dirname, extname, relative } from "node:path";
import { spawnSync } from "node:child_process";
import ts from "../../desktop/node_modules/typescript/lib/typescript.js";
const root = fileURLToPath(new URL("../..", import.meta.url));
const strip = process.argv.includes("--strip"),
    failures = [];
const ignored = new Set(["node_modules", ".build", "dist", ".git", "__pycache__"]);
function swiftComments(source) {
    let quote = "";
    for (let i = 0; i < source.length; i++) {
        if (quote) {
            if (source[i] === "\\") i++;
            else if (source.startsWith(quote, i)) {
                i += quote.length - 1;
                quote = "";
            }
        } else if (source.startsWith('"""', i)) {
            quote = '"""';
            i += 2;
        } else if (source[i] === '"') quote = '"';
        else if (source.startsWith("//", i) || source.startsWith("/*", i)) return true;
    }
    return false;
}
async function files(directory) {
    const result = [];
    for (const e of await readdir(directory, { withFileTypes: true })) {
        if (ignored.has(e.name) || e.isSymbolicLink()) continue;
        const path = resolve(directory, e.name);
        if (e.isDirectory()) result.push(...(await files(path)));
        else if (e.isFile()) result.push(path);
    }
    return result;
}
let markdown = 0,
    removed = 0;
for (const file of await files(root)) {
    let source = await readFile(file, "utf8");
    const path = relative(root, file),
        extension = extname(file);
    if (/\.(md|mdx|markdown)$/i.test(extension)) {
        if (path !== "README.md") {
            markdown++;
            if (dirname(path) !== "docs") failures.push(`Markdown outside docs: ${path}`);
        }
        for (const match of source.matchAll(/\]\(([^)]+)\)/g)) {
            const link = match[1].split("#")[0];
            if (
                link &&
                !/^(https?:|codex:|mailto:)/.test(link) &&
                !existsSync(resolve(dirname(file), link))
            )
                failures.push(`Broken doc link: ${path}: ${link}`);
        }
    }
    if (
        /[\\/]?(AGENTS|CLAUDE)\.md$/i.test(path) ||
        path.startsWith("skills/") ||
        path.startsWith("prototypes/")
    )
        failures.push(`Legacy instructions/reference tree: ${path}`);
    if ([".js", ".ts", ".tsx", ".cjs", ".mjs"].includes(extension)) {
        const kind =
            extension === ".tsx"
                ? ts.ScriptKind.TSX
                : extension === ".ts"
                  ? ts.ScriptKind.TS
                  : ts.ScriptKind.JS;
        const ast = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, kind),
            comments = new Map();
        function visit(node) {
            for (const pos of [node.getFullStart(), node.getStart(ast), node.end]) {
                for (const c of [
                    ...(ts.getLeadingCommentRanges(source, pos) || []),
                    ...(ts.getTrailingCommentRanges(source, pos) || []),
                ])
                    comments.set(`${c.pos}:${c.end}`, c);
            }
            ts.forEachChild(node, visit);
        }
        visit(ast);
        if (comments.size && strip) {
            const before = ts.createPrinter({ removeComments: true }).printFile(ast);
            for (const c of [...comments.values()].sort((a, b) => b.pos - a.pos))
                source = source.slice(0, c.pos) + source.slice(c.end);
            const afterAst = ts.createSourceFile(
                file,
                source,
                ts.ScriptTarget.Latest,
                true,
                kind,
            );
            if (ts.createPrinter({ removeComments: true }).printFile(afterAst) !== before)
                throw Error(`Comment removal changed AST: ${path}`);
            await writeFile(file, source);
            removed += comments.size;
        } else if (comments.size) failures.push(`Source comments: ${path}`);
        if (ast.parseDiagnostics.length) failures.push(`Syntax diagnostics: ${path}`);
        if (
            path.startsWith("desktop/renderer/src/") &&
            !path.startsWith("desktop/renderer/src/preview/")
        ) {
            for (const match of source.matchAll(
                /(?:from\s*|import\s*\()(["'])([^"']+)\1/g,
            )) {
                if (!match[2].startsWith(".")) continue;
                const target = relative(root, resolve(dirname(file), match[2]));
                if (
                    target.startsWith("desktop/renderer/src/preview/") ||
                    (path.startsWith("desktop/renderer/src/features/") &&
                        target.startsWith("desktop/renderer/src/app/"))
                )
                    failures.push(`Feature imports composition/preview: ${path}`);
                if (
                    path.startsWith("desktop/renderer/src/platform/") &&
                    target.startsWith("desktop/renderer/src/features/")
                )
                    failures.push(`Platform imports feature: ${path}`);
            }
        }
    }
    if (extension === ".mod" && /\/\/ indirect/.test(source)) {
        if (strip) await writeFile(file, source.replace(/ \/\/ indirect/g, ""));
        else failures.push(`Module comments: ${path}`);
    }
    if (extension === ".swift" && swiftComments(source))
        failures.push(`Source comments: ${path}`);
    if ([".css", ".html"].includes(extension)) {
        const pattern = extension === ".html" ? /<!--[\s\S]*?-->/g : /\/\*[\s\S]*?\*\//g;
        if (pattern.test(source)) failures.push(`Source comments: ${path}`);
    }
}
if (markdown !== 11) failures.push(`Expected eleven docs, found ${markdown}`);
const go = spawnSync(
    process.env.ARX_GO_BINARY || "go",
    [
        "run",
        fileURLToPath(new URL("check.go", import.meta.url)),
        root,
        ...(strip ? ["--strip"] : []),
    ],
    { stdio: "inherit" },
);
if (go.error) throw go.error;
if (go.status) process.exit(go.status);
const python = spawnSync(
    process.env.ARX_PYTHON_BINARY || "python3",
    ["-B", fileURLToPath(new URL("check.py", import.meta.url)), root],
    { stdio: "inherit" },
);
if (python.error) throw python.error;
if (python.status) process.exit(python.status);
if (failures.length) throw Error(failures.join("\n"));
console.log(
    `Documentation/dependency/source policy passed; removed ${removed} JS/TS comments`,
);
