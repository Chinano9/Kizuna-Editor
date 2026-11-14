import { test, describe } from "node:test";
import assert from "node:assert";
import * as alphaTab from "@coderline/alphatab";
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

function assertValidAlphaTex(tex: string) {
    const importer = new alphaTab.importer.AlphaTexImporter();
    importer.initFromString(tex, new alphaTab.Settings());
    const score = importer.readScore();
    assert.ok(score && score.tracks.length > 0, `Expected valid parsed score for: ${tex}`);
    return score;
}

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
        // Should not have double bars like "||" or "| |"
        assert.ok(!output.includes("||"), `Found duplicate bar lines: "${output}"`);
        assert.ok(!/\|\s+\|/.test(output), `Found spaced duplicate bar lines: "${output}"`);
    });

    test("Keeps note/beat effects {...} attached before injecting bar lines", () => {
        const input = '\\title "Demo"\n\\tempo 120\n.\n:4 0.6 2.6 3.6 5.6{v} 0.5 2.5 3.5 5.5{b (0 4)}';
        const prepared = prepareAlphaTexForRender(input, true);
        assert.ok(
            !prepared.includes("| {v}") && !prepared.includes("| { b"),
            `Bar line was injected before effect block: "${prepared}"`,
        );
        const score = assertValidAlphaTex(prepared);
        assert.strictEqual(score.masterBars.length, 2);
    });

    test("Supports pasted AlphaTex with fret.string.duration syntax and chords", () => {
        const input = "3.3.4 5.3.4 7.3.8 5.3.8 3.3.4 (0.1 1.2).2 (0.1 1.2).2";
        assert.strictEqual(looksLikeAlphaTex(input), true);
        const prepared = prepareAlphaTexForRender(input, true);
        const score = assertValidAlphaTex(prepared);
        assert.strictEqual(score.masterBars.length, 2);
    });

    test("Supports single-line pasted AlphaTex with inline '.' separator", () => {
        const input = '\\title "Mi Canción" \\tempo 120 . :4 0.6 2.6 3.6 5.6 0.5 2.5 3.5 5.5';
        assert.strictEqual(looksLikeAlphaTex(input), true);
        const prepared = prepareAlphaTexForRender(input, true);
        const score = assertValidAlphaTex(prepared);
        assert.strictEqual(score.masterBars.length, 2);
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

    test("Returns empty string for header-only AlphaTex template so no fake measure renders by default", () => {
        const prepared = prepareAlphaTexForRender('\\title "New Song"\n.', true);
        assert.strictEqual(prepared, "");
    });

    test("Normalizes TeX % comments and block chords in pasted AlphaTex (Lo-Fi Jazz Vibes)", () => {
        const pasted = `\\title "New Song"
.
\\title "Lo-Fi Jazz Vibes"
\\subtitle "Prueba de Acordes en Bloque"
\\tempo 80
\\ts 4 4
\\track "Guitarra Eléctrica (Limpia)"
\\instrument 26 
.
% Compás 1: Am7 y Dm7 (Notas Blancas)
(0.1 1.2 0.3 2.4 0.5).2  (1.1 1.2 2.3 0.4).2 |

% Compás 2: G7 y Cmaj7 (Notas Blancas)
(1.1 0.2 0.3 0.4 2.5 3.6).2 (0.1 0.2 0.3 2.4 3.5).2 |`;

        assert.strictEqual(looksLikeAlphaTex(pasted), true);
        const prepared = prepareAlphaTexForRender(pasted, true);
        const score = assertValidAlphaTex(prepared);
        assert.strictEqual(score.masterBars.length, 2);

        const playable = parsePlayableNotesFromAlphaTex(pasted, 120);
        assert.strictEqual(playable.events.length, 4);
        assert.strictEqual(playable.events[0].midiNotes.length, 5);
        assert.strictEqual(playable.events[1].midiNotes.length, 4);
        assert.strictEqual(playable.events[2].midiNotes.length, 6);
        assert.strictEqual(playable.events[3].midiNotes.length, 5);
    });

    test("Normalizes dotted chord durations (.4., .2.) without '.' separator (Smoke on the Water)", () => {
        const smoke = `\\title "Smoke on the Water"
\\artist "Deep Purple"
\\tempo 112
\\track "Guitarra Eléctrica"
\\instrument 30

(0.4 0.3).4 (3.4 3.3).4 (5.4 5.3).4. { d } (0.4 0.3).8 |
(3.4 3.3).4 (6.4 6.3).8 (5.4 5.3).2 r.8 |
(0.4 0.3).4 (3.4 3.3).4 (5.4 5.3).4. { d } (3.4 3.3).8 |
(0.4 0.3).2. r.4 |`;

        const preparedAuto = prepareAlphaTexForRender(smoke, true);
        const scoreAuto = assertValidAlphaTex(preparedAuto);
        assert.strictEqual(scoreAuto.title, "Smoke on the Water");
        assert.strictEqual(scoreAuto.masterBars.length, 4);

        const preparedRaw = prepareAlphaTexForRender(`\\title "New Song"\n.\n${smoke}`, false);
        const scoreRaw = assertValidAlphaTex(preparedRaw);
        assert.strictEqual(scoreRaw.title, "Smoke on the Water");
        assert.strictEqual(scoreRaw.masterBars.length, 4);

        const playable = parsePlayableNotesFromAlphaTex(smoke, 120);
        assert.strictEqual(playable.events.length, 12);
    });

    test("Normalizes dotted note durations (7.5.4., 10.5.8. { d }) without '.' separator (Seven Nation Army)", () => {
        const sevenNation = `\\title "Seven Nation Army"
\\artist "The White Stripes"
\\tempo 124
\\track "Guitarra / Bajo"
\\instrument 29

7.5.4. 7.5.8 10.5.8. { d } 7.5.8. { d } 5.5.8 |
3.5.2 2.5.2 |
7.5.4. 7.5.8 10.5.8. { d } 7.5.8. { d } 5.5.8 |
3.5.8. { d } 5.5.8. { d } 3.5.8 2.5.2 |`;

        const preparedAuto = prepareAlphaTexForRender(sevenNation, true);
        const scoreAuto = assertValidAlphaTex(preparedAuto);
        assert.strictEqual(scoreAuto.title, "Seven Nation Army");
        assert.strictEqual(scoreAuto.masterBars.length, 4);

        const playable = parsePlayableNotesFromAlphaTex(sevenNation, 120);
        assert.strictEqual(playable.events.length, 16);
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
