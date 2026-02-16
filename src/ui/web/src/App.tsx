import './App.css'
import './Prompt.css'
import './Output.css'
import type { Component } from 'solid-js';
import { onMount, onCleanup, createSignal, createEffect} from "solid-js";
import Comp from './Comp';
import { For } from "solid-js"

const API_URL = import.meta.env.VITE_API_URL;
const WEBSOCKET_URL = API_URL.replace(/^http/, 'ws')

interface ProcessMetadata {
  cmdId: number
  pid?: number
  status: string
  exitCode?: number
  startedAt: Date
  exitedAt?: Date
}

const Output: Component<{
  command: string
  metadata?: ProcessMetadata
  setRef?: (el: HTMLDivElement) => void,
  hidden: bool
}> = (props) => {
  let outputRef: HTMLDivElement | undefined;

  const formatDuration = (start: Date, end?: Date): string => {
    const endTime = end || new Date();
    const ms = endTime.getTime() - start.getTime();
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    const mins = Math.floor(ms / 60000);
    const secs = Math.floor((ms % 60000) / 1000);
    return `${mins}m ${secs}s`;
  };

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

  const getStatusIcon = (status: string, exitCode?: number): string => {
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
          setTimeElapsed(Date.now() - props.metadata!.startedAt.getTime());
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
        'status-running': props.metadata?.status === 'running',
        'status-success': props.metadata?.status === 'exited' && props.metadata?.exitCode === 0,
        'status-error': props.metadata?.status === 'exited' && props.metadata?.exitCode !== 0,
      }}
    >
      <div class="output-header">
        <div class="output-header-left">
          <span class="status-icon">
            {props.metadata ? getStatusIcon(props.metadata.status, props.metadata.exitCode) : '○'}
          </span>
          <span class="command">{props.command}</span>
          <span class="status-text">{props.metadata?.status || 'pending'}</span>
          {props.metadata?.exitCode != null && (
            <span class="exit-code">• {props.metadata.exitCode}</span>
          )}
        </div>
        <div class="output-header-right">
          <button
            class="copy-btn"
            onClick={() => {
              const text = outputRef.textContent
              navigator.clipboard.writeText(text);
            }}
            title="Copy output"
            >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
              <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
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
    </div>
  );

}

const Prompt: Component<{
  focus?: boolean,
  afterStartingSession?: () => void
}> = (props) => {
  const [showOutput, setShowOutput] = createSignal<boolean>(false)
  const [metadata, setMetadata] = createSignal<ProcessMetadata | null>(null)
  const [command, setCommand] = createSignal<string>("") 

  let inputRef: HTMLInputElement | undefined;
  let outputRef: HTMLDivElement | undefined;
  let cmdId: number | undefined

  onMount(() => {
    if (props.focus && inputRef) {
      inputRef.focus();
    }
  });

  // TODO: handle errors
  const updateMetadata = async () => {
    const res = await fetch(`${API_URL}/cmd/${cmdId}/metadata`);
    const data = await res.json();
    const md = data.metadata
    const metadata = {
      ...md,
      startedAt: new Date(md.startedAt),
      exitedAt: md.exitedAt ? new Date(md.exitedAt) : undefined,
    }
    setMetadata(metadata);
  }

  const appendOutput = (text: string) => {
    const node = outputRef;
    if (!node) return
    node.textContent += text;
    node.scrollTop = node.scrollHeight;
  };

  let ws: WebSocket | null = null;
  const startSession = (cmdId: number) => {
    ws = new WebSocket(`${WEBSOCKET_URL}/ws/${cmdId}`);
    ws.onopen = async () => {
      await updateMetadata()
      setShowOutput(true)
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
      appendOutput(text)
    };
  }

  const parseInput = (cmd: string) => {
    return cmd.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(a => a.replace(/"/g, "")) || [];
  };

  const handleKeyDown = async (e: KeyboardEvent) => {
    if (e.key != "Enter" || !inputRef) {
      return
    }
    const cmd = inputRef.value;
    inputRef.disabled = true;
    setCommand(cmd);
    const argv = parseInput(cmd);

    try {
      const res = await fetch(`${API_URL}/run`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ argv }),
      });
      const data = await res.json()
      cmdId = data.id
      startSession(data.id)
      if (props.afterStartingSession) {
        props.afterStartingSession()
      }
    } catch (err) {
      appendOutput(`Error: ${err}`);
    } finally {
      inputRef.disabled = true;
    }
  }

  return (
    <div class="prompt">
      <div class="prompt-input-row">
        <label>$</label>
        <input type="text" ref={inputRef} onKeyDown={handleKeyDown}/>
      </div>
      <Output
        hidden={!showOutput()}
        command={command()}
        metadata={metadata()}
        setRef={el => (outputRef = el)}
      />
    </div>
  );
};

const App: Component = () => {
  const [prompts, setPrompts] = createSignal<{id: number}[]>([{id: 0}]);

  const addPrompt = () => {
    const id = prompts().length;
    setPrompts([...prompts(), {id}]);
  }

  return (
    <div class="repl-container">
      <For each={prompts()}>
        {(p, i) => (
          <Prompt
            focus={i() === prompts().length - 1}
            afterStartingSession={addPrompt}
          />
        )}
      </For>
    </div>
  );
};

export default App;
