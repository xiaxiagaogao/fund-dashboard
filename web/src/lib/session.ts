import { writable } from "svelte/store";
import type { Me } from "./api";

export const session = writable<Me | null>(null);
