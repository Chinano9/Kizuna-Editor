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
