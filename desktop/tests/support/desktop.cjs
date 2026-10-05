const { _electron: electron } = require("@playwright/test");
const path = require("node:path");
function desktopEnvironment(profile) {
    const env = {
        ...process.env,
        ELECTRON_RUN_AS_NODE: "",
        ARX_DEV_PROFILE: profile,
        ARX_MODELS_DIR: path.join(profile, "models"),
    };
    delete env.ARX_DEV_URL;
    return env;
}
function launchDesktop(project, env) {
    return electron.launch({ args: [project], env });
}
module.exports = { desktopEnvironment, launchDesktop };
