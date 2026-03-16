import * as config from '../config'

export async function complete(src: string, cursor: number): Promise<string[]> {
  const res = await fetch(`${config.API_URL}/complete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ src, cursor })
  })
  const data = await res.json()
  return data.completions ?? []
}
