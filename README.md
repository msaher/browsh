# browsh

https://github.com/user-attachments/assets/142746cb-9ea3-4d13-a5f3-35bc15ec9700

A web-based shell with first-class lua support.

Why? Terminals are nice, but they're stuck with plain text. They can't mix fonts or display buttons or forms. TUI style interfaces are non-trivial to make and almost certainly require a dependency. It'd be great if programs were able to output some form of markup to describe how to render their output... If only there was a standardized markup language that's widely used...

browsh is a shell that runs in the browser. Programs output arbitrary HTML which gets rendered in the UI. They can even output forms which are then submitted to standard input as json.

# Lua support

Shells are great for interactive use or one-liners, but they fall short when you need a real programming language (there's a reason all major distros ship python).

Instead of inventing yet another shell language, browsh embeds lua allowing you to escape into a real language when needed.

```lua
cat myfile.go | :lua {
    longest = ""
    for line in sh.stdin do
        for w in line:gmatch("%w+") do
            if #w > #longest then longest = w end
        end
    end
    sh.print("longest identifier:", longest)
}
```

Lua blocks are treated as normal commands, meaning they can be used in pipes.

```lua
cat server.log
| :lua {
  for line in sh.stdin do
    local h, m, s, msg = line:match("(%d%d):(%d%d):(%d%d) .-%[ERROR%] (.*)")
    if h then
      local secs = h*3600 + m*60 + s
      sh.print(secs .. " " .. msg)
    end
  end
}
| sort -n
```

# Usage

Run `make` to build `browsh`.
