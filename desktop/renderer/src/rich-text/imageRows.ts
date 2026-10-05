import type { Element, Root, RootContent } from "hast";

function whitespace(node: RootContent) {
    return (
        (node.type === "text" && !node.value.trim()) ||
        (node.type === "element" && node.tagName === "br")
    );
}

function imagesIn(node: RootContent): Element[] | null {
    if (node.type !== "element") return null;
    if (node.tagName === "img") return [node];
    if (node.tagName !== "p" && node.tagName !== "a") return null;
    const children = node.children.filter((child) => !whitespace(child));
    if (!children.length) return null;
    const images: Element[] = [];
    for (const child of children) {
        const found = imagesIn(child);
        if (!found) return null;
        images.push(...found);
    }
    return node.tagName === "a" ? [node] : images;
}

export function imageRows() {
    return (root: Root) => {
        const visit = (parent: Root | Element) => {
            const children: RootContent[] = [];
            for (let index = 0; index < parent.children.length; index++) {
                const node = parent.children[index]!;
                const images = imagesIn(node);
                if (!images) {
                    if (node.type === "element") visit(node);
                    children.push(node);
                    continue;
                }
                let next = index + 1;
                while (next < parent.children.length) {
                    if (whitespace(parent.children[next]!)) {
                        next++;
                        continue;
                    }
                    const found = imagesIn(parent.children[next]!);
                    if (!found) break;
                    images.push(...found);
                    index = next++;
                }
                children.push({
                    type: "element",
                    tagName: "span",
                    properties: {
                        className: ["image-row"],
                        role: "region",
                        ariaLabel: "Images",
                        tabIndex: 0,
                    },
                    children: images,
                });
            }
            parent.children = children;
        };
        visit(root);
    };
}
