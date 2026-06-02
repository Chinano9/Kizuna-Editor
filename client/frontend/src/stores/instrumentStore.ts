import { writable, get } from "svelte/store";
import type { main } from "../../wailsjs/go/models";
import { GetInstruments } from "../../wailsjs/go/main/App";

// Create a writable store to hold the list of instruments
const { subscribe, set } = writable<main.Instrument[]>([]);

let instrumentsLoaded = false;

// Function to load instruments from the backend
async function loadInstruments() {
  if (instrumentsLoaded) return;

  try {
    const instruments = await GetInstruments();
    set(instruments);
    instrumentsLoaded = true;
  } catch (err) {
    console.error("Failed to load instruments:", err);
    // Keep the store as an empty array on error
  }
}

// Export a custom store that includes the load function
export const instrumentStore = {
  subscribe,
  load: loadInstruments,
  // Helper function to get an instrument name by its ID
  getNameById: (id: number): string => {
    const instruments = get({ subscribe });
    const instrument = instruments.find((inst) => inst.id === id);
    return instrument ? instrument.name : "Unknown";
  },
};
