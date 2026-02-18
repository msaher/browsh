import * as config from "../config"
import { createSignal, type Signal } from "solid-js";

export const SIGNALS = {
  TERM: "SIGTERM",
  KILL: "SIGKILL",
  STOP: "SIGSTOP",
  CONT: "SIGCONT",
  INT:  "SIGINT",
} as const;

export type CmdSignal = typeof SIGNALS[keyof typeof SIGNALS];

export interface Metadata {
  cmdId: number
  pid?: number
  status: string
  exitCode?: number
  startedAt: Date
  exitedAt?: Date
}

export interface Cmd {
  id: number
  args: string
  metadata: Metadata
  isPaused: boolean
  ws: WebSocket
}

export function sendStdin(c: Cmd | null, text: string) {
  if (!c) return
  if (c.ws && c.ws.readyState === WebSocket.OPEN) {
    c.ws.send(text);
  }
}

export function sendEOF(c: Cmd | null) {
  if (!c) return
  if (c.ws && c.ws.readyState === WebSocket.OPEN) {
    c.ws.send('\x04'); // ascii EOT character
  }
}

export async function signal(c: Cmd, signal: CmdSignal) {
  await fetch(`${config.API_URL}/signal/${c.metadata.pid}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ signal })
  });
  if (signal === SIGNALS.STOP) c.isPaused = true
  if (signal === SIGNALS.CONT) c.isPaused = false
}

export async function register(argv: string[]): Promise<number> {
  const res = await fetch(`${config.API_URL}/run`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ argv }),
  });
  const data = await res.json()
  return data.id
}

export function useCmdSession() {
  const [cmd, setCmd] = createSignal<Cmd | null>(null)

  function startCmd(cmdId: number, args: string, onMessage: (e: MessageEvent) => void) {
    const ws = new WebSocket(`${config.WEBSOCKET_URL}/ws/${cmdId}`);
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

    ws.onmessage = onMessage

    const c: Cmd = {
      id: cmdId,
      args: args,
      metadata: {status: "waiting", cmdId: cmdId, startedAt: new Date()},
      isPaused: false,
      ws: ws,
    }
    setCmd(c)
  }

  async function updateMetadata() {
    const c = cmd()!
    const res = await fetch(`${config.API_URL}/cmd/${c.id}/metadata`);
    const data = await res.json();
    const md = data.metadata
    const metadata = {
      ...md,
      startedAt: new Date(data.metadata.startedAt),
      exitedAt: md.exitedAt ? new Date(data.metadata.exitedAt) : undefined,
    }
    setCmd({ ...c, metadata })
  }

  async function togglePause() {
    const c = cmd()
    if (!c) return
    await signal(c, c.isPaused ? SIGNALS.CONT : SIGNALS.STOP)
    setCmd({ ...c })
  }

  return [cmd, startCmd, togglePause] as const
}
