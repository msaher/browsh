import './CompletionList.css'
import { For, Show } from "solid-js";

export function CompletionList(props: {
  suggestions: string[],
  activeIdx: number,
}) {
  return (
    <div class="completion-list">
      <For each={props.suggestions}>
        {(item, i) => (
          <div
            classList={{
              "completion-item": true,
              active: i() === props.activeIdx
            }}
          >
            {item}
          </div>
        )}
      </For>
    </div>
  )
}
