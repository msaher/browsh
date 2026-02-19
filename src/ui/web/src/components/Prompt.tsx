import './Prompt.css'
import { onMount, createSignal } from "solid-js";
import { For, Show } from "solid-js"
import { Output } from "./Output"
import { CompletionList } from './CompletionList'
import type { Component } from 'solid-js';
import * as cmds from '../hooks/cmds'
import * as cmp from '../hooks/cmp'

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
  const [showCompletion, setShowCompletion] = createSignal(false)
  const comp = cmp.useCompletion()

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

  const onTab = async () => {
    const input = inputRef!

    if (!showCompletion()) {
      await comp.populate(input.value)
      setShowCompletion(true)
    } else {
      comp.setNext()
    }

    const item = comp.items()[comp.activeIdx()]
    cmp.addCompletion(input, item)
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
          suggestions={comp.items()}
          activeIdx={comp.activeIdx()}
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
