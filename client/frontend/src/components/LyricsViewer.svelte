<script lang="ts">
    export let source: string = "";

    let lyrics = "";

    // Reactive statement to parse lyrics whenever the source changes
    $: {
        if (typeof source === "string") {
            lyrics = parseLyrics(source);
        } else {
            lyrics = "Invalid track data.";
        }
    }

    /**
     * A simple parser to extract lyrics from an AlphaTex string.
     * It finds lines that start with a colon followed by a space,
     * which denotes a lyric line in this convention.
     */
    function parseLyrics(alphaTex: string): string {
        if (!alphaTex) return "No lyrics found.";

        const lines = alphaTex.split("\n");
        const lyricLines = lines
            .filter((line) => line.trim().startsWith(": "))
            .map((line) => line.trim().substring(2)); // Remove the ': ' prefix

        return lyricLines.join("\n") || "No lyrics found in this track.";
    }
</script>

<div class="lyrics-container">
    <pre>{lyrics}</pre>
</div>

<style>
    .lyrics-container {
        padding: 20px;
        font-family: "Courier New", Courier, monospace;
        font-size: 1rem;
        line-height: 1.5;
        white-space: pre-wrap; /* Allows wrapping of long lines */
        word-wrap: break-word;
        color: #333;
        background-color: #fff;
        height: 100%;
        overflow-y: auto;
    }

    pre {
        margin: 0;
        font-family: inherit; /* Inherit the font from the container */
    }
</style>
