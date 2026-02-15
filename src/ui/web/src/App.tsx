import './App.css'
import './Prompt.css'
import type { Component } from 'solid-js';
import { onMount, createSignal } from "solid-js";
import Comp from './Comp';
import { For } from "solid-js"

const API_URL = import.meta.env.VITE_API_URL;
const WEBSOCKET_URL = API_URL.replace(/^http/, 'ws')

const Prompt: Component<{
  focus?: boolean,
  afterStartingSession?: () => void
}> = (props) => {
  let inputRef: HTMLInputElement | undefined;
  let outputRef: HTMLDivElement | undefined;

  onMount(() => {
    if (props.focus && inputRef) {
      inputRef.focus();
    }
  });

  const appendOutput = (text: string) => {
    const node = outputRef;
    node.textContent += text;
    node.scrollTop = node.scrollHeight;
  };

  let ws: WebSocket | null = null;
  const startSession = (cmdId: number) => {
    ws = new WebSocket(`${WEBSOCKET_URL}/ws/${cmdId}`);
    ws.onopen = () => {
      if (outputRef) {
        outputRef.classList.remove("hidden")
      }
      console.log("WebSocket connected");
    };

    ws.onclose = () => {
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
    inputRef.disabled = true; // TODO: loading animation
    const argv = parseInput(cmd)

    try {
      const res = await fetch(`${API_URL}/run`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ argv }),
      });
      const data = await res.json()
      const cmdId = data.id
      startSession(cmdId)
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
      <div class="prompt-output hidden" ref={outputRef}></div>
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
