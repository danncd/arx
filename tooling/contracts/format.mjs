import prettier from "../../desktop/node_modules/prettier/index.mjs";
let source = "";
for await (const chunk of process.stdin) source += chunk;
process.stdout.write(
    await prettier.format(source, {
        parser: process.argv[2].endsWith(".ts") ? "typescript" : "babel",
        tabWidth: 4,
    }),
);
