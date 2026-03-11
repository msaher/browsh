import { onMount, createSignal, Show } from "solid-js";
import * as jobs from '../hooks/jobs'
import './Block.css'

function StopIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="6" y="6" width="12" height="12" rx="2"/>
    </svg>
  )
}

function ForceKillIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10"/>
      <line x1="15" y1="9" x2="9" y2="15"/>
      <line x1="9" y1="9" x2="15" y2="15"/>
    </svg>
  )
}

function CopyIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
    </svg>
  )
}

function PauseIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <rect x="6" y="4" width="4" height="16"/>
      <rect x="14" y="4" width="4" height="16"/>
    </svg>
  )
}

function ResumeIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <polygon points="5 3 19 12 5 21 5 3"/>
    </svg>
  )
}

export interface BlockProps {
  src: string
}

type BlockStatus = 'running' | 'paused' | 'ok' | 'err'

export default function Block(props: BlockProps) {
  const [job, setJob] = createSignal<jobs.Job | undefined>(undefined)
  const [status, setStatus] = createSignal<BlockStatus>('running')
  let outputRef: HTMLDivElement | undefined

  const isRunning = () => status() === 'running' || status() === 'paused'
  const isDone = () => status() === 'ok' || status() === 'err'
  const exitCode = () => job()?.exitCode ?? -1

  function onOutputMount(el: HTMLDivElement) {
    outputRef = el
    el.addEventListener('submit', (e) => {
      e.preventDefault()
      const form = e.target as HTMLFormElement
      const data = Object.fromEntries(new FormData(form))
      jobs.sendStdin(job(), JSON.stringify(data) + '\n')
    })
  }

  function onmessage(event: MessageEvent) {
    if (!outputRef || !event.data) return
    const msg = JSON.parse(event.data)
    if (msg.stream === 'control') {
      setJob({ ...job()!, exitCode: msg.exitCode, status: 'exited' })
      setStatus(msg.exitCode === 0 ? 'ok' : 'err')
      return
    }
    if (msg.data) {
      outputRef.insertAdjacentHTML('beforeend', msg.data)
      outputRef.scrollTop = outputRef.scrollHeight
    }
  }

  onMount(async () => {
    const j = await jobs.registerAndStartJob(props.src, onmessage)
    setJob(j)
  })

  function onStop() {
    jobs.signal(job(), "SIGINT")
  }

  function onKill() {
    jobs.signal(job(), "SIGKILL")
  }

  function onPauseResume() {
    if (status() === 'paused') {
      jobs.signal(job(), "SIGCONT")
      setStatus('running')
    } else {
      jobs.signal(job(), "SIGSTOP")
      setStatus('paused')
    }
  }

  function onCopy() {
    navigator.clipboard.writeText(outputRef?.innerText ?? '')
  }

  function inputOnKeydown(e: any) {
    if (e.key === 'Enter') {
      e.preventDefault()
      const val = e.currentTarget.value
      if (!val) return
        jobs.sendStdin(job(), val + '\n')
      e.currentTarget.value = ''
    }
    if (e.key === 'd' && e.ctrlKey) {
      e.preventDefault()
      jobs.sendEOF(job())
      e.currentTarget.value = ''
    }
  }

  return (
    <div classList={{ block: true, running: status() === 'running', paused: status() === 'paused', ok: status() === 'ok', err: status() === 'err' }}>
      <div class="block-header">
        <div class="block-src">
          <span class="block-sigil">❯</span>
          <span class="block-cmd">{props.src}</span>
        </div>
        <div class="block-meta">
          <Show when={isDone()}>
            <span classList={{ 'exit-badge': true, ok: status() === 'ok', err: status() === 'err' }}>
              {status() === 'ok' ? '✓' : `✗ ${exitCode()}`}
            </span>
          </Show>
          <Show when={status() === 'running'}>
            <span class="running-dot" />
          </Show>
          <Show when={status() === 'paused'}>
            <span class="paused-label">paused</span>
          </Show>
          <div class="block-actions">
            <Show when={isRunning()}>
              <button class="action-btn" onClick={onPauseResume} title={status() === 'paused' ? 'Resume' : 'Pause'}>
                {status() === 'paused' ? <ResumeIcon /> : <PauseIcon />}
              </button>
              <button class="action-btn" onClick={onStop} title="Stop">
                <StopIcon />
              </button>
              <button classList={{ 'action-btn': true, danger: true }} onClick={onKill} title="Force kill">
                <ForceKillIcon />
              </button>
            </Show>
            <button class="action-btn" onClick={onCopy} title="Copy output">
              <CopyIcon />
            </button>
          </div>
        </div>
      </div>
      <div ref={onOutputMount} class="block-output" />
      <Show when={isRunning()}>
        <div class="block-stdin">
          <span class="stdin-sigil">›</span>
          <input
            class="stdin-input"
            type="text"
            placeholder="stdin..."
            spellcheck={false}
            autocomplete="off"
            onKeyDown={inputOnKeydown}
          />
          <button class="eof-btn" onClick={() => jobs.sendEOF(job())} title="Send EOF (Ctrl+D)">
            EOF
          </button>
        </div>
      </Show>
    </div>
  )
}
