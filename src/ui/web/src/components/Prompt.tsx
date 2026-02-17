import './Prompt.css'
import { onMount, createSignal } from "solid-js";
import { For } from "solid-js"
import { Output } from "./Output"
import type { Component } from 'solid-js';
import * as cmd from '../hooks/cmd'

function parseInput(c: string): string {
  return c.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(a => a.replace(/"/g, "")) || [];
};

export const Prompt: Component<{
  focus?: boolean,
  afterStartingSession?: () => void
}> = (props) => {
  let inputRef: HTMLInputElement | undefined;
  let outputRef: HTMLDivElement | undefined;
  let cmdId: number | undefined
  let ws: Websocket | null

  const [showOutput, setShowOutput] = createSignal<boolean>(false)
  const [metadata, startCmd] = cmd.useCmdSession()
  const [command, setCommand] = createSignal<string>("")

  onMount(() => {
    if (props.focus && inputRef) {
      inputRef.focus();
    }
  });

  // TODO: handle errors

  const handleKeyDown = async (e: KeyboardEvent) => {
    if (e.key != "Enter" || !inputRef) {
      return
    }
    const c = inputRef.value;
    inputRef.disabled = true;
    setCommand(c);
    const argv = parseInput(c);

    // TODO: handle error?
    const cmdId = await cmd.register(argv)
    ws = startCmd(cmdId, outputRef)
    setShowOutput(true)
    if (props.afterStartingSession) {
      props.afterStartingSession()
    }

  }

  return (
    <div class="prompt">
      <div class="prompt-input-row">
        <label>$</label>
        <input type="text" ref={inputRef} onKeyDown={handleKeyDown}/>
      </div>
      <Output
        hidden={!showOutput()}
        command={command()}
        metadata={metadata()}
        setRef={el => (outputRef = el)}
        onSendStdin={() => cmd.sendStdin(ws)}
        onSendEOF={() => cmd.sendEOF(ws)}
      />
    </div>
  );
};
