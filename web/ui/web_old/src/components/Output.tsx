import './Output.css'
import type { Component } from 'solid-js';
import { onCleanup, createSignal, createEffect} from "solid-js";
import * as jobs from "../hooks/jobs"

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

// TODO: rename. I don't like the name output
export const Output: Component<{
  job: jobs.Job,
  togglePause: () => void,
  setRef?: (el: HTMLDivElement) => void,
}> = (props) => {
  let outputRef: HTMLDivElement | undefined;

  // const getDuration = (): string => {
  //   if (!props.cmd.metadata.startedAt) return '';
  //   if (props.cmd.metadata.status === 'exited' && props.cmd.metadata.exitedAt) {
  //     // finished - show final duration
  //     return formatDuration(props.cmd.metadata.startedAt, props.cmd.metadata.exitedAt);
  //   } else {
  //     // running - show live counter
  //     return formatDuration(props.cmd.metadata.startedAt, new Date(props.cmd.metadata.startedAt.getTime() + timeElapsed()));
  //   }
  // };

  // for non STOP/CONT signals
  // const handleSignal = async (signal: cmds.CmdSignal) => {
  //   if (!props.cmd.metadata.pid) return;
  //   try {
  //     await cmds.signal(props.cmd, signal)
  //   } catch (err) {
  //     // TODO: handle error
  //     console.error('Failed to send signal:', err);
  //   }
  // };

  // reset pause state when process exits
  // createEffect(() => {
  //   if (props.cmd.metadata.status === 'exited') {
  //     setIsPaused(false);
  //   }
  // })

  const getStatusIcon = (status: string, paused: boolean, exitCode?: number): string => {
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
    if (props.cmd.metadata.status === 'running' && props.cmd.metadata.startedAt) {
      // start the interval if not already running
      if (!intervalId) {
        intervalId = setInterval(() => {
          if (props.cmd.isPaused) {
            setTimeElapsed(Date.now() - props.cmd!.metadata.startedAt.getTime());
          }
        }, 100);
      }
    } else if (props.cmd.metadata.status === 'exited') {
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
        'status-running': props.cmd.metadata.status === 'running' && !props.cmd.isPaused,
        'status-success': props.cmd.metadata.status === 'exited' && props.cmd.metadata.exitCode === 0,
        'status-error': props.cmd.metadata.status === 'exited' && props.cmd.metadata.exitCode !== 0,
        'status-paused': props.cmd.isPaused
      }}
    >
      <div class="output-header">
        <div class="output-header-left">
          <span class="status-icon">
            {getStatusIcon(props.cmd.metadata.status, props.cmd.isPaused, props.cmd.metadata.exitCode)}
          </span>
          <span class="command">{props.cmd.args}</span>
          <span class="status-text">
            {props.cmd.isPaused
              ? "paused"
              : props.cmd.metadata.status}
          </span>
          {props.cmd.metadata.exitCode != null && (
            <span class="exit-code">• {props.cmd.metadata.exitCode}</span>
          )}
        </div>
        <div class="output-header-right">
        {props.cmd.metadata.status === 'running' && (
          <>
          <button
            class="signal-btn"
            onClick={props.togglePause}
            title={props.cmd.isPaused ? "Resume (SIGCONT)" : "Pause (SIGSTOP)"}
          >
            {props.cmd.isPaused ? <ResumeIcon /> : <PauseIcon />}
          </button>
          <button class="signal-btn" onClick={() => handleSignal(cmds.SIGNALS.INT)} title="Stop (SIGINT)">
            <StopIcon />
          </button>
          <button class="signal-btn signal-btn-danger" onClick={() => handleSignal(cmds.SIGNALS.KILL)} title="Force kill (SIGKILL)">
            <ForceKillIcon />
            </button>
          </>
        )}

          <button
            class="copy-btn"
            onClick={() => {
              const text = outputRef!.textContent
              navigator.clipboard.writeText(text);
            }}
            title="Copy output"
            >
            <CopyIcon />
          </button>

          {props.cmd.metadata.startedAt && (
            <span class="duration">{getDuration()}</span>
          )}

          {props.cmd.metadata.pid != null && (
            <span class="pid">PID: {props.cmd.metadata.pid}</span>
          )}
        </div>
      </div>
      <div class="output" ref={el => {
        outputRef = el
        props.setRef?.(el)
      }}/>

      {props.cmd?.metadata.status === 'running' && (
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
              cmds.sendStdin(props.cmd, text + '\n')
              e.currentTarget.value = '';
            }
          }
          if (e.key === 'd' && e.ctrlKey) {
            e.preventDefault();
            cmds.sendEOF(props.cmd)
            console.log("sent eof")
            e.currentTarget.value = '';
          }
        }}
        />
        <button
        class="eof-btn"
        onClick={() => cmds.sendEOF(props.cmd)}
        title="Send EOF (Ctrl+D)"
        >
    EOF
  </button>
        </div>
      )}

    </div>
  );
}
