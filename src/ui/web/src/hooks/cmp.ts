export function getPrevSymbolRange(input: HTMLInputElement): { start: number; end: number } {
  const cursorPos = input.selectionStart ?? 0;
  const text = input.value;

  if (cursorPos === 0 || text[cursorPos - 1].trim() === "") {
    return { start: 0, end: 0 }; // whitespace or start → give up
  }

  // Find start of this symbol/word
  let start = cursorPos - 1;
  while (start >= 0 && text[start].trim() !== "") {
    start--;
  }

  return { start: start + 1, end: cursorPos };
}
