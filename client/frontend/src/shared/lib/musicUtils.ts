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

const REGEX_TS = /\\ts\s+(\d+)\s+(\d+)/i;

// Header / metadata directives that should not be treated as body notes when no '.' separator is present
const HEADER_DIRECTIVE_LINE_REGEX =
    /^\\(title|subtitle|artist|album|words|music|copyright|tempo|ts|ks|clef|tuning|instrument|capo|chord|staff|track|lyrics)\b/i;

// Directives in the body that consume following parameter tokens
const DIRECTIVES_WITH_ONE_ARG = new Set([
    "\\tempo",
    "\\instrument",
    "\\capo",
    "\\rc",
    "\\clef",
    "\\ks",
    "\\jump",
    "\\section",
    "\\marker",
    "\\track",
    "\\staff",
    "\\title",
    "\\subtitle",
    "\\artist",
    "\\album",
    "\\words",
    "\\music",
    "\\copyright",
]);

// --- PARSING HELPERS ---

/**
 * Parses a raw duration denominator string (e.g., "4", "8.", "16") into a measure fraction.
 */
function parseDurationValue(rawDur: string): number | null {
    if (!rawDur) return null;
    const hasDot = rawDur.endsWith(".");
    const cleanToken = rawDur.replace(/\./g, "");
    const val = parseInt(cleanToken, 10);

    if (!isNaN(val) && val > 0) {
        let duration = 1 / val;
        if (hasDot) duration *= 1.5;
        return duration;
    }
    return null;
}

/**
 * Extracts duration value from a range duration token (e.g., ":4", ":8.").
 * Handles dotted notes.
 */
function parseDuration(token: string): number | null {
    if (token.startsWith(":")) {
        return parseDurationValue(token.substring(1));
    }
    return null;
}

/**
 * Extracts Time Signature from a string (e.g., "\ts 4 4").
 */
function parseTimeSignature(line: string): { num: number; den: number } | null {
    const match = line.match(REGEX_TS);
    if (match) {
        return { num: parseInt(match[1], 10), den: parseInt(match[2], 10) };
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

interface ParsedNoteToken {
    explicitDuration: number | null;
    isDottedCurrent: boolean;
    multiplier: number;
}

/**
 * Parses a single token to determine if it is a valid AlphaTex note or rest,
 * extracting any explicit duration suffix (e.g. "3.3.8") or repeat multiplier (e.g. "0.6*4").
 */
function parseNoteOrRestToken(token: string): ParsedNoteToken | null {
    if (!token) return null;

    // 1. Guitar / stringed coordinate: fret.string[.duration][*multiplier] (e.g. "0.6", "3.3.4", "x.6", "-.6", "0.6*4")
    const coordMatch = token.match(/^(?:\d{1,2}|x|-)\.[1-8](?:\.(\d+)(\.?)|(\.))?(?:\*(\d+))?$/i);
    if (coordMatch) {
        const explicitDuration = coordMatch[1] ? parseDurationValue(coordMatch[1]) : null;
        const isDottedCurrent = Boolean(coordMatch[2] || coordMatch[3]);
        const multiplier = coordMatch[4] ? Math.max(1, parseInt(coordMatch[4], 10)) : 1;
        return {
            explicitDuration,
            isDottedCurrent,
            multiplier,
        };
    }

    // 2. Rest token: r[.duration][*multiplier] or r. (e.g. "r", "r.", "r.4", "r.8*2")
    const restMatch = token.match(/^r(?:\.(\d+)(\.?)|(\.))?(?:\*(\d+))?$/i);
    if (restMatch) {
        const explicitDuration = restMatch[1] ? parseDurationValue(restMatch[1]) : null;
        const isDottedCurrent = Boolean(restMatch[2] || restMatch[3]);
        const multiplier = restMatch[4] ? Math.max(1, parseInt(restMatch[4], 10)) : 1;
        return {
            explicitDuration,
            isDottedCurrent,
            multiplier,
        };
    }

    // 3. Standard notation / piano pitch: C4, D#4.8, Bb3*2
    const pitchMatch = token.match(/^[A-Ga-g][#b]{0,2}-?\d(?:\.(\d+)(\.?)|(\.))?(?:\*(\d+))?$/);
    if (pitchMatch) {
        const explicitDuration = pitchMatch[1] ? parseDurationValue(pitchMatch[1]) : null;
        const isDottedCurrent = Boolean(pitchMatch[2] || pitchMatch[3]);
        const multiplier = pitchMatch[4] ? Math.max(1, parseInt(pitchMatch[4], 10)) : 1;
        return {
            explicitDuration,
            isDottedCurrent,
            multiplier,
        };
    }

    // 4. Bare fret / tied / dead beat: 0, 2., -, x, 0*4
    const bareMatch = token.match(/^(?:\d{1,2}(\.)?|-|x)(?:\*(\d+))?$/i);
    if (bareMatch) {
        const isDottedCurrent = Boolean(bareMatch[1]);
        const multiplier = bareMatch[2] ? Math.max(1, parseInt(bareMatch[2], 10)) : 1;
        return {
            explicitDuration: null,
            isDottedCurrent,
            multiplier,
        };
    }

    return null;
}

/**
 * Checks whether a braced beat/note effect block (e.g. "{ d }", "{ dd v }") specifies a dotted beat.
 */
function hasDottedBeatEffect(braceBlock: string): boolean {
    if (!braceBlock.startsWith("{") || !braceBlock.endsWith("}")) return false;
    const inner = braceBlock.slice(1, -1);
    return /(?:^|\s)dd?(?:\s|$)/i.test(inner);
}

/**
 * Converts TeX-style '%' line comments outside of quoted strings into AlphaTab '//' comments.
 */
function normalizeAlphaTexLineComment(line: string): string {
    let inQuote: string | null = null;
    const len = line.length;
    for (let i = 0; i < len; i++) {
        const ch = line[i];
        if (inQuote) {
            if (ch === "\\" && i + 1 < len) {
                i++;
                continue;
            }
            if (ch === inQuote) {
                inQuote = null;
            }
            continue;
        }
        if (ch === '"' || ch === "'") {
            inQuote = ch;
            continue;
        }
        if (ch === "/" && line[i + 1] === "/") {
            return line;
        }
        if (ch === "%") {
            return `${line.slice(0, i)}//${line.slice(i + 1)}`;
        }
    }
    return line;
}

/**
 * Strips '//' and '%' line comments outside of quoted strings from a single line.
 */
function stripAlphaTexLineComment(line: string): string {
    let inQuote: string | null = null;
    const len = line.length;
    for (let i = 0; i < len; i++) {
        const ch = line[i];
        if (inQuote) {
            if (ch === "\\" && i + 1 < len) {
                i++;
                continue;
            }
            if (ch === inQuote) {
                inQuote = null;
            }
            continue;
        }
        if (ch === '"' || ch === "'") {
            inQuote = ch;
            continue;
        }
        if ((ch === "/" && line[i + 1] === "/") || ch === "%") {
            return line.slice(0, i).trim();
        }
    }
    return line.trim();
}

/**
 * Joins tokenized AlphaTex tokens back into a line while keeping chord parentheses tight.
 */
function joinAlphaTexTokens(tokens: string[]): string {
    let out = "";
    for (let i = 0; i < tokens.length; i++) {
        const tok = tokens[i];
        if (!tok) continue;
        if (i === 0) {
            out = tok;
            continue;
        }
        const prev = tokens[i - 1];
        if (prev === "(" || tok.startsWith(")")) {
            out += tok;
        } else {
            out += " " + tok;
        }
    }
    return out.trim();
}

/**
 * Converts dotted duration suffixes (e.g. "7.5.4.", "(5.4 5.3).4. { d }", "(0.4 0.3).2.", "r.4.")
 * into AlphaTab 1.8's "{ d }" beat effect syntax so AlphaTab's parser does not misinterpret
 * trailing duration dots as section-separator '.' tokens.
 */
function normalizeAlphaTexDottedLine(line: string): string {
    const trimmed = line.trim();
    if (
        !trimmed ||
        trimmed === "." ||
        trimmed.startsWith("//") ||
        trimmed.startsWith("%") ||
        HEADER_DIRECTIVE_LINE_REGEX.test(trimmed)
    ) {
        return line;
    }

    const tokens = tokenizeAlphaTexLine(trimmed);
    let inChord = false;
    let mutated = false;
    const out: string[] = [];

    const ensureDotEffect = (dotProp: "d" | "dd", nextIdx: number) => {
        mutated = true;
        if (
            nextIdx < tokens.length &&
            tokens[nextIdx].startsWith("{") &&
            tokens[nextIdx].endsWith("}")
        ) {
            const inner = tokens[nextIdx].slice(1, -1).trim();
            if (!/(?:^|\s)dd?(?:\s|$)/i.test(inner)) {
                tokens[nextIdx] = inner ? `{ ${dotProp} ${inner} }` : `{ ${dotProp} }`;
            }
        } else {
            out.push(`{ ${dotProp} }`);
        }
    };

    for (let i = 0; i < tokens.length; i++) {
        const tok = tokens[i];
        if (tok.startsWith("//")) {
            out.push(tok);
            continue;
        }
        if (tok === "(") {
            inChord = true;
            out.push(tok);
            continue;
        }
        if (tok.startsWith(")")) {
            inChord = false;
            const chordDotMatch = tok.match(/^\)(?:\.(\d+))?\.(\.)?(\*\d+)?$/);
            if (chordDotMatch) {
                const cleanTok =
                    ")" +
                    (chordDotMatch[1] ? `.${chordDotMatch[1]}` : "") +
                    (chordDotMatch[3] ?? "");
                out.push(cleanTok);
                ensureDotEffect(chordDotMatch[2] ? "dd" : "d", i + 1);
                continue;
            }
            if (tok === ")" && i + 1 < tokens.length) {
                const sepDotMatch = tokens[i + 1].match(/^\.(\d+)\.(\.)?(\*\d+)?$/);
                if (sepDotMatch) {
                    i++;
                    out.push(`).${sepDotMatch[1]}${sepDotMatch[3] ?? ""}`);
                    ensureDotEffect(sepDotMatch[2] ? "dd" : "d", i + 1);
                    continue;
                }
            }
            out.push(tok);
            continue;
        }

        if (!inChord) {
            const rangeDotMatch = tok.match(/^:(\d+)\.(\.)?$/);
            if (rangeDotMatch) {
                mutated = true;
                out.push(`:${rangeDotMatch[1]}`);
                continue;
            }

            const coordDotMatch = tok.match(
                /^((?:\d{1,2}|x|-)\.[1-8](?:\.\d+)?)\.(\.)?(\*\d+)?$/i,
            );
            if (coordDotMatch) {
                out.push(`${coordDotMatch[1]}${coordDotMatch[3] ?? ""}`);
                ensureDotEffect(coordDotMatch[2] ? "dd" : "d", i + 1);
                continue;
            }

            const restDotMatch = tok.match(/^(r(?:\.\d+)?)\.(\.)?(\*\d+)?$/i);
            if (restDotMatch) {
                out.push(`${restDotMatch[1]}${restDotMatch[3] ?? ""}`);
                ensureDotEffect(restDotMatch[2] ? "dd" : "d", i + 1);
                continue;
            }

            const pitchDotMatch = tok.match(
                /^([A-Ga-g][#b]{0,2}-?\d(?:\.\d+)?)\.(\.)?(\*\d+)?$/,
            );
            if (pitchDotMatch) {
                out.push(`${pitchDotMatch[1]}${pitchDotMatch[3] ?? ""}`);
                ensureDotEffect(pitchDotMatch[2] ? "dd" : "d", i + 1);
                continue;
            }
        }

        out.push(tok);
    }

    return mutated ? joinAlphaTexTokens(out) : line;
}

/**
 * Normalizes raw AlphaTex source by:
 * 1. Converting '%' comments outside quotes to '//' comments so AlphaTab's parser accepts them.
 * 2. Splitting glued leading '.' separators (e.g. '.\title' or '.:4') onto their own line.
 * 3. Converting dotted duration suffixes ('.4.', '7.5.4.') into AlphaTab '{ d }' beat properties.
 * 4. Removing redundant empty default headers (e.g. '\title "New Song"\n.') when followed by another header.
 */
export function normalizeAlphaTexSource(source: string): string {
    if (!source || typeof source !== "string") return "";

    const rawLines = source.split("\n");
    const expandedLines: string[] = [];

    for (const rawLine of rawLines) {
        const normalizedLine = normalizeAlphaTexLineComment(rawLine);
        const trimmed = normalizedLine.trim();
        // If a line starts with '.' immediately glued to a directive, duration, chord, or note (e.g. '.\title', '.:4')
        if (/^\.(?=\\|:|\(|r\b|[A-Ga-g][#b]?\d|\d{1,2}\.[1-8])/.test(trimmed)) {
            expandedLines.push(".");
            expandedLines.push(normalizeAlphaTexDottedLine(trimmed.slice(1).trim()));
        } else {
            expandedLines.push(normalizeAlphaTexDottedLine(normalizedLine));
        }
    }

    // Check if there are standalone '.' lines where the section before the first '.'
    // is a redundant header followed by another pasted header.
    const dotIndices: number[] = [];
    for (let i = 0; i < expandedLines.length; i++) {
        if (expandedLines[i].trim() === ".") {
            dotIndices.push(i);
        }
    }

    if (dotIndices.length >= 2) {
        const firstDot = dotIndices[0];
        const secondDot = dotIndices[1];
        const beforeFirstDot = expandedLines
            .slice(0, firstDot)
            .map((l) => stripAlphaTexLineComment(l))
            .filter((l) => l.length > 0);
        const betweenDots = expandedLines
            .slice(firstDot + 1, secondDot)
            .map((l) => stripAlphaTexLineComment(l))
            .filter((l) => l.length > 0);

        const allBeforeAreHeaders = beforeFirstDot.every((l) =>
            HEADER_DIRECTIVE_LINE_REGEX.test(l),
        );
        const allBetweenAreHeaders =
            betweenDots.length > 0 &&
            betweenDots.every((l) => HEADER_DIRECTIVE_LINE_REGEX.test(l));

        if (allBeforeAreHeaders && allBetweenAreHeaders) {
            const hasSecondTitle = betweenDots.some((l) => /^\\title\b/i.test(l));
            const isDefaultFirstHeader =
                beforeFirstDot.length === 1 &&
                /^\\title\s+["']New Song["']$/i.test(beforeFirstDot[0]);
            if (isDefaultFirstHeader || hasSecondTitle) {
                return expandedLines.slice(firstDot + 1).join("\n");
            }
            return [
                ...expandedLines.slice(0, firstDot),
                ...expandedLines.slice(firstDot + 1),
            ].join("\n");
        }
    } else if (dotIndices.length === 1) {
        const firstDot = dotIndices[0];
        const beforeFirstDot = expandedLines
            .slice(0, firstDot)
            .map((l) => stripAlphaTexLineComment(l))
            .filter((l) => l.length > 0);
        const afterFirstDot = expandedLines
            .slice(firstDot + 1)
            .map((l) => stripAlphaTexLineComment(l))
            .filter((l) => l.length > 0);

        const isDefaultFirstHeader =
            beforeFirstDot.length === 1 &&
            /^\\title\s+["']New Song["']$/i.test(beforeFirstDot[0]);
        if (isDefaultFirstHeader && afterFirstDot.length > 0 && /^\\title\b/i.test(afterFirstDot[0])) {
            return expandedLines.slice(firstDot + 1).join("\n");
        }
    }

    return expandedLines.join("\n");
}

/**
 * Tokenizes a single AlphaTex line while keeping quoted strings ("...") and
 * braced effect/property blocks ({...}) intact so their contents are never
 * split or misinterpreted as notes.
 */
function tokenizeAlphaTexLine(line: string): string[] {
    const tokens: string[] = [];
    let current = "";
    let i = 0;
    const len = line.length;

    const flush = () => {
        if (current.length > 0) {
            tokens.push(current);
            current = "";
        }
    };

    while (i < len) {
        const ch = line[i];

        // Line comment '//' or '%' outside quotes
        if ((ch === "/" && line[i + 1] === "/") || ch === "%") {
            flush();
            const commentBody = ch === "%" ? `//${line.slice(i + 1)}` : line.slice(i);
            tokens.push(commentBody.trim());
            break;
        }

        // If '.' at the start of a token is immediately followed by ':' or '\', flush '.' as its own separator token
        if (current === "." && (ch === ":" || ch === "\\")) {
            flush();
        }

        // Whitespace
        if (/\s/.test(ch)) {
            flush();
            i++;
            continue;
        }

        // Quoted string ("..." or '...')
        if (ch === '"' || ch === "'") {
            flush();
            const quote = ch;
            let str = ch;
            i++;
            while (i < len) {
                const c = line[i];
                str += c;
                i++;
                if (c === "\\" && i < len) {
                    str += line[i];
                    i++;
                } else if (c === quote) {
                    break;
                }
            }
            tokens.push(str);
            continue;
        }

        // Braced block {...} (beat/note effects like {v}, {b (0 4)}, {score})
        if (ch === "{") {
            flush();
            let depth = 1;
            let block = "{";
            i++;
            while (i < len && depth > 0) {
                const c = line[i];
                if (c === "{") depth++;
                else if (c === "}") depth--;
                block += c;
                i++;
            }
            tokens.push(block);
            continue;
        }

        // Chord open '('
        if (ch === "(") {
            flush();
            tokens.push("(");
            i++;
            continue;
        }

        // Chord close ')' with optional attached duration/multiplier suffix like ').8' or ')*4'
        if (ch === ")") {
            flush();
            let closeTok = ")";
            i++;
            while (i < len && /[0-9.*]/.test(line[i])) {
                closeTok += line[i];
                i++;
            }
            tokens.push(closeTok);
            continue;
        }

        // Bar line '|'
        if (ch === "|") {
            flush();
            tokens.push("|");
            i++;
            continue;
        }

        current += ch;
        i++;
    }

    flush();
    return tokens;
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

    constructor() {}

    public setTimeSignature(num: number, den: number) {
        if (num > 0 && den > 0) {
            this.limit = num * (1 / den);
        }
        this.resetAccumulator();
    }

    public setDuration(val: number) {
        if (val > 0) {
            this.currentDuration = val;
        }
    }

    public startTuplet(n: number) {
        this.tupletRatio = getTupletRatio(n);
        this.inTuplet = true;
    }

    public endTuplet() {
        this.inTuplet = false;
        this.tupletRatio = 1;
    }

    public addBeat(isDottedCurrent: boolean = false, multiplier: number = 1) {
        if (this.inChord) return;

        let duration = this.currentDuration;
        if (isDottedCurrent) {
            duration *= 1.5;
        }
        if (this.inTuplet) {
            duration *= this.tupletRatio;
        }

        this.accumulator += duration * Math.max(1, multiplier);
    }

    public addNote(token: string) {
        if (this.inChord) return;
        const parsed = parseNoteOrRestToken(token);
        if (!parsed) return;

        if (parsed.explicitDuration !== null) {
            this.setDuration(parsed.explicitDuration);
        }
        this.addBeat(parsed.isDottedCurrent, parsed.multiplier);
    }

    public shouldInsertBar(): boolean {
        // Never insert a bar line in the middle of a chord or tuplet group
        if (this.inChord || this.inTuplet) return false;

        // Use epsilon for floating point comparison stability
        if (this.accumulator >= this.limit - 0.001) {
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

/**
 * Checks whether the Next non-empty line in the array begins with an explicit bar line '|'.
 */
function nextLineStartsWithBar(lines: string[], currentLineIndex: number): boolean {
    for (let idx = currentLineIndex + 1; idx < lines.length; idx++) {
        const t = lines[idx].trim();
        if (!t) continue;
        return t.startsWith("|");
    }
    return false;
}

// --- MAIN ORCHESTRATOR ---

// Simple memoization caches to avoid re-parsing the same AlphaTex string across multiple components
let lastInjectSource = "";
let lastInjectResult = "";

/**
 * Scans the AlphaTex source code and automatically injects bar lines ('|')
 * when the measure duration limit is reached.
 */
export function injectBars(source: string): string {
    if (!source || typeof source !== "string") return "";
    if (source === lastInjectSource) return lastInjectResult;

    const normalizedSource = normalizeAlphaTexSource(source);
    const lines = normalizedSource.split("\n");
    const tokenizedLines = lines.map((line) => tokenizeAlphaTexLine(line.trim()));

    // Detect if the score contains an explicit '.' header/body separator token anywhere
    const hasExplicitDotSeparator = tokenizedLines.some((tokens) => tokens.includes("."));

    const state = new RhythmState();
    const result: string[] = [];

    // Tracks if we are currently parsing the metadata header
    let isHeader = hasExplicitDotSeparator;

    for (let lineIdx = 0; lineIdx < lines.length; lineIdx++) {
        const trimmed = lines[lineIdx].trim();
        const tokens = tokenizedLines[lineIdx];

        if (!trimmed) {
            result.push("");
            continue;
        }

        // 1. Standalone '.' line transitions from Header to Body
        if (trimmed === ".") {
            isHeader = false;
            state.resetAccumulator();
            result.push(".");
            continue;
        }

        // 2. Pure comment line in header or body passes through untouched
        if (trimmed.startsWith("//") || trimmed.startsWith("%")) {
            result.push(normalizeAlphaTexLineComment(trimmed));
            continue;
        }

        // 3. If there is no '.' separator in the document, keep leading metadata lines intact
        if (!hasExplicitDotSeparator && HEADER_DIRECTIVE_LINE_REGEX.test(trimmed)) {
            const ts = parseTimeSignature(trimmed);
            if (ts) state.setTimeSignature(ts.num, ts.den);
            result.push(trimmed);
            continue;
        }

        // 4. Pure header line (before standalone '.' line and with no inline '.' token)
        if (isHeader && !tokens.includes(".")) {
            const ts = parseTimeSignature(trimmed);
            if (ts) state.setTimeSignature(ts.num, ts.den);
            result.push(trimmed);
            continue;
        }

        // 5. Standalone inline '\ts N D' line
        if (/^\\ts\s+\d+\s+\d+$/i.test(trimmed)) {
            const ts = parseTimeSignature(trimmed);
            if (ts) state.setTimeSignature(ts.num, ts.den);
            result.push(trimmed);
            continue;
        }

        // --- PARSE TOKENS (Supports both inline '.' transition and body notation) ---
        const outTokens: string[] = [];

        for (let i = 0; i < tokens.length; i++) {
            const token = tokens[i];
            if (!token) continue;

            // Handle inline header-to-body transition '.'
            if (isHeader) {
                outTokens.push(token);
                if (token.toLowerCase() === "\\ts") {
                    const num = parseInt(tokens[i + 1] ?? "", 10);
                    const den = parseInt(tokens[i + 2] ?? "", 10);
                    if (!isNaN(num) && !isNaN(den)) {
                        state.setTimeSignature(num, den);
                    }
                } else if (token === ".") {
                    isHeader = false;
                    state.resetAccumulator();
                }
                continue;
            }

            // Comments pass through
            if (token.startsWith("//")) {
                outTokens.push(token);
                continue;
            }

            // A. Directives in body (\tuplet, \ts, \tempo, \instrument, \rc, \staff, \track, etc.)
            if (token.startsWith("\\")) {
                const lowerDir = token.toLowerCase();
                outTokens.push(token);

                if (lowerDir === "\\tuplet") {
                    const nextToken = tokens[i + 1];
                    const tupletN = nextToken ? parseInt(nextToken, 10) : NaN;
                    if (!isNaN(tupletN)) {
                        state.startTuplet(tupletN);
                        outTokens.push(nextToken);
                        i++;
                        // Optional '\tuplet 3 { ... }' opening brace
                        if (tokens[i + 1] === "{") {
                            outTokens.push("{");
                            i++;
                        }
                    }
                    continue;
                }

                if (lowerDir === "\\ts") {
                    const num = parseInt(tokens[i + 1] ?? "", 10);
                    const den = parseInt(tokens[i + 2] ?? "", 10);
                    if (!isNaN(num) && !isNaN(den)) {
                        state.setTimeSignature(num, den);
                        outTokens.push(tokens[i + 1], tokens[i + 2]);
                        i += 2;
                    } else if (
                        tokens[i + 1] &&
                        /^(common|half)$/i.test(tokens[i + 1])
                    ) {
                        state.setTimeSignature(4, 4);
                        outTokens.push(tokens[i + 1]);
                        i++;
                    }
                    continue;
                }

                if (lowerDir === "\\tuning") {
                    while (
                        i + 1 < tokens.length &&
                        /^(?:[A-Ga-g][#b]?\d|piano|none)$/i.test(tokens[i + 1])
                    ) {
                        i++;
                        outTokens.push(tokens[i]);
                    }
                    continue;
                }

                if (DIRECTIVES_WITH_ONE_ARG.has(lowerDir)) {
                    // Consume non-note argument token if present (e.g. \tempo 120, \instrument 25, \rc 2)
                    if (
                        i + 1 < tokens.length &&
                        tokens[i + 1] !== "|" &&
                        !tokens[i + 1].startsWith(":") &&
                        !tokens[i + 1].startsWith("\\")
                    ) {
                        i++;
                        outTokens.push(tokens[i]);
                    }
                }

                // Consume any trailing quoted strings or {...} property blocks attached to the directive
                while (
                    i + 1 < tokens.length &&
                    (tokens[i + 1].startsWith('"') ||
                        tokens[i + 1].startsWith("'") ||
                        tokens[i + 1].startsWith("{"))
                ) {
                    i++;
                    outTokens.push(tokens[i]);
                }
                continue;
            }

            // B. Duration Changes (:4, :8., etc.)
            const newDuration = parseDuration(token);
            if (newDuration !== null) {
                state.setDuration(newDuration);
                outTokens.push(token);
                continue;
            }

            // C. Structure Control
            if (token === "}") {
                state.endTuplet();
                outTokens.push(token);
                continue;
            }

            if (token.startsWith("{")) {
                outTokens.push(token);
                continue;
            }

            if (token === "(") {
                state.inChord = true;
                outTokens.push(token);
                continue;
            }

            if (token.startsWith(")")) {
                state.inChord = false;
                outTokens.push(token);

                // Parse optional chord duration/multiplier suffix like ').8' or ')*4'
                const suffix = token.slice(1);
                let multiplier = 1;
                let isDotted = false;
                if (suffix) {
                    const chordSuffixMatch = suffix.match(/^(?:\.(\d+)(\.?)|(\.))?(?:\*(\d+))?$/);
                    if (chordSuffixMatch) {
                        if (chordSuffixMatch[1]) {
                            const explicitDur = parseDurationValue(chordSuffixMatch[1]);
                            if (explicitDur !== null) state.setDuration(explicitDur);
                        }
                        if (chordSuffixMatch[2] || chordSuffixMatch[3]) {
                            isDotted = true;
                        }
                        if (chordSuffixMatch[4]) {
                            multiplier = Math.max(1, parseInt(chordSuffixMatch[4], 10));
                        }
                    }
                } else if (
                    i + 1 < tokens.length &&
                    /^\.\d+\.?$/.test(tokens[i + 1])
                ) {
                    // Space-separated chord duration suffix: '( 0.1 1.2 ) .8'
                    i++;
                    outTokens.push(tokens[i]);
                    const rawDur = tokens[i].slice(1);
                    if (rawDur.endsWith(".")) isDotted = true;
                    const explicitDur = parseDurationValue(rawDur.replace(/\./g, ""));
                    if (explicitDur !== null) state.setDuration(explicitDur);
                }

                // Keep any trailing beat effects {...} attached before injecting a bar line,
                // and check if any effect block specifies a dotted beat ('{ d }' or '{ dd }')
                while (i + 1 < tokens.length && tokens[i + 1].startsWith("{")) {
                    i++;
                    outTokens.push(tokens[i]);
                    if (hasDottedBeatEffect(tokens[i])) {
                        isDotted = true;
                    }
                }

                state.addBeat(isDotted, multiplier);

                if (state.shouldInsertBar()) {
                    const nextTok = tokens[i + 1];
                    const hasExistingBarNext =
                        nextTok === "|" ||
                        (nextTok === undefined && nextLineStartsWithBar(lines, lineIdx));
                    if (!hasExistingBarNext) {
                        outTokens.push("|");
                    }
                }
                continue;
            }

            if (token === "|") {
                state.resetAccumulator();
                outTokens.push("|");
                continue;
            }

            // D. Note or Rest Processing
            const noteInfo = parseNoteOrRestToken(token);
            outTokens.push(token);

            if (noteInfo !== null) {
                let isDottedNote = noteInfo.isDottedCurrent;

                // Consume any {...} note/beat effect blocks BEFORE checking for bar injection
                // so '|' is never placed between a note and its '{...}' effect!
                while (i + 1 < tokens.length && tokens[i + 1].startsWith("{")) {
                    i++;
                    outTokens.push(tokens[i]);
                    if (hasDottedBeatEffect(tokens[i])) {
                        isDottedNote = true;
                    }
                }

                if (!state.inChord) {
                    if (noteInfo.explicitDuration !== null) {
                        state.setDuration(noteInfo.explicitDuration);
                    }
                    state.addBeat(isDottedNote, noteInfo.multiplier);
                }

                if (state.shouldInsertBar()) {
                    const nextTok = tokens[i + 1];
                    const hasExistingBarNext =
                        nextTok === "|" ||
                        (nextTok === undefined && nextLineStartsWithBar(lines, lineIdx));
                    if (!hasExistingBarNext) {
                        outTokens.push("|");
                    }
                }
            }
        }

        result.push(joinAlphaTexTokens(outTokens));
    }

    lastInjectSource = source;
    lastInjectResult = result.join("\n");
    return lastInjectResult;
}

// --- ALPHATEX DETECTION & RENDER PREPARATION ---

const ALPHATEX_DIRECTIVE_REGEX =
    /(?:^|\s)\\(title|subtitle|artist|album|words|music|copyright|tempo|ts|ks|clef|tuning|instrument|capo|chord|tuplet|staff|track|lyrics|ro|rc|ae|ac|jump|section|marker)\b/im;
const ALPHATEX_DURATION_REGEX = /(?:^|\s):(?:-4|-2|1|2|4|8|16|32|64|128|256)\b/m;
const ALPHATEX_FRET_STRING_REGEX =
    /(?:^|\s|\()(?:\d{1,2}|x|-)\.[1-8](?:\.(?:1|2|4|8|16|32|64)\.?)?(?:\*\d+)?(?:\s|\)|$|\||\{)/im;
const ALPHATEX_PITCH_NOTE_REGEX =
    /(?:^|\s)[A-Ga-g][#b]?\d(?:\.(?:1|2|4|8|16|32|64)\.?)?(?:\*\d+)?(?:\s|$|\||\{)/m;

/**
 * Splits an AlphaTex source string into its metadata header and musical body,
 * supporting standalone '.' lines, inline '.' separators, and headerless bodies.
 * Strips line comments ('//' and '%') from bodyText so comments never corrupt
 * bar counting or note synthesis.
 */
function splitAlphaTexHeaderAndBody(source: string): {
    hasDotSeparator: boolean;
    bodyText: string;
} {
    if (!source || typeof source !== "string") {
        return { hasDotSeparator: false, bodyText: "" };
    }

    const normalized = normalizeAlphaTexSource(source);
    const lines = normalized.split("\n").map((l) => stripAlphaTexLineComment(l));
    const dotLineIdx = lines.indexOf(".");
    if (dotLineIdx !== -1) {
        const bodyLines = lines
            .slice(dotLineIdx + 1)
            .filter((l) => l.length > 0 && l !== ".");
        return {
            hasDotSeparator: true,
            bodyText: bodyLines.join("\n").trim(),
        };
    }

    // Check for inline '.' token outside quotes/braces
    const bodyParts: string[] = [];
    let foundInlineDot = false;
    for (const line of lines) {
        if (!line) continue;
        if (foundInlineDot) {
            bodyParts.push(line);
            continue;
        }
        const tokens = tokenizeAlphaTexLine(line);
        const inlineDotIdx = tokens.indexOf(".");
        if (inlineDotIdx !== -1) {
            foundInlineDot = true;
            const afterDot = tokens.slice(inlineDotIdx + 1).join(" ").trim();
            if (afterDot) bodyParts.push(afterDot);
        }
    }

    if (foundInlineDot) {
        return {
            hasDotSeparator: true,
            bodyText: bodyParts.join("\n").trim(),
        };
    }

    // No '.' separator: body consists of non-metadata lines
    const nonHeaderLines = lines.filter(
        (l) => l.length > 0 && !HEADER_DIRECTIVE_LINE_REGEX.test(l),
    );
    return {
        hasDotSeparator: false,
        bodyText: nonHeaderLines.join("\n").trim(),
    };
}

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

    // Bare bar lines or rests or chords
    if (/(?:^|\s)\|(?:\s|$)/.test(trimmed)) {
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
 * Converts '%' comments to '//', cleans up redundant default headers, and
 * injects automatic bar lines when enabled. Returns "" when no musical body
 * notes/bars exist yet so a blank/header-only editor does not render a fake default score.
 */
export function prepareAlphaTexForRender(
    source: string,
    autoBarEnabled: boolean = true,
): string {
    if (!source || typeof source !== "string") return "";
    const trimmed = source.trim();
    if (!trimmed || !looksLikeAlphaTex(trimmed)) return "";

    const normalized = normalizeAlphaTexSource(trimmed);
    const processed = autoBarEnabled ? injectBars(normalized) : normalized;
    const { bodyText } = splitAlphaTexHeaderAndBody(processed);

    // Do not render a fake default score if there are no body notes, chords, or bars written yet
    if (!bodyText) {
        return "";
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

let lastMetaKey = "";
let lastMetaResult: ScoreMetadata | null = null;

/**
 * Extracts musical timing metadata (BPM, time signature, measure count, duration)
 * from an AlphaTex source string.
 */
export function extractScoreMetadata(
    source: string,
    fallbackBpm: number = 120,
    fallbackTimeSignature: string = "4/4",
): ScoreMetadata {
    const cacheKey = `${fallbackBpm}|${fallbackTimeSignature}|${source || ""}`;
    if (cacheKey === lastMetaKey && lastMetaResult) {
        return lastMetaResult;
    }

    let bpm = fallbackBpm > 0 ? fallbackBpm : 120;
    let tsNum = 4;
    let tsDen = 4;

    if (fallbackTimeSignature && fallbackTimeSignature.includes("/")) {
        const [n, d] = fallbackTimeSignature.split("/").map((v) => parseInt(v.trim(), 10));
        if (!isNaN(n) && n > 0) tsNum = n;
        if (!isNaN(d) && d > 0) tsDen = d;
    }

    const normalized = source && typeof source === "string" ? normalizeAlphaTexSource(source) : "";

    if (normalized) {
        const tempoMatches = [...normalized.matchAll(/(?:\\tempo|^tempo)\s+(\d+)/gim)];
        if (tempoMatches.length > 0) {
            const parsedTempo = parseInt(tempoMatches[tempoMatches.length - 1][1], 10);
            if (!isNaN(parsedTempo) && parsedTempo > 0) {
                bpm = parsedTempo;
            }
        }

        const tsMatches = [...normalized.matchAll(/\\ts\s+(\d+)\s+(\d+)/gi)];
        if (tsMatches.length > 0) {
            const lastTs = tsMatches[tsMatches.length - 1];
            const parsedNum = parseInt(lastTs[1], 10);
            const parsedDen = parseInt(lastTs[2], 10);
            if (!isNaN(parsedNum) && parsedNum > 0) tsNum = parsedNum;
            if (!isNaN(parsedDen) && parsedDen > 0) tsDen = parsedDen;
        }
    }

    const processed = normalized ? injectBars(normalized) : "";
    const { bodyText } = splitAlphaTexHeaderAndBody(processed);

    let barCount = 1;
    if (bodyText) {
        // Strip metadata/track directives before counting non-empty bar segments
        const cleanBody = bodyText
            .split("\n")
            .filter((l) => !HEADER_DIRECTIVE_LINE_REGEX.test(l.trim()))
            .join(" ");
        const segments = cleanBody
            .split("|")
            .map((s) => s.trim())
            .filter((s) => s.length > 0);
        barCount = Math.max(1, segments.length);
    }

    const beatsPerBar = tsNum * (4 / tsDen);
    const secondsPerBeat = 60 / Math.max(1, bpm);
    const secondsPerBar = beatsPerBar * secondsPerBeat;
    const estimatedDurationSeconds = barCount * secondsPerBar;

    lastMetaKey = cacheKey;
    lastMetaResult = {
        bpm,
        timeSignature: { num: tsNum, den: tsDen },
        barCount,
        beatsPerBar,
        secondsPerBeat,
        secondsPerBar,
        estimatedDurationSeconds,
    };
    return lastMetaResult;
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

// Standard guitar/bass tuning MIDI numbers: String 1 (E4=64) down to String 8
const STANDARD_GUITAR_STRING_MIDI: Record<number, number> = {
    1: 64, // E4
    2: 59, // B3
    3: 55, // G3
    4: 50, // D3
    5: 45, // A2
    6: 40, // E2
    7: 35, // B1
    8: 30, // F#1
};

const PITCH_CLASS_SEMITONES: Record<string, number> = {
    c: 0,
    d: 2,
    e: 4,
    f: 5,
    g: 7,
    a: 9,
    b: 11,
};

export interface ScheduledAlphaTexEvent {
    startTimeSeconds: number;
    durationSeconds: number;
    midiNotes: number[];
    barIndex: number;
}

let lastPlayableKey = "";
let lastPlayableResult: {
    events: ScheduledAlphaTexEvent[];
    durationSeconds: number;
    metadata: ScoreMetadata;
} | null = null;

/**
 * Parses an AlphaTex string into a timeline of playable note/chord events
 * for real-time Web Audio synthesis when playing the score directly or alongside a recorded take.
 */
export function parsePlayableNotesFromAlphaTex(
    source: string,
    fallbackBpm: number = 120,
): { events: ScheduledAlphaTexEvent[]; durationSeconds: number; metadata: ScoreMetadata } {
    const cacheKey = `${fallbackBpm}|${source || ""}`;
    if (cacheKey === lastPlayableKey && lastPlayableResult) {
        return lastPlayableResult;
    }

    const metadata = extractScoreMetadata(source, fallbackBpm);
    if (!source || !looksLikeAlphaTex(source)) {
        return { events: [], durationSeconds: 0, metadata };
    }

    const processed = injectBars(source);
    const { bodyText } = splitAlphaTexHeaderAndBody(processed);

    if (!bodyText) {
        return {
            events: [],
            durationSeconds: 0,
            metadata,
        };
    }

    // Tokenize line by line so line comments never swallow following lines
    const tokens: string[] = [];
    for (const line of bodyText.split("\n")) {
        const lineTokens = tokenizeAlphaTexLine(line.trim());
        for (const tok of lineTokens) {
            if (!tok.startsWith("//") && !tok.startsWith("%")) {
                tokens.push(tok);
            }
        }
    }

    const events: ScheduledAlphaTexEvent[] = [];
    const wholeNoteSeconds = (60 / metadata.bpm) * 4;
    let currentNoteFraction = 0.25; // Default :4 quarter note
    let currentTime = 0;
    let currentBar = 0;
    let inChord = false;
    let chordMidiNotes: number[] = [];

    function tokenToMidi(tok: string): number | null {
        const coordMatch = tok.match(/^(\d{1,2})\.([1-8])(?:\.\d+\.?)?(?:\*\d+)?$/);
        if (coordMatch) {
            const fret = parseInt(coordMatch[1], 10);
            const str = parseInt(coordMatch[2], 10);
            const baseMidi = STANDARD_GUITAR_STRING_MIDI[str] ?? 40;
            return baseMidi + fret;
        }
        const pitchMatch = tok.match(/^([A-Ga-g])([#b]{0,2})(-?\d)(?:\.\d+\.?)?(?:\*\d+)?$/);
        if (pitchMatch) {
            const base = PITCH_CLASS_SEMITONES[pitchMatch[1].toLowerCase()] ?? 0;
            const accidentals = pitchMatch[2];
            let accOffset = 0;
            for (const ch of accidentals) {
                if (ch === "#") accOffset += 1;
                else if (ch === "b") accOffset -= 1;
            }
            const octave = parseInt(pitchMatch[3], 10);
            return (octave + 1) * 12 + base + accOffset;
        }
        const fretOnlyMatch = tok.match(/^(\d{1,2})\.?(?:\*\d+)?$/);
        if (fretOnlyMatch) {
            const fret = parseInt(fretOnlyMatch[1], 10);
            return 52 + fret; // E3 base
        }
        return null;
    }

    for (let i = 0; i < tokens.length; i++) {
        const token = tokens[i];

        if (
            token === "." ||
            token.startsWith("{") ||
            token.startsWith('"') ||
            token.startsWith("'")
        ) {
            continue;
        }

        if (token.startsWith("\\")) {
            const lowerDir = token.toLowerCase();
            if (lowerDir === "\\ts") {
                i += 2;
            } else if (DIRECTIVES_WITH_ONE_ARG.has(lowerDir) || lowerDir === "\\tuplet") {
                if (
                    i + 1 < tokens.length &&
                    tokens[i + 1] !== "|" &&
                    !tokens[i + 1].startsWith(":") &&
                    !tokens[i + 1].startsWith("\\")
                ) {
                    i += 1;
                }
            }
            continue;
        }

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

        if (token.startsWith(")")) {
            inChord = false;
            const suffix = token.slice(1);
            let multiplier = 1;
            let isDotted = false;
            if (suffix) {
                const chordSuffixMatch = suffix.match(/^(?:\.(\d+)(\.?)|(\.))?(?:\*(\d+))?$/);
                if (chordSuffixMatch) {
                    if (chordSuffixMatch[1]) {
                        const explicitDur = parseDurationValue(chordSuffixMatch[1]);
                        if (explicitDur !== null) currentNoteFraction = explicitDur;
                    }
                    if (chordSuffixMatch[2] || chordSuffixMatch[3]) {
                        isDotted = true;
                    }
                    if (chordSuffixMatch[4]) {
                        multiplier = Math.max(1, parseInt(chordSuffixMatch[4], 10));
                    }
                }
            } else if (
                i + 1 < tokens.length &&
                /^\.\d+\.?$/.test(tokens[i + 1])
            ) {
                i++;
                const rawDur = tokens[i].slice(1);
                if (rawDur.endsWith(".")) isDotted = true;
                const explicitDur = parseDurationValue(rawDur.replace(/\./g, ""));
                if (explicitDur !== null) currentNoteFraction = explicitDur;
            }

            while (i + 1 < tokens.length && tokens[i + 1].startsWith("{")) {
                i++;
                if (hasDottedBeatEffect(tokens[i])) {
                    isDotted = true;
                }
            }

            const durSecs = currentNoteFraction * (isDotted ? 1.5 : 1) * wholeNoteSeconds;
            for (let m = 0; m < multiplier; m++) {
                if (chordMidiNotes.length > 0) {
                    events.push({
                        startTimeSeconds: currentTime,
                        durationSeconds: durSecs,
                        midiNotes: [...chordMidiNotes],
                        barIndex: currentBar,
                    });
                }
                currentTime += durSecs;
            }
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

        const noteInfo = parseNoteOrRestToken(token);
        if (noteInfo && noteInfo.explicitDuration !== null) {
            currentNoteFraction = noteInfo.explicitDuration;
        }

        if (token === "r" || token.startsWith("r.") || token.startsWith("r*")) {
            let isDotted = noteInfo ? noteInfo.isDottedCurrent : token === "r.";
            while (i + 1 < tokens.length && tokens[i + 1].startsWith("{")) {
                i++;
                if (hasDottedBeatEffect(tokens[i])) {
                    isDotted = true;
                }
            }
            const mult = noteInfo ? noteInfo.multiplier : 1;
            const durSecs = currentNoteFraction * (isDotted ? 1.5 : 1) * wholeNoteSeconds * mult;
            currentTime += durSecs;
            continue;
        }

        const midi = tokenToMidi(token);
        if (midi !== null) {
            const mult = noteInfo ? noteInfo.multiplier : 1;
            let isDotted = noteInfo ? noteInfo.isDottedCurrent : false;
            while (i + 1 < tokens.length && tokens[i + 1].startsWith("{")) {
                i++;
                if (hasDottedBeatEffect(tokens[i])) {
                    isDotted = true;
                }
            }
            const durSecs = currentNoteFraction * (isDotted ? 1.5 : 1) * wholeNoteSeconds;
            for (let m = 0; m < mult; m++) {
                events.push({
                    startTimeSeconds: currentTime,
                    durationSeconds: durSecs,
                    midiNotes: [midi],
                    barIndex: currentBar,
                });
                currentTime += durSecs;
            }
        }
    }

    const totalDuration = events.length > 0
        ? Math.max(currentTime, metadata.estimatedDurationSeconds)
        : 0;
    lastPlayableKey = cacheKey;
    lastPlayableResult = {
        events,
        durationSeconds: totalDuration,
        metadata,
    };
    return lastPlayableResult;
}

