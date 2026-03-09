import * as config from '../config'

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

export async function startJob(id: number) {
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

  ws.onmessage = async (event: MessageEvent) => {
    console.log(event.data)
  }

  return ws
}

export async function registerAndStartJob(src: string) {
  const id = await registerJob(src)
  const ws = await startJob(id)
  return ws
}

// class Job {
//   ws: WebSocket
//
//   constructor(src: string) {
//     this.src = src
//   }
// }
