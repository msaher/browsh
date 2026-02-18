import * as config from "../config"
import { createSignal, type Signal } from "solid-js";

export function addCompletion(input: HTMLInputElement, item: string) {
  let start = 0;
  let end = 0;
  const cursorPos = input.selectionStart ?? 0;
  // insert
  if (cursorPos === 0) {
    input.value = item
    // return [0, 0] as const
    return
