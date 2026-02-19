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
  }

  // blank
  const text = input.value
  if (text[cursorPos-1].trim() === "") {
    input.value = input.value + item
    return
    // return  [cursorPos, cursorPos] as const
  }

  // find start of prev word
  start = cursorPos-1
  while (start >= 0 && text[start].trim() !== "") {
    start--;
  }

  input.value = input.value.slice(0, start+1) + item + input.value.slice(cursorPos)
  // return [start + 1, cursorPos]
}

export async function getCompletions(text: string): Promise<string[]> {
  const res = await fetch(`${config.API_URL}/complete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text })
  })
  const data = await res.json()
  return data.result
}

export function useCompletion() {
  const [items, setItems] = createSignal<string[]>([])
  const [activeIdx, setActiveIdx] = createSignal<number>(0)

  async function populate(text: string) {
    const result = await getCompletions(text)
    setItems(result)
  }

  function setNext() {
    const idx = (activeIdx() + 1) % items().length
    setActiveIdx(idx)
  }

  function setPrev() {
    const idx = (activeIdx() - 1) % items().length
    setActiveIdx(idx)
  }


  return {
    items,
    activeIdx,
    setActiveIdx,
    populate,
    setNext,
    setPrev,
  }
}
