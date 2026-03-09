import { createSignal } from 'solid-js';
import './Prompt.css';

interface Props {
  onSubmit: (src: string) => void;
}

export default function Prompt(props: Props) {
  const [src, setSrc] = createSignal('');
  const [focused, setFocused] = createSignal(false);
  let textareaRef: HTMLTextAreaElement | undefined;

  function resize() {
    if (!textareaRef) return;
    textareaRef.style.height = 'auto';
    textareaRef.style.height = textareaRef.scrollHeight + 'px';
  }

  function onInput(e: InputEvent & { currentTarget: HTMLTextAreaElement }) {
    setSrc(e.currentTarget.value);
    resize();
  }

  function onKeyDown(e: KeyboardEvent & { currentTarget: HTMLTextAreaElement }) {
    if (e.key === 'Enter' && e.shiftKey) {
      return;
    }
    if (e.key === 'Enter') {
      e.preventDefault();
      const val = src().trim();
      if (!val) return;
      props.onSubmit(val);
      setSrc('');
      if (textareaRef) {
        textareaRef.value = '';
        textareaRef.style.height = 'auto';
      }
    }
  }

  return (
    <div class={`prompt-wrap ${focused() ? 'focused' : ''}`}>
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
