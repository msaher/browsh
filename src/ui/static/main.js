function stringToArgv(str) {
    const re = /[^\s"]+|"([^"]*)"/g;
    const args = [];
    let match;
    while ((match = re.exec(str)) !== null) {
        args.push(match[1] !== undefined ? match[1] : match[0]);
    }
    return args;
}

async function requestRun(argv) {
  return await fetch("/run", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ argv })
  })
}

class Repl {
  constructor(node) {
    this.node = node
  }

  getInputNode() {
    const input = this.node.querySelector(".repl-input");
    return input
  }

  listenForInput() {
    const input = this.getInputNode()
    input.addEventListener("keydown", async (e) => {
      if (e.key === "Enter") {
        input.disabled = true;
        const cmd = input.value;
        const argv = stringToArgv(cmd)
        const response = await requestRun(argv)
        const output = await response.text()
        // set output
        const replOutput = this.node.querySelector(".repl-output")
        console.log(replOutput)
        // replOutput.innerHTML = output
      }

    });
  }

}

function makeReplNode() {
  const replTemplate = document.getElementById("repl-template")
  const fragment = replTemplate.content.cloneNode(true)
  const root = fragment.firstElementChild
  return new Repl(root)
}

function main() {
  // insert repl at body
  const repl = makeReplNode();
  document.body.appendChild(repl.node);
  repl.listenForInput()
  repl.getInputNode().focus()

}

main()
