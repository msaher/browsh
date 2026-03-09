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

// export interface Metadata {
//   cmdId: number
//   pid?: number
//   status: string
//   exitCode?: number
//   startedAt: Date
//   exitedAt?: Date
// }

export interface Job {
  id: number
  args: string
  // metadata: Metadata
  isPaused: boolean
  ws: WebSocket
}

export function sendStdin(j: Job | null, text: string) {
  if (!j) return
  if (j.ws && j.ws.readyState === WebSocket.OPEN) {
    j.ws.send(text);
  }
}

export function sendEOF(j: Job | null) {
  if (!j) return
  if (j.ws && j.ws.readyState === WebSocket.OPEN) {
    j.ws.send('\x04'); // ascii EOT character
  }
}

export async function signal(j: Job, signal: JobSignal) {
  await fetch(`${config.API_URL}/signal/${j.metadata.pid}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ signal })
  });
  if (signal === SIGNALS.STOP) j.isPaused = true
  if (signal === SIGNALS.CONT) j.isPaused = false
}

export async function register(src: string): Promise<number> {
  const res = await fetch(`${config.API_URL}/job/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ src }),
  });
  const data = await res.json()
  return data.id
}

export function useJobSession() {
  const [job, setJob] = createSignal<Job | null>(null)

  function startJob(id: number, args: string, onMessage: (e: MessageEvent) => void) {
    const ws = new WebSocket(`${config.WEBSOCKET_URL}/job/${id}/ws`);
    ws.onopen = async () => {
      // await updateMetadata()
      console.log("WebSocket connected");
    };

    ws.onclose = async () => {
      // await updateMetadata()
      console.log("WebSocket closed");
    };

    ws.onerror = (err) => {
      console.error("WebSocket error:", err);
    };

    ws.onmessage = onMessage

    const j: Job = {
      id: Id,
      args: args,
      // metadata: {status: "waiting", cmdId: cmdId, startedAt: new Date()},
      isPaused: false,
      ws: ws,
    }
    setCmd(c)
  }

  // async function updateMetadata() {
  //   const j = job()!
  //   const res = await fetch(`${config.API_URL}/job/${j.id}/metadata`);
  //   const data = await res.json();
  //   const md = data.metadata
  //   const metadata = {
  //     ...md,
  //     startedAt: new Date(data.metadata.startedAt),
  //     exitedAt: md.exitedAt ? new Date(data.metadata.exitedAt) : undefined,
  //   }
  //   setJob({ ...c, metadata })
  // }

  async function togglePause() {
    const j = job()
    if (!j) return
    await signal(j, j.isPaused ? SIGNALS.CONT : SIGNALS.STOP)
    setJob({ ...j })
  }

  return [job, startJob, togglePause] as const
}
