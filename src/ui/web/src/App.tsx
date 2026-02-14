import './App.css'
import './Prompt.css'
import type { Component } from 'solid-js';
import { onMount, createSignal } from "solid-js";
import Comp from './Comp';

const API_URL = import.meta.env.VITE_API_URL;

const Prompt: Component<{ focus?: boolean }> = (props) => {
  let inputRef: HTMLInputElement | undefined;
  const [output, setOutput] = createSignal("");

  onMount(() => {
    if (props.focus && inputRef) {
      inputRef.focus();
    }
  });

  const parseInput = (cmd: string) => {
    return cmd.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(a => a.replace(/"/g, "")) || [];
  };

  const handleKeyDown = async (e: KeyboardEvent) => {
    if (e.key != "Enter" || !inputRef) {
      return
    }
    const cmd = inputRef.value;
    inputRef.disabled = true; // TODO: loading animation
    const argv = parseInput(cmd)

    try {
      const res = await fetch(`${API_URL}/run`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ argv }),
      });
      const text = await res.text();
      setOutput(text);
    } catch (err) {
      setOutput(`Error: ${err}`);
    } finally {
      inputRef.disabled = false;
      inputRef.value = "";
      inputRef.focus();
    }
  }

  return (
    <div class="prompt">
      <div class="prompt-input-row">
        <label>$</label>
        <input type="text" ref={inputRef} onKeyDown={handleKeyDown}/>
      </div>
      <div class="prompt-output">{output()}</div>
    </div>
  );
};

const App: Component = () => {
  return (
    <>
      <Prompt focus/>
    </>
  );
};

export default App;
