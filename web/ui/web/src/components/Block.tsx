import { onMount, createSignal } from "solid-js";
import Prompt from './Prompt';
import * as jobs from '../hooks/jobs'

function StopIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <rect x="6" y="6" width="12" height="12" rx="2"/>
    </svg>
  )
}

function ForceKillIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <circle cx="12" cy="12" r="10"/>
    <line x1="15" y1="9" x2="9" y2="15"/>
    <line x1="9" y1="9" x2="15" y2="15"/>
    </svg>
  )
}

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
    </svg>
  )
}

function PauseIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <rect x="6" y="4" width="4" height="16"/>
    <rect x="14" y="4" width="4" height="16"/>
    </svg>
  )
}

function ResumeIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <polygon points="5 3 19 12 5 21 5 3"/>
    </svg>
  )
}

export interface BlockProps {
  src: string
}

export default function Block(props: BlockProps) {
  const [job, setJob] = createSignal<jobs.Job | undefined>(undefined)
  let outputRef: HTMLDivElement | undefined;

  function onmessage(event: MessageEvent) {
    if (!outputRef || !event.data) return;
    const msg = JSON.parse(event.data)

    if (msg.stream === 'control') {
      setJob({
        ...job(),
        exitCode: msg.exitCode
      })
      return
    }

    if (msg.data) {
      outputRef.innerHTML += msg.data;
      outputRef.scrollTop = outputRef.scrollHeight;
    }
  }

  onMount(async () => {
    const j = await jobs.registerAndStartJob(props.src, onmessage)
    setJob(j)
  })

  return (
    <div>
      <div>{jobs.isDone(job()) && `exit status: ${job().exitCode}`}</div>
      <div ref={el => (outputRef = el)}></div>
    </div>
  )
}
