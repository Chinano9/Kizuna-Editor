<script lang="ts">
    /*
        Legacy proxy wrapper for backward compatibility with layouts.
        Delegates completely to the decomposed modular audio-deck feature
        and forwards playback synchronization events.
    */
    import WorkbarCore from "@/features/audio-deck/components/Workbar.svelte";
    import type { Track } from "@shared/types/wails";

    export let songID: number;
    export let activeTrack: Track | null = null;
</script>

<WorkbarCore
    {songID}
    {activeTrack}
    on:playbackUpdate
    on:playStateChange
/>
