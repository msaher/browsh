import type { Component } from 'solid-js';
import { For, createSignal } from "solid-js";
import Prompt from './Prompt';
import Block from './Block'
import { BlockProps } from './Block'
import './App.css'

const App: Component = () => {
  const [blocks, setBlocks] = createSignal<BlockProps[]>([]);

  function onSubmit(src: string) {
    setBlocks([...blocks(), {src}]);
  }

  return (
    <div class="app">
      <Prompt onSubmit={onSubmit}/>
      <For each={blocks()}>
        {(b, i) => (
          <Block {...b}/>
        )}
      </For>
    </div>
  );
};

export default App;
