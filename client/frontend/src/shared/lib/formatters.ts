/*
    Kizuna Editor - Utilidades de Formateo
*/

/**
 * Formatea segundos a formato de tiempo de estudio mm:ss o mm:ss.ms
 */
export function formatTime(seconds: number, includeMs = false): string {
    if (isNaN(seconds) || seconds < 0) return includeMs ? "00:00.0" : "00:00";
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    const mm = mins.toString().padStart(2, "0");
    const ss = secs.toString().padStart(2, "0");

    if (includeMs) {
        const ms = Math.floor((seconds % 1) * 10);
        return `${mm}:${ss}.${ms}`;
    }
    return `${mm}:${ss}`;
}

/**
 * Formatea bytes a una cadena legible (KB, MB, GB)
 */
export function formatBytes(bytes: number, decimals = 1): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}

/**
 * Formatea fechas ISO a formato legible local
 */
export function formatDate(isoString: string): string {
    if (!isoString) return "";
    try {
        const date = new Date(isoString);
        return date.toLocaleDateString(undefined, {
            year: "numeric",
            month: "short",
            day: "numeric",
        });
    } catch {
        return isoString;
    }
}
