import { test, describe } from "node:test";
import assert from "node:assert";
import { injectBars } from "../src/shared/lib/musicUtils.ts";

describe("Music Utilities - AlphaTex Bar Injection", () => {
    test("Passes through metadata lines unchanged", () => {
        const input = '\\title "My Song"\n\\tempo 120';
        const output = injectBars(input);
        assert.strictEqual(output, input);
    });

    test("Injects bar after 4 quarter notes in 4/4 time", () => {
        const input = "\\ts 4 4\n.\n:4 0 0 0 0";
        const output = injectBars(input);
        assert.ok(output.includes("|"), `Expected bar line in output: "${output}"`);
    });

    test("Handles existing bar lines without duplication", () => {
        const input = "\\ts 4 4\n.\n:4 0 0 0 0 | :4 0 0 0 0 |";
        const output = injectBars(input);
        assert.ok(output.includes("|"));
        // Should not have double bars like "||"
        assert.ok(!output.includes("||"), `Found duplicate bar lines: "${output}"`);
    });

    test("Handles empty or null input gracefully", () => {
        assert.strictEqual(injectBars(""), "");
        // @ts-ignore
        assert.strictEqual(injectBars(null), "");
    });
});
