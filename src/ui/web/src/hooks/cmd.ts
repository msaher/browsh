import * as config from "../config"
import * as solid from "solid-js";

export interface ProcessMetadata {
  cmdId: number
  pid?: number
  status: string
  exitCode?: number
  startedAt: Date
  exitedAt?: Date
}

export async function register(argv: string[]): number {
  const res = await fetch(`${config.API_URL}/run`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ argv }),
  });
  const data = await res.json()
  return data.id
}

export function sendStdin(ws: WebSocket | null = null) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(text);
  }
}

export function sendEOF(ws: WebSocket | null = null) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send('\x04'); // ascii EOT character
  }
}

function appendOutput(text: string, outputRef: HTMLDivElement | undefined) {
  if (!outputRef) return
  outputRef.textContent += text;
  outputRef.scrollTop = outputRef.scrollHeight;
}

// outputRef is a bit of a leaky abstraction but I want to avoid
// having callbacks for appending for better performance
export function useCmdSession() {
  const [metadata, setMetadata] = solid.createSignal<ProcessMetadata | null>(null)
  let ws: WebSocket | null = null;
  let cmdId: number | undefined

  function startCmd(id: number, outputRef: HTMLDivElement | undefined): WebSocket {
    cmdId = id
    ws = new WebSocket(`${config.WEBSOCKET_URL}/ws/${cmdId}`);
    ws.onopen = async () => {
      await updateMetadata()
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
      appendOutput(text, outputRef)
    };

    return ws
  }

  async function updateMetadata() {
    const res = await fetch(`${config.API_URL}/cmd/${cmdId}/metadata`);
    const data = await res.json();
    const md = data.metadata
    const metadata = {
      ...md,
      startedAt: new Date(md.startedAt),
      exitedAt: md.exitedAt ? new Date(md.exitedAt) : undefined,
    }
    setMetadata(metadata);
  }

  return [metadata, startCmd]

}
