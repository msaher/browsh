import './Output.css'
import type { Component } from 'solid-js';
import { onCleanup, createSignal, createEffect} from "solid-js";

interface ProcessMetadata {
  cmdId: number
  pid?: number
  status: string
  exitCode?: number
  startedAt: Date
  exitedAt?: Date
}

const SIGNALS = {
  TERM: "SIGTERM",
  KILL: "SIGKILL",
  STOP: "SIGSTOP",
  CONT: "SIGCONT",
  INT:  "SIGINT",
} as const;

type Signal = typeof SIGNALS[keyof typeof SIGNALS];

const formatDuration = (start: Date, end?: Date): string => {
  const endTime = end || new Date();
  const ms = endTime.getTime() - start.getTime();
  // if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const mins = Math.floor(ms / 60000);
  const secs = Math.floor((ms % 60000) / 1000);
  return `${mins}m ${secs}s`;
};


const StopIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <rect x="6" y="6" width="12" height="12" rx="2"/>
  </svg>
);

const ForceKillIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <circle cx="12" cy="12" r="10"/>
    <line x1="15" y1="9" x2="9" y2="15"/>
    <line x1="9" y1="9" x2="15" y2="15"/>
  </svg>
);

const CopyIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
  </svg>
);

const PauseIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <rect x="6" y="4" width="4" height="16"/>
    <rect x="14" y="4" width="4" height="16"/>
  </svg>
);

const ResumeIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <polygon points="5 3 19 12 5 21 5 3"/>
  </svg>
);

export const Output: Component<{
  command: string,
  metadata?: ProcessMetadata,
  setRef?: (el: HTMLDivElement) => void,
  hidden: boolean,
  onSendStdin?: (text: string) => void,
  onSendEOF?: () => void,
}> = (props) => {
  let outputRef: HTMLDivElement | undefined;

  const getDuration = (): string => {
    if (!props.metadata?.startedAt) return '';
    if (props.metadata.status === 'exited' && props.metadata.exitedAt) {
      // finished - show final duration
      return formatDuration(props.metadata.startedAt, props.metadata.exitedAt);
    } else {
      // running - show live counter
      return formatDuration(props.metadata.startedAt, new Date(props.metadata.startedAt.getTime() + timeElapsed()));
    }
  };

  const [isPaused, setIsPaused] = createSignal(false);
  const handleSignal = async (signal: Signal) => {
    if (!props.metadata?.pid) return;
    try {
      await fetch(`${API_URL}/signal/${props.metadata.pid}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ signal })
      });

      if (signal === SIGNALS.STOP) setIsPaused(true);
      if (signal === SIGNALS.CONT) setIsPaused(false);

    } catch (err) {
      // TODO: handle error
      console.error('Failed to send signal:', err);
    }
  };

  // reset pause state when process exits
  createEffect(() => {
    if (props.metadata?.status === 'exited') {
      setIsPaused(false);
    }
  })


  const getStatusIcon = (status: string, exitCode?: number, paused: boolean): string => {
    if (paused) return '⏸';
    if (status === 'running') return '▶';
    if (status === 'exited' && exitCode === 0) return '✓';
    if (status === 'exited' && exitCode !== 0) return '✗';
    return '○';
  };

  // start a timer
  const [timeElapsed, setTimeElapsed] = createSignal<number>(0);
  let intervalId: number | undefined;
  createEffect(() => {
    if (props.metadata?.status === 'running' && props.metadata.startedAt) {
      // start the interval if not already running
      if (!intervalId) {
        intervalId = setInterval(() => {
          if (!isPaused()) {
            setTimeElapsed(Date.now() - props.metadata!.startedAt.getTime());
          }
        }, 100);
      }
    } else if (props.metadata?.status === 'exited') {
      // stop the interval when exited
      if (intervalId) {
        clearInterval(intervalId);
        intervalId = undefined;
      }
    }
  });

  onCleanup(() => {
    if (intervalId) clearInterval(intervalId);
  });

  return (
    <div
      class="output-container"
      classList={{
        hidden: props.hidden,
        'status-running': props.metadata?.status === 'running' && !isPaused(),
        'status-success': props.metadata?.status === 'exited' && props.metadata?.exitCode === 0,
        'status-error': props.metadata?.status === 'exited' && props.metadata?.exitCode !== 0,
        'status-paused': isPaused(),
      }}
    >
      <div class="output-header">
        <div class="output-header-left">
          <span class="status-icon">
            {props.metadata ? getStatusIcon(props.metadata.status, props.metadata.exitCode, isPaused()) : '○'}
          </span>
          <span class="command">{props.command}</span>
          <span class="status-text">
            {isPaused()
              ? "paused"
              : props.metadata?.status ?? "pending"}
          </span>
          {props.metadata?.exitCode != null && (
            <span class="exit-code">• {props.metadata.exitCode}</span>
          )}
        </div>
        <div class="output-header-right">
        {props.metadata?.status === 'running' && (
          <>
          <button
            class="signal-btn"
            onClick={() => handleSignal(isPaused() ? SIGNALS.CONT : SIGNALS.STOP)}
            title={isPaused() ? "Resume (SIGCONT)" : "Pause (SIGSTOP)"}
          >
            {isPaused() ? <ResumeIcon /> : <PauseIcon />}
          </button>
          <button class="signal-btn" onClick={() => handleSignal(SIGNALS.INT)} title="Stop (SIGINT)">
            <StopIcon />
          </button>
          <button class="signal-btn signal-btn-danger" onClick={() => handleSignal(SIGNALS.KILL)} title="Force kill (SIGKILL)">
            <ForceKillIcon />
            </button>
          </>
        )}

          <button
            class="copy-btn"
            onClick={() => {
              const text = outputRef.textContent
              navigator.clipboard.writeText(text);
            }}
            title="Copy output"
            >
            <CopyIcon />
          </button>

          {props.metadata?.startedAt && (
            <span class="duration">{getDuration()}</span>
          )}

          {props.metadata?.pid != null && (
            <span class="pid">PID: {props.metadata.pid}</span>
          )}
        </div>
      </div>
      <div class="output" ref={el => {
        outputRef = el
        props.setRef?.(el)
      }}/>

      {props.metadata?.status === 'running' && (
        <div class="stdin-container">
        <span class="stdin-prompt">›</span>
        <textarea
        class="stdin-input"
        placeholder="Type input and press Enter..."
        rows={1}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            const text = e.currentTarget.value;
            if (text.trim()) {
              props.onSendStdin?.(text + '\n');
              e.currentTarget.value = '';
            }
          }
          if (e.key === 'd' && e.ctrlKey) {
            e.preventDefault();
            props.onSendEOF?.();
            console.log("sent eof")
            e.currentTarget.value = '';
          }
        }}
        />
        <button
        class="eof-btn"
        onClick={() => props.onSendEOF?.()}
        title="Send EOF (Ctrl+D)"
        >
    EOF
  </button>
        </div>
      )}

    </div>
  );
}
