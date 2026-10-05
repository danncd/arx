import { packager } from "@electron/packager";
import { execFile } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";

if (process.platform !== "darwin") throw new Error("macOS packaging requires a Mac");

const run = promisify(execFile);
const desktop = fileURLToPath(new URL("../", import.meta.url));
const project = fileURLToPath(new URL("../../", import.meta.url));
const metadata = JSON.parse(await readFile(join(desktop, "package.json"), "utf8"));
const temporary = await mkdtemp(join(tmpdir(), "arx-package-"));
const stage = join(temporary, "app");
const iconset = join(temporary, "Arx.iconset");
const icon = join(temporary, "Arx.icns");

try {
    await mkdir(join(stage, ".build"), { recursive: true });
    await mkdir(iconset);
    for (const folder of ["main", "preload", "contracts"]) {
        await cp(join(desktop, folder), join(stage, folder), { recursive: true });
    }
    await cp(join(desktop, ".build", "renderer"), join(stage, ".build", "renderer"), {
        recursive: true,
    });
    await writeFile(
        join(stage, "package.json"),
        JSON.stringify({
            name: "arx",
            productName: metadata.productName,
            version: metadata.version,
            main: metadata.main,
        }),
    );

    const source = join(desktop, "renderer", "public", "arx.png");
    const opaque = join(temporary, "Arx.png");
    await run("/usr/bin/xcrun", [
        "swift",
        join(desktop, "scripts", "icon-background.swift"),
        source,
        opaque,
    ]);
    for (const size of [16, 32, 128, 256, 512]) {
        await run("/usr/bin/sips", [
            "-z",
            String(size),
            String(size),
            opaque,
            "--out",
            join(iconset, `icon_${size}x${size}.png`),
        ]);
        await run("/usr/bin/sips", [
            "-z",
            String(size * 2),
            String(size * 2),
            opaque,
            "--out",
            join(iconset, `icon_${size}x${size}@2x.png`),
        ]);
    }
    await run("/usr/bin/iconutil", ["-c", "icns", iconset, "-o", icon]);

    const output = await packager({
        dir: stage,
        out: process.env.ARX_PACKAGE_OUT || join(project, "dist"),
        name: metadata.productName,
        platform: "darwin",
        arch: process.arch,
        electronVersion: metadata.devDependencies.electron,
        appBundleId: "com.danncd.arx",
        extendInfo: { CFBundleIconFile: "Arx.icns" },
        icon,
        asar: true,
        extraResource: join(desktop, ".build", "backend"),
        overwrite: true,
        quiet: true,
    });
    for (const directory of output) {
        const app = join(directory, `${metadata.productName}.app`);
        await run("/usr/bin/codesign", ["--force", "--deep", "--sign", "-", app]);
        await run("/usr/bin/codesign", ["--verify", "--deep", "--strict", app]);
    }
    process.stdout.write(`${output.join("\n")}\n`);
} finally {
    await rm(temporary, { recursive: true, force: true });
}
