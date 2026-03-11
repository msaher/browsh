import { createSignal, onMount } from 'solid-js';
import './Prompt.css';
import * as api from '../hooks/api'

interface Props {
  onSubmit: (src: string) => void;
}

export default function Prompt(props: Props) {
  const [src, setSrc] = createSignal('');
  const [focused, setFocused] = createSignal(false);
  const [completions, setCompletions] = createSignal<string[]>([])
  const [compIndex, setCompIndex] = createSignal(-1)
  const [baseWord, setBaseWord] = createSignal('')
  let textareaRef: HTMLTextAreaElement | undefined;

  onMount(() => {
    if (!textareaRef) return
    textareaRef.focus()
  })

  function resize() {
    if (!textareaRef) return;
    textareaRef.style.height = 'auto';
    textareaRef.style.height = textareaRef.scrollHeight + 'px';
  }

  function clearCompletions() {
    setCompletions([])
    setCompIndex(-1)
    setBaseWord('')
  }

  function applyCompletion(base: string, comps: string[], index: number) {
    const cursor = textareaRef?.selectionStart ?? base.length
    const before = base.slice(0, cursor)
    const after = base.slice(cursor)
    const lastSpace = before.lastIndexOf(' ')
    const prefix = before.slice(0, lastSpace + 1)
    const next = prefix + comps[index] + after
    setSrc(next)
    if (textareaRef) {
      textareaRef.value = next
      const pos = prefix.length + comps[index].length
      textareaRef.setSelectionRange(pos, pos)
    }
  }

  function onInput(e: InputEvent & { currentTarget: HTMLTextAreaElement }) {
    setSrc(e.currentTarget.value);
    clearCompletions()
    resize();
  }

  async function onKeyDown(e: KeyboardEvent & { currentTarget: HTMLTextAreaElement }) {
    if (e.key === 'Tab') {
      e.preventDefault()
      const current = src()
      const cursor = e.currentTarget.selectionStart ?? current.length

      let comps = completions()
      if (comps.length === 0) {
        comps = await api.complete(current, cursor)
        if (comps.length === 0) return
        setCompletions(comps)
        setBaseWord(current)
        setCompIndex(0)
        applyCompletion(current, comps, 0)
        return
      }

      const next = e.shiftKey
        ? (compIndex() - 1 + comps.length) % comps.length
        : (compIndex() + 1) % comps.length
      setCompIndex(next)
      applyCompletion(baseWord(), comps, next)
      return
    }

    if (completions().length > 0 && e.key !== 'Shift') {
      clearCompletions()
    }

    if (e.key === 'Enter' && e.shiftKey) return

    if (e.key === 'Enter') {
      e.preventDefault();
      const val = src().trim();
      if (!val) return;
      props.onSubmit(val);
      setSrc('');
      clearCompletions()
      if (textareaRef) {
        textareaRef.value = '';
        textareaRef.style.height = 'auto';
      }
    }
  }

  return (
    <div classList={{ 'prompt-wrap': true, focused: focused() }}>
      <span class="prompt-sigil">❯</span>
      <textarea
        ref={textareaRef}
        class="prompt-input"
        placeholder="type a command..."
        rows={1}
        value={src()}
        onInput={onInput}
        onKeyDown={onKeyDown}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        spellcheck={false}
        autocomplete="off"
      />
      <kbd class="prompt-hint">⏎ run</kbd>
    </div>
  );
}
