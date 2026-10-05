class ViewWrites {
    constructor(save, delay = 200) {
        this.save = save;
        this.delay = delay;
        this.pending = new Map();
        this.waiters = [];
        this.timer = null;
        this.inflight = Promise.resolve();
    }

    set(key, value) {
        if (typeof key !== "string" || typeof value !== "string")
            return Promise.reject(new Error("Invalid view preference"));
        this.pending.set(key, value);
        const result = new Promise((resolve, reject) => {
            this.waiters.push({ resolve, reject });
        });
        if (!this.timer) {
            this.timer = setTimeout(() => void this.flush().catch(() => {}), this.delay);
        }
        return result;
    }

    flush() {
        clearTimeout(this.timer);
        this.timer = null;
        if (!this.pending.size) return this.inflight.catch(() => {});
        const values = Object.fromEntries(this.pending);
        const waiters = this.waiters;
        this.pending.clear();
        this.waiters = [];
        const task = this.inflight.catch(() => {}).then(() => this.save(values));
        this.inflight = task;
        void task.then(
            () => waiters.forEach(({ resolve }) => resolve({})),
            (error) => waiters.forEach(({ reject }) => reject(error)),
        );
        return task;
    }
}

module.exports = { ViewWrites };
