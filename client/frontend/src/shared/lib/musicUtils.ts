/*
    Kizuna Editor - A local-first songwriting environment.
    Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/**
 * musicUtils.ts
 * Utility functions for parsing and manipulating AlphaTex strings.
 * Primary focus: Automatic bar line injection based on rhythm calculation.
 */

// --- CONSTANTS & REGEX ---

const REGEX_NOTE = /^(\d+|r)/;
const REGEX_TS = /\\ts\s+(\d+)\s+(\d+)/;
// const REGEX_TUPLET_START = /\\tuplet\s+(\d+)/; // Reserved for future use

// Distinguishes Guitar Coordinates (Fret.String, e.g., "3.5") from Dotted Durations (e.g., "2.")
const REGEX_GUITAR_COORD = /^\d+\.\d+$/;

// --- PARSING HELPERS ---

/**
 * Extracts duration value from a token (e.g., ":4", ":8.").
 * Handles dotted notes.
 */
function parseDuration(token: string): number | null {
    if (token.startsWith(':')) {
        const hasDot = token.includes('.');
        // Remove ':' and '.' to parse the base number
        const cleanToken = token.substring(1).replace('.', '');
        const val = parseInt(cleanToken);

        if (!isNaN(val) && val !== 0) {
            let duration = 1 / val;
            if (hasDot) duration *= 1.5;
            return duration;
        }
    }
    return null;
}

/**
 * Extracts Time Signature from a line (e.g., "\ts 4 4").
 */
function parseTimeSignature(line: string): { num: number, den: number } | null {
    const match = line.match(REGEX_TS);
    if (match) {
        return { num: parseInt(match[1]), den: parseInt(match[2]) };
    }
    return null;
}

/**
 * Calculates the duration ratio for tuplets.
 * Logic based on standard notation conventions.
 */
function getTupletRatio(num: number): number {
    // 3 notes in the time of 2
    if (num > 2 && num <= 3) return 2 / num;
    // 5-7 notes in the time of 4
    if (num > 3 && num <= 7) return 4 / num;
    // 9+ notes usually fit in the time of 8
    if (num > 7) return 8 / num;
    return 1;
}

// --- STATE MANAGEMENT ---

/**
 * Tracks the rhythmic accumulation of the current measure.
 */
class RhythmState {
    private currentDuration: number = 0.25; // Default: Quarter note (1/4)
    private accumulator: number = 0;
    private limit: number = 1.0; // Default: 4/4 time signature

    public inChord: boolean = false;
    public inTuplet: boolean = false;
    private tupletRatio: number = 1;

    constructor() { }

    public setTimeSignature(num: number, den: number) {
        this.limit = num * (1 / den);
        this.resetAccumulator();
    }

    public setDuration(val: number) {
        this.currentDuration = val;
    }

    public startTuplet(n: number) {
        this.tupletRatio = getTupletRatio(n);
        this.inTuplet = true;
    }

    public endTuplet() {
        this.inTuplet = false;
        this.tupletRatio = 1;
    }

    public addNote(token: string) {
        // Durations inside a chord stack do not advance the timeline
        if (this.inChord) return;

        let duration = this.currentDuration;

        // CRITICAL CHECK: Dotted Note vs. Guitar String Coordinate
        // "3.5" means Fret 3 on String 5 (No duration change)
        // "r." or ":4." means Dotted duration
        const isGuitarCoord = REGEX_GUITAR_COORD.test(token);

        if (!isGuitarCoord && token.includes('.') && !token.includes(',')) {
            duration *= 1.5;
        }

        if (this.inTuplet) {
            duration *= this.tupletRatio;
        }

        this.accumulator += duration;
    }

    public shouldInsertBar(): boolean {
        // Never insert a bar line in the middle of a chord or tuplet group
        if (this.inChord || this.inTuplet) return false;

        // Use epsilon for floating point comparison stability
        if (this.accumulator >= (this.limit - 0.001)) {
            this.accumulator = 0;
            return true;
        }
        return false;
    }

    public resetAccumulator() {
        this.accumulator = 0;
        this.inChord = false;
        this.inTuplet = false;
        this.tupletRatio = 1;
    }
}

// --- MAIN ORCHESTRATOR ---

/**
 * Scans the AlphaTex source code and automatically injects bar lines ('|')
 * when the measure duration limit is reached.
 */
export function injectBars(source: string): string {
    if (!source) return "";

    const lines = source.split('\n');
    const state = new RhythmState();
    let result: string[] = [];

    // Tracks if we are currently parsing the metadata header
    let isHeader = true;

    lines.forEach(line => {
        const trimmed = line.trim();

        // 1. Detect Header/Body transition
        if (trimmed === '.') {
            isHeader = false;
            state.resetAccumulator();
            result.push(trimmed);
            return;
        }

        // 2. Parse Header (Looking for initial Time Signature)
        if (isHeader) {
            const ts = parseTimeSignature(trimmed);
            if (ts) state.setTimeSignature(ts.num, ts.den);

            result.push(trimmed);
            return;
        }

        // --- PARSE BODY ---

        // Check for inline Time Signature changes
        const ts = parseTimeSignature(trimmed);
        if (ts) {
            state.setTimeSignature(ts.num, ts.den);
            result.push(trimmed);
            return;
        }

        let newLine = "";

        // Normalize structural tokens for easier splitting
        const tokens = trimmed
            .replace(/{/g, " { ")
            .replace(/}/g, " } ")
            .replace(/\(/g, " ( ")
            .replace(/\)/g, " ) ")
            .split(/\s+/);

        // Process tokens sequentially
        for (let i = 0; i < tokens.length; i++) {
            const token = tokens[i];
            if (!token) continue;

            // A. Tuplet Start
            if (token === '\\tuplet') {
                const nextToken = tokens[i + 1];
                if (nextToken && !isNaN(parseInt(nextToken))) {
                    state.startTuplet(parseInt(nextToken));
                }
                newLine += token + " ";
                continue;
            }

            // B. Duration Changes
            const newDuration = parseDuration(token);
            if (newDuration) {
                state.setDuration(newDuration);
                newLine += token + " ";
                continue;
            }

            // C. Structure Control
            if (token === '{') { /* Handled by startTuplet context */ }
            else if (token === '}') { state.endTuplet(); }
            else if (token === '(') { state.inChord = true; }
            else if (token === ')') { state.inChord = false; }
            else if (token === '|') { state.resetAccumulator(); }

            // D. Note Processing
            else if (token.match(REGEX_NOTE)) {
                // Ensure we don't count the tuplet definition number as a note
                const prevToken = tokens[i - 1];
                if (prevToken !== '\\tuplet') {
                    state.addNote(token);
                }
            }

            newLine += token + " ";

            // E. Bar Injection
            if (state.shouldInsertBar()) {
                newLine += "| ";
            }
        }

        result.push(newLine.replace(/\s+/g, ' ').trim());
    });

    return result.join('\n');
}

// --- ALPHATEX DETECTION & RENDER PREPARATION ---

const ALPHATEX_DIRECTIVE_REGEX = /^\\(title|subtitle|artist|album|words|music|copyright|tempo|ts|ks|clef|tuning|instrument|capo|chord|tuplet|staff|track|lyrics)\b/im;
const ALPHATEX_DURATION_REGEX = /(?:^|\s):(?:-4|-2|1|2|4|8|16|32|64|128|256)\b/m;
const ALPHATEX_FRET_STRING_REGEX = /(?:^|\s|\()\d{1,2}\.[1-8](?:\s|\)|$|\||\{)/m;
const ALPHATEX_PITCH_NOTE_REGEX = /(?:^|\s)[A-Ga-g][#b]?\d(?:\.(?:1|2|4|8|16|32|64))?(?:\s|$|\|)/m;

/**
 * Determines whether the editor text contains AlphaTex notation.
 */
export function looksLikeAlphaTex(source: string): boolean {
    if (!source || typeof source !== "string") return false;
    const trimmed = source.trim();
    if (!trimmed) return false;

    if (
        trimmed.startsWith("score ") ||
        trimmed.startsWith("part ") ||
        trimmed.startsWith("tempo ")
    ) {
        return true;
    }

    if (ALPHATEX_DIRECTIVE_REGEX.test(trimmed)) {
        return true;
    }

    if (ALPHATEX_DURATION_REGEX.test(trimmed)) {
        return true;
    }

    if (ALPHATEX_FRET_STRING_REGEX.test(trimmed)) {
        return true;
    }

    if (ALPHATEX_PITCH_NOTE_REGEX.test(trimmed)) {
        return true;
    }

    // Header separator line '.' followed by notes, rests, or bars
    const lines = trimmed.split("\n").map((l) => l.trim());
    if (lines.includes(".")) {
        return true;
    }

    return false;
}

/**
 * Normalizes and prepares AlphaTex text for AlphaTab rendering.
 * Injects automatic bar lines when enabled and ensures header-only templates
 * have at least one valid rest measure so the staff renders immediately.
 */
export function prepareAlphaTexForRender(
    source: string,
    autoBarEnabled: boolean = true,
): string {
    if (!source || typeof source !== "string") return "";
    const trimmed = source.trim();
    if (!trimmed || !looksLikeAlphaTex(trimmed)) return "";

    const processed = autoBarEnabled ? injectBars(trimmed) : trimmed;
    const lines = processed.split("\n").map((l) => l.trim());
    const dotIndex = lines.indexOf(".");

    if (dotIndex !== -1) {
        const bodyContent = lines
            .slice(dotIndex + 1)
            .join(" ")
            .trim();
        if (!bodyContent) {
            return `${processed}\n:1 r |`;
        }
    }

    return processed;
}

// --- SCORE METADATA & AUDIO SYNCHRONIZATION ---

export interface ScoreMetadata {
    bpm: number;
    timeSignature: { num: number; den: number };
    barCount: number;
    beatsPerBar: number;
    secondsPerBeat: number;
    secondsPerBar: number;
    estimatedDurationSeconds: number;
}

export interface PlaybackSyncState {
    currentBarIndex: number;
    currentBeat: number;
    totalBars: number;
    bpm: number;
    barProgress: number; // 0..1 within current bar
    scoreProgress: number; // 0..1 across entire score
}

/**
 * Extracts musical timing metadata (BPM, time signature, measure count, duration)
 * from an AlphaTex source string.
 */
export function extractScoreMetadata(
    source: string,
    fallbackBpm: number = 120,
    fallbackTimeSignature: string = "4/4",
): ScoreMetadata {
    let bpm = fallbackBpm > 0 ? fallbackBpm : 120;
    let tsNum = 4;
    let tsDen = 4;

    if (fallbackTimeSignature && fallbackTimeSignature.includes("/")) {
        const [n, d] = fallbackTimeSignature.split("/").map((v) => parseInt(v.trim(), 10));
        if (!isNaN(n) && n > 0) tsNum = n;
        if (!isNaN(d) && d > 0) tsDen = d;
    }

    if (source && typeof source === "string") {
        const tempoMatch = source.match(/(?:\\tempo|^tempo)\s+(\d+)/im);
        if (tempoMatch) {
            const parsedTempo = parseInt(tempoMatch[1], 10);
            if (!isNaN(parsedTempo) && parsedTempo > 0) {
                bpm = parsedTempo;
            }
        }

        const tsMatch = source.match(/\\ts\s+(\d+)\s+(\d+)/i);
        if (tsMatch) {
            const parsedNum = parseInt(tsMatch[1], 10);
            const parsedDen = parseInt(tsMatch[2], 10);
            if (!isNaN(parsedNum) && parsedNum > 0) tsNum = parsedNum;
            if (!isNaN(parsedDen) && parsedDen > 0) tsDen = parsedDen;
        }
    }

    const processed = source ? injectBars(source) : "";
    const lines = processed.split("\n").map((l) => l.trim());
    const dotIdx = lines.indexOf(".");
    const bodyText = (dotIdx !== -1 ? lines.slice(dotIdx + 1) : lines.filter((l) => !l.startsWith("\\")))
        .join(" ")
        .trim();

    let barCount = 1;
    if (bodyText) {
        const segments = bodyText
            .split("|")
            .map((s) => s.trim())
            .filter((s) => s.length > 0);
        barCount = Math.max(1, segments.length);
    }

    const beatsPerBar = tsNum * (4 / tsDen);
    const secondsPerBeat = 60 / Math.max(1, bpm);
    const secondsPerBar = beatsPerBar * secondsPerBeat;
    const estimatedDurationSeconds = barCount * secondsPerBar;

    return {
        bpm,
        timeSignature: { num: tsNum, den: tsDen },
        barCount,
        beatsPerBar,
        secondsPerBeat,
        secondsPerBar,
        estimatedDurationSeconds,
    };
}

/**
 * Calculates the synchronized measure, beat, and progress for a given audio playback position.
 */
export function calculatePlaybackSyncState(
    positionSeconds: number,
    durationSeconds: number,
    totalBars: number,
    bpm: number = 120,
    timeSignature: { num: number; den: number } = { num: 4, den: 4 },
): PlaybackSyncState {
    const safeBars = Math.max(1, totalBars);
    const safeBpm = Math.max(1, bpm);
    const safePos = Math.max(0, positionSeconds);

    const beatsPerBar = timeSignature.num * (4 / timeSignature.den);
    const secondsPerBeat = 60 / safeBpm;
    const secondsPerBar = Math.max(0.01, beatsPerBar * secondsPerBeat);
    const scoreDuration = safeBars * secondsPerBar;
    const effectiveDuration = durationSeconds > 0 ? durationSeconds : scoreDuration;

    const currentBarFloat = safePos / secondsPerBar;
    const currentBarIndex = Math.min(safeBars - 1, Math.floor(currentBarFloat));
    const barProgress = Math.min(1, Math.max(0, currentBarFloat - Math.floor(currentBarFloat)));
    const beatInBar = Math.min(
        timeSignature.num,
        Math.floor(barProgress * timeSignature.num) + 1,
    );
    const scoreProgress =
        effectiveDuration > 0 ? Math.min(1, Math.max(0, safePos / effectiveDuration)) : 0;

    return {
        currentBarIndex,
        currentBeat: Math.max(1, beatInBar),
        totalBars: safeBars,
        bpm: safeBpm,
        barProgress,
        scoreProgress,
    };
}

/**
 * Computes the timestamp in seconds corresponding to the start of a given measure index.
 */
export function getTimestampForBar(
    barIndex: number,
    bpm: number = 120,
    timeSignature: { num: number; den: number } = { num: 4, den: 4 },
): number {
    const safeBar = Math.max(0, barIndex);
    const safeBpm = Math.max(1, bpm);
    const beatsPerBar = timeSignature.num * (4 / timeSignature.den);
    const secondsPerBar = beatsPerBar * (60 / safeBpm);
    return safeBar * secondsPerBar;
}

// Standard 6-string guitar tuning MIDI numbers: String 1 (E4=64) down to String 6 (E2=40)
const STANDARD_GUITAR_STRING_MIDI: Record<number, number> = {
    1: 64, // E4
    2: 59, // B3
    3: 55, // G3
    4: 50, // D3
    5: 45, // A2
    6: 40, // E2
};

export interface ScheduledAlphaTexEvent {
    startTimeSeconds: number;
    durationSeconds: number;
    midiNotes: number[];
    barIndex: number;
}

/**
 * Parses an AlphaTex string into a timeline of playable note/chord events
 * for real-time Web Audio synthesis when playing the score directly.
 */
export function parsePlayableNotesFromAlphaTex(
    source: string,
    fallbackBpm: number = 120,
): { events: ScheduledAlphaTexEvent[]; durationSeconds: number; metadata: ScoreMetadata } {
    const metadata = extractScoreMetadata(source, fallbackBpm);
    if (!source || !looksLikeAlphaTex(source)) {
        return { events: [], durationSeconds: 0, metadata };
    }

    const processed = injectBars(source);
    const lines = processed.split("\n").map((l) => l.trim());
    const dotIdx = lines.indexOf(".");
    const bodyLines = dotIdx !== -1 ? lines.slice(dotIdx + 1) : lines.filter((l) => !l.startsWith("\\"));
    const bodyText = bodyLines.join(" ").trim();

    if (!bodyText) {
        return {
            events: [],
            durationSeconds: metadata.estimatedDurationSeconds,
            metadata,
        };
    }

    const tokens = bodyText
        .replace(/\(/g, " ( ")
        .replace(/\)/g, " ) ")
        .replace(/\|/g, " | ")
        .split(/\s+/)
        .filter(Boolean);

    const events: ScheduledAlphaTexEvent[] = [];
    const wholeNoteSeconds = (60 / metadata.bpm) * 4;
    let currentNoteFraction = 0.25; // Default :4 quarter note
    let currentTime = 0;
    let currentBar = 0;
    let inChord = false;
    let chordMidiNotes: number[] = [];

    function tokenToMidi(tok: string): number | null {
        const coordMatch = tok.match(/^(\d{1,2})\.([1-6])$/);
        if (coordMatch) {
            const fret = parseInt(coordMatch[1], 10);
            const str = parseInt(coordMatch[2], 10);
            const baseMidi = STANDARD_GUITAR_STRING_MIDI[str] ?? 40;
            return baseMidi + fret;
        }
        const fretOnlyMatch = tok.match(/^(\d{1,2})$/);
        if (fretOnlyMatch) {
            const fret = parseInt(fretOnlyMatch[1], 10);
            return 52 + fret; // E3 base
        }
        return null;
    }

    for (let i = 0; i < tokens.length; i++) {
        const token = tokens[i];

        const dur = parseDuration(token);
        if (dur !== null) {
            currentNoteFraction = dur;
            continue;
        }

        if (token === "|") {
            currentBar++;
            continue;
        }

        if (token === "(") {
            inChord = true;
            chordMidiNotes = [];
            continue;
        }

        if (token === ")") {
            inChord = false;
            const durSecs = currentNoteFraction * wholeNoteSeconds;
            if (chordMidiNotes.length > 0) {
                events.push({
                    startTimeSeconds: currentTime,
                    durationSeconds: durSecs,
                    midiNotes: [...chordMidiNotes],
                    barIndex: currentBar,
                });
            }
            currentTime += durSecs;
            chordMidiNotes = [];
            continue;
        }

        if (inChord) {
            const midi = tokenToMidi(token);
            if (midi !== null) {
                chordMidiNotes.push(midi);
            }
            continue;
        }

        if (token === "r" || token.startsWith("r.")) {
            const isDotted = token.includes(".");
            const durSecs = currentNoteFraction * (isDotted ? 1.5 : 1) * wholeNoteSeconds;
            currentTime += durSecs;
            continue;
        }

        const midi = tokenToMidi(token);
        if (midi !== null) {
            const durSecs = currentNoteFraction * wholeNoteSeconds;
            events.push({
                startTimeSeconds: currentTime,
                durationSeconds: durSecs,
                midiNotes: [midi],
                barIndex: currentBar,
            });
            currentTime += durSecs;
        }
    }

    const totalDuration = Math.max(currentTime, metadata.estimatedDurationSeconds);
    return {
        events,
        durationSeconds: totalDuration,
        metadata,
    };
}

