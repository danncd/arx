const { test } = require("node:test");
const assert = require("node:assert/strict");
const { PassThrough } = require("node:stream");
const { Client } = require("./client.cjs");

function connect() {
    const input = new PassThrough();
    const output = new PassThrough();
    return { input, output, client: new Client(input, output) };
}

test("matches fragmented, out-of-order responses", async () => {
    const { client, output } = connect();
    const first = client.request("status");
    const second = client.request("status");
    output.write('{"id":"2","res');
    output.write('ult":{"version":"two"}}\n{"id":"1","result":{"version":"one"}}\n');
    assert.deepEqual(await first, { version: "one" });
    assert.deepEqual(await second, { version: "two" });
    client.close();
});

test("rejects pending requests when the stream closes", async () => {
    const { client, output } = connect();
    const pending = assert.rejects(client.request("status"), /disconnected/);
    output.end();
    await pending;
    assert.equal(client.pending.size, 0);
});

test("rejects invalid protocol and oversized messages", async () => {
    for (const message of [
        "not json\n",
        '{"event":"ready","data":{"protocol":99}}\n',
        "x".repeat((16 << 20) + 1),
    ]) {
        const { client, output } = connect();
        const pending = assert.rejects(
            client.request("status"),
            /Invalid backend|too large/,
        );
        output.write(message);
        await pending;
        assert.equal(client.closed, true);
    }
});

test("cleans up timed out requests and ignores late responses", async () => {
    const { client, output } = connect();
    await assert.rejects(client.request("status", {}, 10), /timed out/);
    output.write('{"id":"1","result":{}}\n');
    assert.equal(client.closed, false);
    assert.equal(client.pending.size, 0);
    client.close();
});

test("generation progress does not interrupt pending requests", async () => {
    const { client, output } = connect();
    const events = [];
    client.on("generation", (event) => events.push(event));
    const pending = client.request("generation.state");
    output.write('{"event":"generation","data":{"job":{"state":"running"}}}\n');
    output.write('{"id":"1","result":{"jobs":[]}}\n');
    assert.deepEqual(await pending, { jobs: [] });
    assert.equal(events[0].job.state, "running");
    assert.equal(client.closed, false);
    client.close();
});
