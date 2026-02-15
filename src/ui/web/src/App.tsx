import './App.css'
import './Prompt.css'
import './Output.css'
import type { Component } from 'solid-js';
import { onMount, createSignal } from "solid-js";
import Comp from './Comp';
import { For } from "solid-js"

const API_URL = import.meta.env.VITE_API_URL;
const WEBSOCKET_URL = API_URL.replace(/^http/, 'ws')

interface ProcessMetadata {
  cmdId: number
  pid?: number
  status: string
  exitCode?: number;
}

const Output: Component<{
  metadata?: ProcessMetadata
  setRef?: (el: HTMLDivElement) => void,
  hidden: bool
}> = (props) => {

  return (
    <div
      class="output-container"
      classList={{
        hidden: props.hidden
      }}
    >
      <div class="output-header">
        <div class="output-header">
          <span>Status: {props.metadata ? props.metadata.status : "pending"}</span>
          {props.metadata?.pid != null && <span> | PID: {props.metadata.pid}</span>}
          {props.metadata?.exitCode != null && <span> | Exit: {props.metadata.exitCode}</span>}
        </div>
      </div>
      <div class="output" ref={el => props.setRef?.(el)} />
    </div>
  );
}

const Prompt: Component<{
  focus?: boolean,
  afterStartingSession?: () => void
}> = (props) => {
  const [showOutput, setShowOutput] = createSignal<boolean>(false)
  const [metadata, setMetadata] = createSignal<ProcessMetadata | null>(null)

  let inputRef: HTMLInputElement | undefined;
  let outputRef: HTMLDivElement | undefined;
  let cmdId: number | undefined

  onMount(() => {
    if (props.focus && inputRef) {
      inputRef.focus();
    }
  });

  // TODO: handle errors
  const updateMetadata = async () => {
    const res = await fetch(`${API_URL}/cmd/${cmdId}/metadata`);
    const data = await res.json();
    setMetadata(data.metadata);
  }

  const appendOutput = (text: string) => {
    const node = outputRef;
    if (!node) return
    node.textContent += text;
    node.scrollTop = node.scrollHeight;
  };

  let ws: WebSocket | null = null;
  const startSession = (cmdId: number) => {
    ws = new WebSocket(`${WEBSOCKET_URL}/ws/${cmdId}`);
    ws.onopen = async () => {
      await updateMetadata()
      setShowOutput(true)
      console.log("WebSocket connected");
    };

    ws.onclose = async () => {
      await updateMetadata()
      console.log("WebSocket closed");
    };

    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
    };

    ws.onmessage = async (event: MessageEvent) => {
      const text = await event.data.text()
      appendOutput(text)
    };
  }

  const parseInput = (cmd: string) => {
    return cmd.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(a => a.replace(/"/g, "")) || [];
  };

  const handleKeyDown = async (e: KeyboardEvent) => {
    if (e.key != "Enter" || !inputRef) {
      return
    }
    const cmd = inputRef.value;
    inputRef.disabled = true;
    const argv = parseInput(cmd)

    try {
      const res = await fetch(`${API_URL}/run`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ argv }),
      });
      const data = await res.json()
      cmdId = data.id
      startSession(data.id)
      if (props.afterStartingSession) {
        props.afterStartingSession()
      }
    } catch (err) {
      appendOutput(`Error: ${err}`);
    } finally {
      inputRef.disabled = true;
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
        metadata={metadata()}
        setRef={el => (outputRef = el)}
      />
    </div>
  );
};

const App: Component = () => {
  const [prompts, setPrompts] = createSignal<{id: number}[]>([{id: 0}]);

  const addPrompt = () => {
    const id = prompts().length;
    setPrompts([...prompts(), {id}]);
  }

  return (
    <div class="repl-container">
      <For each={prompts()}>
        {(p, i) => (
          <Prompt
            focus={i() === prompts().length - 1}
            afterStartingSession={addPrompt}
          />
        )}
      </For>
    </div>
  );
};

export default App;
