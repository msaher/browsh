import './Prompt.css'
import { onMount, createSignal} from "solid-js";
import { For } from "solid-js"
import { Output } from "./Output"
import type { Component } from 'solid-js';

const API_URL = import.meta.env.VITE_API_URL;
const WEBSOCKET_URL = API_URL.replace(/^http/, 'ws')

export const Prompt: Component<{
  focus?: boolean,
  afterStartingSession?: () => void
}> = (props) => {
  const [showOutput, setShowOutput] = createSignal<boolean>(false)
  const [metadata, setMetadata] = createSignal<ProcessMetadata | null>(null)
  const [command, setCommand] = createSignal<string>("")

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
    const md = data.metadata
    const metadata = {
      ...md,
      startedAt: new Date(md.startedAt),
      exitedAt: md.exitedAt ? new Date(md.exitedAt) : undefined,
    }
    setMetadata(metadata);
  }

  const appendOutput = (text: string) => {
    const node = outputRef;
    if (!node) return
    node.textContent += text;
    node.scrollTop = node.scrollHeight;
  };

  const sendStdin = (text: string) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(text);
    }
  };

  const sendEOF = () => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send('\x04'); // ascii EOT character
    }
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
      console.log(metadata())
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
    setCommand(cmd);
    const argv = parseInput(cmd);

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
        command={command()}
        metadata={metadata()}
        setRef={el => (outputRef = el)}
        onSendStdin={sendStdin}
        onSendEOF={sendEOF}
      />
    </div>
  );
};
