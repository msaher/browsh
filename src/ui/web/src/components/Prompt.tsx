import './Prompt.css'
import { onMount, createSignal } from "solid-js";
import { For, Show } from "solid-js"
import { Output } from "./Output"
import { CompletionList } from './CompletionList'
import type { Component } from 'solid-js';
import * as cmds from '../hooks/cmds'
import * as cmp from '../hooks/cmp'

interface CompletionRequest {
  line: string;
  cursor: number;
}

function getCompletionContext(input: HTMLInputElement) {
  const line = input.value;
  const cursor = input.selectionStart || 0;

  // find word boundaries around cursor
  // look backwards for word start (space, start of line, or special chars)
  let wordStart = cursor;
  while (wordStart > 0 && !/\s/.test(line[wordStart - 1])) {
    wordStart--;
  }

  // look forwards for word end (space, end of line)
  let wordEnd = cursor;
  while (wordEnd < line.length && !/\s/.test(line[wordEnd])) {
    wordEnd++;
  }

  const prefix = line.substring(wordStart, cursor);

  return { line, cursor, wordStart, wordEnd, prefix };
}

function parseInput(c: string): string[] {
  return c.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(a => a.replace(/"/g, "")) || [];
};

function appendOutput(text: string, outputRef: HTMLDivElement | undefined) {
  if (!outputRef) return
  outputRef.innerHTML += text;
  outputRef.scrollTop = outputRef.scrollHeight;
}

export const Prompt: Component<{
  focus?: boolean,
  afterStartingSession?: () => void
}> = (props) => {
  let inputRef: HTMLInputElement | undefined;
  let outputRef: HTMLDivElement | undefined;

  const [cmd, startCmd, togglePause] = cmds.useCmdSession()

  // TODO: maybe make this a hook
  const [suggestions, setSuggestions] = createSignal<string[]>(["one", "two", "three"]);
  const [activeIndex, setActiveIndex] = createSignal(0);
  const [showCompletion, setShowCompletion] = createSignal(false)

  onMount(() => {
    if (props.focus && inputRef) {
      inputRef.focus();
    }
  });

  const onEnter = async () => {
    const input = inputRef!
    input.disabled = true;
    const args = input.value;
    const argv = parseInput(args);

    // TODO: handle error?
    const cmdId = await cmds.register(argv)
    startCmd(cmdId, args, async (event: MessageEvent) => {
      const text = await event.data.text()
      appendOutput(text, outputRef)
    })
    if (props.afterStartingSession) {
      props.afterStartingSession()
    }
  };

  const onTab = () => {
    const input = inputRef!

    if (suggestions().length === 0) {
      return
    }

    if (!showCompletion()) {
      // TODO: maybe make an obj intead of this
      setShowCompletion(true)
      setActiveIndex(0)
    } else {
      const nextIdx = (activeIndex() + 1) % suggestions().length
      setActiveIndex(nextIdx)
    }

    const {start, end} = cmp.getPrevSymbolRange(input)
    const sug = suggestions()[activeIndex()]

    // replace
    input.value = input.value.slice(0, start) + sug + input.value.slice(end);
  };

  const handleKeyDown = async (e: KeyboardEvent) => {
    if (!inputRef) {
      return
    }

    if (e.key !== "Tab" && showCompletion()) {
      setShowCompletion(false)
    }

    if (e.key === "Enter") {
      onEnter()
    } else if (e.key === "Tab") {
      e.preventDefault()
      onTab()
    }

  }

  return (
    <div class="prompt">
      <Show when={showCompletion()}>
        <CompletionList
          suggestions={suggestions()}
          activeIndex={activeIndex()}
        />
      </Show>
      <div class="prompt-input-row">
        <label>$</label>
        <input type="text" ref={inputRef} onKeyDown={handleKeyDown}/>
      </div>
      <Show when={cmd() !== null}>
        <Output
        cmd={cmd()!}
        togglePause={togglePause}
        setRef={el => (outputRef = el)}
        />
      </Show>
    </div>
  );
};
