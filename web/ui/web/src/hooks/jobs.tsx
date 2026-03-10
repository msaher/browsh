import * as config from '../config'

interface Job {
  id: number
  ws: WebSocket
  status: 'running' | 'exited'
  exitCode: number
}

export function isDone(j: Job | undefined) {
  if (!j) return false
  return j.exitCode !== -1
}

// TODO: handle errors

export async function registerJob(src: string) {
  const res = await fetch(`${config.API_URL}/job/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ src })
  });
  const data = await res.json()
  return data.id
}

export async function startJob(id: number, onmessage: (event: MessageEvent) => void) {
  const ws = new WebSocket(`${config.WEBSOCKET_URL}/job/${id}/ws`);
  ws.onopen = async () => {
    console.log("WebSocket connected");
  }

  ws.onclose = async () => {
    // await updateMetadata()
    console.log("WebSocket closed");
  };

  ws.onerror = (err) => {
    console.error("WebSocket error:", err);
  };

  ws.onmessage = onmessage

  return ws
}

export async function registerAndStartJob(src: string, onmessage: (e: MessageEvent) => void) {
  const id = await registerJob(src)
  const ws = await startJob(id, onmessage)
  const job = {id, ws, exitCode: -1}
  return job
}
