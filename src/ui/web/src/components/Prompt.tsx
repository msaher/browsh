import './Prompt.css'
import { onMount, createSignal } from "solid-js";
import { For, Show } from "solid-js"
import { Output } from "./Output"
import type { Component } from 'solid-js';
import * as cmds from '../hooks/cmds'

function parseInput(c: string): string[] {
  return c.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(a => a.replace(/"/g, "")) || [];
};

function appendOutput(text: string, outputRef: HTMLDivElement | undefined) {
  if (!outputRef) return
  outputRef.textContent += text;
  outputRef.scrollTop = outputRef.scrollHeight;
}

export const Prompt: Component<{
  focus?: boolean,
  afterStartingSession?: () => void
}> = (props) => {
  let inputRef: HTMLInputElement | undefined;
  let outputRef: HTMLDivElement | undefined;

  const [cmd, startCmd, togglePause] = cmds.useCmdSession()

  onMount(() => {
    if (props.focus && inputRef) {
      inputRef.focus();
    }
  });

  const handleKeyDown = async (e: KeyboardEvent) => {
    if (e.key != "Enter" || !inputRef) {
      return
    }
    inputRef.disabled = true;
    const args = inputRef.value;
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
  }

  // TODO: fix dumb cmd!
  return (
    <div class="prompt">
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
