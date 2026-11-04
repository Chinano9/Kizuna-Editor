import { test, describe } from "node:test";
import assert from "node:assert";
import { get } from "svelte/store";
import { toastStore } from "../src/shared/stores/toastStore.ts";

describe("Toast Notification Store", () => {
    test("Can add success, error, warning, and info toasts", () => {
        toastStore.clear();

        const sId = toastStore.success("Saved!");
        const eId = toastStore.error("Something went wrong!");
        const wId = toastStore.warning("Attention required!");
        const iId = toastStore.info("FYI");

        const list = get(toastStore);
        assert.strictEqual(list.length, 4);
        assert.strictEqual(list[0].type, "success");
        assert.strictEqual(list[0].message, "Saved!");
        assert.strictEqual(list[1].type, "error");
        assert.strictEqual(list[2].type, "warning");
        assert.strictEqual(list[3].type, "info");
    });

    test("Can dismiss specific toast by ID", () => {
        toastStore.clear();
        const id1 = toastStore.info("Message 1");
        const id2 = toastStore.info("Message 2");

        assert.strictEqual(get(toastStore).length, 2);

        toastStore.dismiss(id1);

        const list = get(toastStore);
        assert.strictEqual(list.length, 1);
        assert.strictEqual(list[0].id, id2);
    });

    test("Can clear all toasts", () => {
        toastStore.info("1");
        toastStore.info("2");
        toastStore.clear();
        assert.strictEqual(get(toastStore).length, 0);
    });
});
