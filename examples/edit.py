#!/usr/bin/env python3
import sys, json, html, os

if len(sys.argv) < 2:
    print("usage: edit <file>")
    sys.exit(1)

path = sys.argv[1]

if os.path.exists(path):
    with open(path) as f:
        content = f.read()
else:
    content = ""

escaped = html.escape(content)

print(f"""<style>
.ed-wrap {{ font-family: monospace; font-size: 13px; }}
.ed-path {{ color: #666; margin-bottom: 6px; }}
.ed-wrap textarea {{ width: 100%; min-height: 300px; background: #111; color: #ccc; border: 1px solid #333; border-radius: 4px; padding: 8px; font-family: monospace; font-size: 13px; resize: vertical; box-sizing: border-box; }}
.ed-wrap textarea:focus {{ outline: none; border-color: #555; }}
.ed-wrap button {{ margin-top: 6px; background: none; border: 1px solid #555; color: #aaa; border-radius: 3px; cursor: pointer; padding: 3px 10px; font-size: 12px; }}
.ed-wrap button:hover {{ border-color: #aaa; color: #eee; }}
.ed-saved {{ color: #4a4; font-size: 12px; margin-left: 8px; }}
</style>
<div class="ed-wrap">
<div class="ed-path">{html.escape(path)}</div>
<form>
<textarea spellcheck=false name="content">{escaped}</textarea>
<div><button type="submit">save</button></div>
</form>
</div>""", flush=True)

for line in sys.stdin:
    data = json.loads(line)
    with open(path, "w") as f:
        f.write(data["content"])
    print('<span class="ed-saved">saved</span>', flush=True)
