export const previewPrompt = "Show me how Markdown looks in Arx.";

export const previewResponse = `## A simple place to work

This is a local preview of the original Arx Markdown renderer.

### The useful details

- Read and edit project files.
- Look up documentation on the web.
- Run commands and review their output.

Use **bold text**, *emphasis*, and \`inline code\` when they make an answer easier to read.

> Keep the interface quiet and let the work take the space.

### Code

\`\`\`typescript
function greet(name: string) {
    return \"Hello, \" + name;
}

const message = greet(\"Danny\");
console.log(message);
\`\`\`

### Changes

\`\`\`diff
- const title = \"Untitled\";
+ const title = \"Arx\";
\`\`\`

### Tools

| Tool | Purpose |
| --- | --- |
| Files | Read, search, and edit |
| Web | Search and read pages |
| Bash | Run commands |

---

- [x] Desktop shell
- [ ] Agent connection

This conversation is sample content; no model was called.
`;
