import { onMount, createSignal } from "solid-js";
import Prompt from './Prompt';
import * as jobs from '../hooks/jobs'

export interface BlockProps {
  src: string
}

export default function Block(props: BlockProps) {
  let outputRef: HTMLDivElement | undefined;

  function onmessage(event: MessageEvent) {
    if (!outputRef) return
    const text = JSON.parse(event.data);
    outputRef.innerHTML += text.data;
    outputRef.scrollTop = outputRef.scrollHeight;
  }

  onMount(async () => {
    const job = jobs.registerAndStartJob(props.src, onmessage)
  })

  return (
    <div ref={el => (outputRef = el)}></div>
  )
}
