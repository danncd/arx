const { EventEmitter } = require("node:events");

const maxMessageBytes = 1 << 20;
const maxResponseBytes = 16 << 20;

class Client extends EventEmitter {
    constructor(input, output) {
        super();
        this.input = input;
        this.pending = new Map();
        this.nextID = 0;
        this.buffer = "";
        this.closed = false;
        output.setEncoding("utf8");
        output.on("data", (chunk) => this.receive(chunk));
        output.on("end", () => this.close(new Error("Backend disconnected")));
        output.on("error", (error) => this.close(error));
        input.on("error", (error) => this.close(error));
    }

    request(method, params = {}, timeout = 5000) {
        if (this.closed) return Promise.reject(new Error("Backend disconnected"));
        const id = String(++this.nextID);
        const message = JSON.stringify({ id, method, params }) + "\n";
        if (Buffer.byteLength(message) > maxMessageBytes) {
            return Promise.reject(new Error("Request is too large"));
        }
        return new Promise((resolve, reject) => {
            const timer =
                timeout > 0
                    ? setTimeout(() => {
                          this.pending.delete(id);
                          reject(new Error("Backend request timed out"));
                      }, timeout)
                    : undefined;
            this.pending.set(id, { resolve, reject, timer });
            this.input.write(message, (error) => {
                if (!error) return;
                this.close(error);
            });
        });
    }

    receive(chunk) {
        if (this.closed) return;
        this.buffer += chunk;
        let end;
        while ((end = this.buffer.indexOf("\n")) !== -1) {
            const line = this.buffer.slice(0, end);
            this.buffer = this.buffer.slice(end + 1);
            if (Buffer.byteLength(line) > maxResponseBytes) {
                this.close(new Error("Backend response is too large"));
                return;
            }
            try {
                const message = JSON.parse(line);
                if (!message || typeof message !== "object") throw Error();
                if (message.event === "ready" && message.data?.protocol === 1) {
                    this.emit("ready");
                } else if (
                    ["permission", "chat", "local", "generation"].includes(
                        message.event,
                    ) &&
                    "data" in message
                ) {
                    this.emit(message.event, message.data);
                } else if (
                    typeof message.id === "string" &&
                    "result" in message !== "error" in message
                ) {
                    const pending = this.pending.get(message.id);
                    if (!pending) continue;
                    this.pending.delete(message.id);
                    clearTimeout(pending.timer);
                    if (message.error) {
                        pending.reject(
                            new Error(message.error.message || "Backend request failed"),
                        );
                    } else {
                        pending.resolve(message.result);
                    }
                } else {
                    throw Error();
                }
            } catch {
                this.close(new Error("Invalid backend response"));
                return;
            }
        }
        if (Buffer.byteLength(this.buffer) > maxResponseBytes) {
            this.close(new Error("Backend response is too large"));
        }
    }

    close(error = new Error("Backend disconnected")) {
        if (this.closed) return;
        this.closed = true;
        for (const pending of this.pending.values()) {
            clearTimeout(pending.timer);
            pending.reject(error);
        }
        this.pending.clear();
        this.buffer = "";
        this.emit("disconnected", error);
    }
}

module.exports = { Client };
