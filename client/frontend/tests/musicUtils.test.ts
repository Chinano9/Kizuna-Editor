import { test, describe } from "node:test";
import assert from "node:assert";
import {
    injectBars,
    looksLikeAlphaTex,
    prepareAlphaTexForRender,
    extractScoreMetadata,
    calculatePlaybackSyncState,
    getTimestampForBar,
    parsePlayableNotesFromAlphaTex,
} from "../src/shared/lib/musicUtils.ts";
import { audioPlayerStore } from "../src/features/audio-deck/stores/audioPlayerStore.ts";
import { get } from "svelte/store";

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

describe("Music Utilities - AlphaTex Detection & Render Preparation", () => {
    test("Detects standard AlphaTex directives and default templates", () => {
        assert.strictEqual(looksLikeAlphaTex('\\title "New Song"\n.'), true);
        assert.strictEqual(looksLikeAlphaTex("\\tempo 120\n.\n:4 0.6 2.6 3.6 5.6"), true);
        assert.strictEqual(looksLikeAlphaTex("0.6 2.6 3.5 5.5"), true);
        assert.strictEqual(looksLikeAlphaTex("Plain lyrics without notation"), false);
    });

    test("Prepares header-only AlphaTex template with initial rest measure", () => {
        const prepared = prepareAlphaTexForRender('\\title "New Song"\n.', true);
        assert.ok(prepared.includes(":1 r |"), `Expected rest measure in: "${prepared}"`);
    });
});

describe("Music Utilities - Score Synchronization & Playback Timeline", () => {
    test("Extracts tempo, time signature, and bar count from AlphaTex", () => {
        const source = '\\title "Demo"\n\\tempo 120\n\\ts 4 4\n.\n:4 0.6 2.6 3.6 5.6 | :4 0.5 2.5 3.5 5.5 |';
        const meta = extractScoreMetadata(source);
        assert.strictEqual(meta.bpm, 120);
        assert.deepStrictEqual(meta.timeSignature, { num: 4, den: 4 });
        assert.strictEqual(meta.barCount, 2);
        assert.strictEqual(meta.secondsPerBar, 2);
        assert.strictEqual(meta.estimatedDurationSeconds, 4);
    });

    test("Calculates synchronized bar and beat from audio playback position", () => {
        // At 120 BPM in 4/4, each beat is 0.5s and each bar is 2.0s
        const syncAtStart = calculatePlaybackSyncState(0, 4, 2, 120, { num: 4, den: 4 });
        assert.strictEqual(syncAtStart.currentBarIndex, 0);
        assert.strictEqual(syncAtStart.currentBeat, 1);

        const syncSecondBar = calculatePlaybackSyncState(2.5, 4, 2, 120, { num: 4, den: 4 });
        assert.strictEqual(syncSecondBar.currentBarIndex, 1);
        assert.strictEqual(syncSecondBar.currentBeat, 2);
        assert.strictEqual(getTimestampForBar(1, 120, { num: 4, den: 4 }), 2);
    });

    test("Parses playable note events from AlphaTex for real-time synthesis", () => {
        const source = "\\tempo 120\n.\n:4 0.6 2.6 (0.1 1.2) r";
        const parsed = parsePlayableNotesFromAlphaTex(source, 120);
        assert.strictEqual(parsed.events.length, 3);
        assert.strictEqual(parsed.events[0].startTimeSeconds, 0);
        assert.strictEqual(parsed.events[1].startTimeSeconds, 0.5);
        assert.strictEqual(parsed.events[2].midiNotes.length, 2);
        assert.ok(parsed.durationSeconds >= 2);
    });

    test("audioPlayerStore coordinates seek and score sync state", async () => {
        audioPlayerStore.reset();
        let seekTarget = -1;
        audioPlayerStore.registerSeekHandler((secs) => {
            seekTarget = secs;
        });

        await audioPlayerStore.requestSeek(3.5);
        audioPlayerStore.setScoreSync(1, 4, 4, 120);

        const state = get(audioPlayerStore);
        assert.strictEqual(seekTarget, 3.5);
        assert.strictEqual(state.positionSeconds, 3.5);
        assert.strictEqual(state.currentBar, 1);
        assert.strictEqual(state.currentBeat, 4);
        assert.strictEqual(state.totalBars, 4);
        assert.strictEqual(state.bpm, 120);
        audioPlayerStore.reset();
    });
});
