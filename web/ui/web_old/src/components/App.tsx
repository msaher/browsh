import './App.css'
import type { Component } from 'solid-js';
import { Prompt } from './Prompt'
import { For, createSignal } from "solid-js";

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
