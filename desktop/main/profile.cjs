const path = require("node:path");
const fs = require("node:fs");
const { createHash } = require("node:crypto");

function migrateProfile(source, target) {
    if (fs.existsSync(target) || !fs.existsSync(source)) return;
    fs.mkdirSync(path.dirname(target), { recursive: true, mode: 0o700 });
    const stage = fs.mkdtempSync(path.join(path.dirname(target), ".arx-profile-"));
    try {
        fs.cpSync(source, stage, {
            recursive: true,
            filter: (file) => !path.basename(file).startsWith("Singleton"),
        });
        const account = path.join(stage, "keychain-account");
        if (!fs.existsSync(account)) {
            fs.writeFileSync(account, createHash("sha256").update(source).digest("hex"), {
                mode: 0o600,
            });
        }
        fs.renameSync(stage, target);
    } finally {
        fs.rmSync(stage, { recursive: true, force: true });
    }
}

function configureProfile(app) {
    const explicit = process.env.ARX_DEV_PROFILE;
    const directory =
        explicit ||
        path.join(app.getPath("appData"), app.isPackaged ? "Arx" : "Arx Development");
    if (!path.isAbsolute(directory)) throw new Error("Profile path must be absolute");
    if (!app.isPackaged && !explicit) {
        migrateProfile(path.join(app.getAppPath(), ".build", "shell-profile"), directory);
    }
    fs.mkdirSync(directory, { recursive: true, mode: 0o700 });
    app.setPath("userData", directory);
    app.setPath("sessionData", directory);
    return directory;
}

module.exports = { configureProfile, migrateProfile };
