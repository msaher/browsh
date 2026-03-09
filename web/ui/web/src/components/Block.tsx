import { onMount, createSignal } from "solid-js";
import Prompt from './Prompt';
import * as jobs from '../hooks/jobs'

export interface BlockProps {
  src: string
}

export default function Block(props: BlockProps) {

  onMount(async () => {
    const ws = jobs.registerAndStartJob(props.src)
  })

  return (
    <div>{props.src}</div>
  )
}
