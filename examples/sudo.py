#!/usr/bin/env python3
import sys, json, subprocess, html

if len(sys.argv) < 2:
    print("usage: sudo.py cmd [args...]")
    sys.exit(1)

cmd = sys.argv[1:]

form = """
<style>
.sudo-box{font-family:sans-serif;padding:12px;border:1px solid #ccc;border-radius:6px;display:inline-block}
.sudo-box input{margin-top:6px}
</style>
<form class="sudo-box">
<div>sudo password required</div>
<input type="password" name="password" autofocus>
<button type="submit">authenticate</button>
</form>
"""
sys.stdout.write(form)
sys.stdout.flush()

line = sys.stdin.readline()
if not line:
    sys.exit(1)

data = json.loads(line)
pw = data.get("password","")

p = subprocess.Popen(
    ["sudo","-S"] + cmd,
    stdin=subprocess.PIPE
)

p.stdin.write((pw + "\n").encode())
p.stdin.close()
p.wait()

sys.exit(p.returncode)
