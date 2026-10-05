const { EventEmitter } = require("node:events");
const { spawn } = require("node:child_process");
const { ViewWrites } = require("./view-writes.cjs");
const { Client } = require("./client.cjs");

class Backend extends EventEmitter {
    constructor(binary, directory) {
        super();
        this.binary = binary;
        this.directory = directory;
        this.child = null;
        this.client = null;
        this.status = { state: "stopped", error: null };
        this.stopping = null;
        this.views = new ViewWrites((views) =>
            this.client.request("save_views", { views }),
        );
    }

    update(state, error = null) {
        this.status = { state, error };
        this.emit("status", this.status);
    }

    start() {
        if (this.child) return;
        this.update("starting");
        const child = spawn(this.binary, [], {
            stdio: ["pipe", "pipe", "ignore"],
            env: { ...process.env, ARX_STATE: this.directory || "" },
        });
        this.child = child;
        this.client = new Client(child.stdin, child.stdout);
        this.client.on("permission", (request) => this.emit("permission", request));
        this.client.on("chat", (event) => this.emit("chat", event));
        this.client.on("generation", (event) => this.emit("generation", event));
        this.client.on("local", (event) => this.emit("local", event));
        const timer = setTimeout(() => {
            this.update("error", "Backend did not start");
            child.kill();
        }, 5000);
        this.client.once("ready", () => {
            clearTimeout(timer);
            if (this.status.state === "starting") this.update("ready");
        });
        this.client.once("disconnected", (error) => {
            clearTimeout(timer);
            if (!["stopping", "error"].includes(this.status.state)) {
                this.update("error", error.message);
            }
            child.kill();
        });
        child.once("error", () => {
            clearTimeout(timer);
            this.update("error", "Could not start backend");
        });
        child.once("close", () => {
            clearTimeout(timer);
            this.client.close();
            this.child = null;
            this.client = null;
            if (this.status.state === "stopping") this.update("stopped");
            else if (this.status.state !== "error")
                this.update("error", "Backend stopped unexpectedly");
        });
    }

    async request(method, params, timeout) {
        if (this.status.state !== "ready" || !this.client) {
            return Promise.reject(new Error("Backend is not connected"));
        }
        if (method === "save_view") return this.views.set(params?.key, params?.value);
        await this.views.flush();
        return this.client.request(method, params, timeout);
    }

    stop() {
        if (this.stopping) return this.stopping;
        if (!this.child) {
            this.update("stopped");
            return Promise.resolve();
        }
        const child = this.child;
        this.update("stopping");
        this.stopping = new Promise((resolve) => {
            const timer = setTimeout(() => child.kill("SIGKILL"), 6000);
            child.once("close", () => {
                clearTimeout(timer);
                resolve();
            });
            void this.views
                .flush()
                .catch((error) => this.emit("persistence-error", error))
                .then(() => this.client.request("shutdown", {}, 5000))
                .catch(() => child.kill());
        }).finally(() => {
            this.stopping = null;
        });
        return this.stopping;
    }

    async restart() {
        await this.stop();
        this.start();
        return this.status;
    }
}

module.exports = { Backend };
