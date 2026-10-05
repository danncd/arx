import { test } from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createHash } from "node:crypto";
import profile from "../../../main/profile.cjs";

test("profile migration preserves data and credential identity without replacing existing data", (t) => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "arx-profile-test-"));
    t.after(() => fs.rmSync(root, { recursive: true, force: true }));
    const source = path.join(root, "old");
    const target = path.join(root, "new");
    fs.mkdirSync(path.join(source, "attachments"), { recursive: true });
    fs.writeFileSync(path.join(source, "transcript.jsonl"), "saved chat");
    fs.writeFileSync(path.join(source, "attachments", "image"), "image");
    profile.migrateProfile(source, target);
    assert.equal(
        fs.readFileSync(path.join(target, "transcript.jsonl"), "utf8"),
        "saved chat",
    );
    assert.equal(
        fs.readFileSync(path.join(target, "keychain-account"), "utf8"),
        createHash("sha256").update(source).digest("hex"),
    );
    assert.equal(
        fs.readFileSync(path.join(target, "attachments", "image"), "utf8"),
        "image",
    );
    fs.writeFileSync(path.join(target, "transcript.jsonl"), "new chat");
    profile.migrateProfile(source, target);
    assert.equal(
        fs.readFileSync(path.join(target, "transcript.jsonl"), "utf8"),
        "new chat",
    );
    assert.equal(
        fs.readFileSync(path.join(source, "transcript.jsonl"), "utf8"),
        "saved chat",
    );
});

for (const packaged of [false, true]) {
    test(`explicit profile isolates a ${packaged ? "packaged" : "development"} launch`, (t) => {
        const root = fs.mkdtempSync(path.join(os.tmpdir(), "arx-profile-override-"));
        const previous = process.env.ARX_DEV_PROFILE;
        t.after(() => {
            if (previous === undefined) delete process.env.ARX_DEV_PROFILE;
            else process.env.ARX_DEV_PROFILE = previous;
            fs.rmSync(root, { recursive: true, force: true });
        });
        process.env.ARX_DEV_PROFILE = path.join(root, "isolated");
        const paths = {};
        const app = {
            isPackaged: packaged,
            getPath: () => root,
            setPath: (key, value) => (paths[key] = value),
        };
        assert.equal(profile.configureProfile(app), process.env.ARX_DEV_PROFILE);
        assert.equal(paths.userData, paths.sessionData);
        process.env.ARX_DEV_PROFILE = "relative";
        assert.throws(() => profile.configureProfile(app), /absolute/);
    });
}
