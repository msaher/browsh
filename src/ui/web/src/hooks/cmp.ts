import * as config from "../config"
import { createSignal, type Signal } from "solid-js";

export function addCompletion(input: HTMLInputElement, item: string) {
  let start = 0;
  let end = 0;
