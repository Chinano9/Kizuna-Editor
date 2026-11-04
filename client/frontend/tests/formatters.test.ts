import { test, describe } from "node:test";
import assert from "node:assert";
import { formatTime, formatBytes, formatDate } from "../src/shared/lib/formatters.ts";

describe("Formatters Library", () => {
    test("formatTime formats seconds to mm:ss correctly", () => {
        assert.strictEqual(formatTime(0), "00:00");
        assert.strictEqual(formatTime(65), "01:05");
        assert.strictEqual(formatTime(3600), "60:00");
    });

    test("formatTime formats seconds with milliseconds", () => {
        assert.strictEqual(formatTime(65.4, true), "01:05.4");
        assert.strictEqual(formatTime(-1, true), "00:00.0");
    });

    test("formatBytes formats byte sizes", () => {
        assert.strictEqual(formatBytes(0), "0 B");
        assert.strictEqual(formatBytes(1024), "1 KB");
        assert.strictEqual(formatBytes(1024 * 1024 * 2.5), "2.5 MB");
    });

    test("formatDate handles ISO strings and empty values", () => {
        assert.strictEqual(formatDate(""), "");
        const formatted = formatDate("2026-11-03T00:00:00Z");
        assert.ok(formatted.length > 0);
    });
});
