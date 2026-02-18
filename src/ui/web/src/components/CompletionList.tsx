import './CompletionList.css'
import { For, Show } from "solid-js";

export function CompletionList(props: {
  suggestions: string[],
}) {
  return (
    <div class="completion-list">
      <For each={props.suggestions}>
        {(item, i) => (
          <div
            classList={{
              "completion-item": true,
              active: i() === props.activeIndex
            }}
          >
            {item}
          </div>
        )}
      </For>
    </div>
  )
}
