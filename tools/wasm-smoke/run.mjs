import fs from "node:fs";
import path from "node:path";
import vm from "node:vm";

const bundleDir = process.argv[2];
if (!bundleDir) {
  console.error("usage: node tools/wasm-smoke/run.mjs <wasm-bundle-dir> [--click x,y] [--drag x1,y1,x2,y2] [--a11y-click role,name] [--a11y-focus role,name] [--a11y-key role,name,key] [--a11y-text role,name,text] [--wheel x,y,deltaY] [--text text] [--compose text] [--paste text] [--key key] [--expect-text text] [--expect-latest-text-y-lt text,y] [--expect-fill color] [--expect-stroke color] [--expect-line-width width] [--expect-rounded] [--expect-clip] [--expect-alpha value] [--expect-checkmark] [--expect-selection] [--expect-arc] [--expect-svg-path] [--expect-a11y role,name] [--expect-a11y-focused role,name] [--expect-a11y-not-focused role,name] [--expect-a11y-checked role,name,value] [--expect-a11y-disabled role,name,value] [--dump-calls]");
  process.exit(2);
}
const options = parseOptions(process.argv.slice(3));

const root = path.resolve(bundleDir);
const wasmExecPath = path.join(root, "wasm_exec.js");
const wasmPath = path.join(root, "app.wasm");

const calls = [];
const listeners = new Map();

function makeStyle() {
  return {
    set width(value) {
      this._width = value;
    },
    get width() {
      return this._width;
    },
    set height(value) {
      this._height = value;
    },
    get height() {
      return this._height;
    },
  };
}

function makeContext() {
  return {
    set fillStyle(value) {
      calls.push(["set", "fillStyle", value]);
    },
    set strokeStyle(value) {
      calls.push(["set", "strokeStyle", value]);
    },
    set font(value) {
      calls.push(["set", "font", value]);
    },
    set lineWidth(value) {
      calls.push(["set", "lineWidth", value]);
    },
    set globalAlpha(value) {
      calls.push(["set", "globalAlpha", value]);
    },
    fillRect(...args) {
      calls.push(["fillRect", ...args]);
    },
    strokeRect(...args) {
      calls.push(["strokeRect", ...args]);
    },
    fillText(...args) {
      calls.push(["fillText", String(args[0]), ...args.slice(1)]);
    },
    measureText(text) {
      calls.push(["measureText", String(text)]);
      return {
        width: Array.from(String(text)).length * 8,
        actualBoundingBoxAscent: 12,
        actualBoundingBoxDescent: 4,
      };
    },
    setTransform(...args) {
      calls.push(["setTransform", ...args]);
    },
    beginPath() {
      calls.push(["beginPath"]);
    },
    moveTo(...args) {
      calls.push(["moveTo", ...args]);
    },
    lineTo(...args) {
      calls.push(["lineTo", ...args]);
    },
    arcTo(...args) {
      calls.push(["arcTo", ...args]);
    },
    arc(...args) {
      calls.push(["arc", ...args]);
    },
    bezierCurveTo(...args) {
      calls.push(["bezierCurveTo", ...args]);
    },
    quadraticCurveTo(...args) {
      calls.push(["quadraticCurveTo", ...args]);
    },
    closePath() {
      calls.push(["closePath"]);
    },
    fill() {
      calls.push(["fill"]);
    },
    stroke() {
      calls.push(["stroke"]);
    },
    save() {
      calls.push(["save"]);
    },
    rect(...args) {
      calls.push(["rect", ...args]);
    },
    clip() {
      calls.push(["clip"]);
    },
    restore() {
      calls.push(["restore"]);
    },
    translate(...args) {
      calls.push(["translate", ...args]);
    },
    scale(...args) {
      calls.push(["scale", ...args]);
    },
  };
}

const canvas = {
  style: makeStyle(),
  width: 0,
  height: 0,
  clientWidth: 800,
  clientHeight: 600,
  getContext(kind) {
    if (kind !== "2d") {
      throw new Error(`unexpected canvas context ${kind}`);
    }
    return makeContext();
  },
  getBoundingClientRect() {
    return { left: 0, top: 0 };
  },
  addEventListener(type, fn) {
    const list = listeners.get(type) || [];
    list.push(fn);
    listeners.set(type, list);
  },
};

const status = { textContent: "" };
const a11yRoot = makeElement("div");

globalThis.document = {
  getElementById(id) {
    if (id === "vugra-canvas") {
      return canvas;
    }
    if (id === "status") {
      return status;
    }
    if (id === "vugra-a11y") {
      return a11yRoot;
    }
    return null;
  },
  createElement(tagName) {
    return makeElement(tagName);
  },
};
globalThis.addEventListener = (type, fn) => {
  const key = `global:${type}`;
  const list = listeners.get(key) || [];
  list.push(fn);
  listeners.set(key, list);
};
globalThis.devicePixelRatio = 2;

vm.runInThisContext(fs.readFileSync(wasmExecPath, "utf8"), {
  filename: wasmExecPath,
});

const go = new Go();
const bytes = fs.readFileSync(wasmPath);
const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
go.run(instance);

await new Promise((resolve) => setTimeout(resolve, 50));

if (status.textContent !== "Running") {
  throw new Error(`expected status Running, got ${JSON.stringify(status.textContent)}`);
}
if (canvas.width <= 0 || canvas.height <= 0) {
  throw new Error(`canvas backing store was not initialized: ${canvas.width}x${canvas.height}`);
}
if (!calls.some((call) => call[0] === "setTransform" && call[1] === 2)) {
  throw new Error("DPR transform was not applied");
}
if (!calls.some((call) => call[0] === "fillText" || call[0] === "fillRect")) {
  throw new Error("no canvas draw calls were recorded");
}

for (const action of options.actions) {
  await runAction(action);
}

if (options.expectText && !calls.some((call) => call[0] === "fillText" && call[1] === options.expectText)) {
  throw new Error(`expected rendered text ${JSON.stringify(options.expectText)}`);
}
for (const expectation of options.expectLatestTextYLT) {
  const call = findLatestTextCall(expectation.text);
  if (!call) {
    throw new Error(`expected rendered text ${JSON.stringify(expectation.text)}`);
  }
  const y = Number(call[3]);
  if (!(y < expectation.y)) {
    throw new Error(`expected latest text ${JSON.stringify(expectation.text)} y ${y} to be < ${expectation.y}`);
  }
}
for (const color of options.expectFill) {
  if (!calls.some((call) => call[0] === "set" && call[1] === "fillStyle" && call[2] === color)) {
    throw new Error(`expected fillStyle ${color}`);
  }
}
for (const color of options.expectStroke) {
  if (!calls.some((call) => call[0] === "set" && call[1] === "strokeStyle" && call[2] === color)) {
    throw new Error(`expected strokeStyle ${color}`);
  }
}
for (const width of options.expectLineWidth) {
  if (!calls.some((call) => call[0] === "set" && call[1] === "lineWidth" && Number(call[2]) === width)) {
    throw new Error(`expected lineWidth ${width}`);
  }
}
if (options.expectRounded && !calls.some((call) => call[0] === "arcTo")) {
  throw new Error("expected rounded rectangle path");
}
if (options.expectClip && !calls.some((call) => call[0] === "clip")) {
  throw new Error("expected canvas clip");
}
for (const alpha of options.expectAlpha) {
  if (!calls.some((call) => call[0] === "set" && call[1] === "globalAlpha" && Number(call[2]) === alpha)) {
    throw new Error(`expected globalAlpha ${alpha}`);
  }
}
if (options.expectCheckmark && !hasCheckmarkPath()) {
  throw new Error("expected checkbox checkmark path");
}
if (options.expectSelection && !calls.some((call, index) => (
  call[0] === "fillRect" &&
  calls.slice(Math.max(0, index - 3), index).some((candidate) => candidate[0] === "set" && candidate[1] === "fillStyle" && candidate[2] === "#bfdbfe")
))) {
  throw new Error("expected text selection highlight");
}
if (options.expectArc && !calls.some((call) => call[0] === "arc")) {
  throw new Error("expected canvas arc");
}
if (options.expectSVGPath && !hasSVGPath()) {
  throw new Error("expected SVG path drawing");
}
for (const expectation of options.expectA11y) {
  if (!findA11yNode(expectation.role, expectation.name)) {
    throw new Error(`expected a11y node role=${expectation.role} name=${expectation.name}`);
  }
}
for (const expectation of options.expectA11yChecked) {
  const node = findA11yNode(expectation.role, expectation.name);
  if (!node) {
    throw new Error(`expected a11y node role=${expectation.role} name=${expectation.name}`);
  }
  if (node.attributes["aria-checked"] !== expectation.value) {
    throw new Error(`expected a11y checked ${expectation.value}, got ${node.attributes["aria-checked"]}`);
  }
}
for (const expectation of options.expectA11yDisabled) {
  const node = findA11yNode(expectation.role, expectation.name);
  if (!node) {
    throw new Error(`expected a11y node role=${expectation.role} name=${expectation.name}`);
  }
  if ((node.attributes["aria-disabled"] || "") !== expectation.value) {
    throw new Error(`expected a11y disabled ${expectation.value}, got ${node.attributes["aria-disabled"]}`);
  }
}
for (const expectation of options.expectA11yFocused) {
  const node = findA11yNode(expectation.role, expectation.name);
  if (!node) {
    throw new Error(`expected a11y node role=${expectation.role} name=${expectation.name}`);
  }
  if (node.attributes["data-vugra-focused"] !== "true" || node.attributes["aria-current"] !== "true") {
    throw new Error(`expected focused a11y node role=${expectation.role} name=${expectation.name}`);
  }
}
for (const expectation of options.expectA11yNotFocused) {
  const node = findA11yNode(expectation.role, expectation.name);
  if (!node) {
    throw new Error(`expected a11y node role=${expectation.role} name=${expectation.name}`);
  }
  if (node.attributes["data-vugra-focused"] === "true" || node.attributes["aria-current"] === "true") {
    throw new Error(`expected a11y node not to be focused role=${expectation.role} name=${expectation.name}`);
  }
}

console.log(JSON.stringify({
  status: status.textContent,
  backingStore: { width: canvas.width, height: canvas.height },
  cssSize: { width: canvas.style.width || `${canvas.clientWidth}px`, height: canvas.style.height || `${canvas.clientHeight}px` },
  drawCalls: calls.length,
  text: calls.filter((call) => call[0] === "fillText").map((call) => call[1]),
  listeners: Array.from(listeners.keys()).sort(),
  a11y: summarizeA11y(a11yRoot),
  calls: options.dumpCalls ? calls : undefined,
}));

process.exit(0);

async function runAction(action) {
  switch (action.type) {
    case "click":
      dispatchCanvasEvent("mousedown", action.x, action.y);
      dispatchCanvasEvent("mouseup", action.x, action.y);
      await delay();
      return;
    case "drag":
      dispatchCanvasEvent("mousedown", action.x1, action.y1);
      dispatchCanvasEvent("mousemove", action.x2, action.y2, { movementX: action.x2 - action.x1, movementY: action.y2 - action.y1 });
      dispatchCanvasEvent("mouseup", action.x2, action.y2);
      await delay();
      return;
    case "a11y-click": {
      const node = findA11yNode(action.role, action.name);
      if (!node) {
        throw new Error(`a11y click target missing role=${action.role} name=${action.name}`);
      }
      const beforeDrawCalls = calls.length;
      dispatchElementEvent(a11yRoot, "click", {
        target: node,
        preventDefault() {},
      });
      await delay();
      if (calls.length <= beforeDrawCalls) {
        throw new Error("a11y click did not trigger additional rendering");
      }
      return;
    }
    case "a11y-focus": {
      const node = findA11yNode(action.role, action.name);
      if (!node) {
        throw new Error(`a11y focus target missing role=${action.role} name=${action.name}`);
      }
      node.focus();
      await delay();
      return;
    }
    case "a11y-key": {
      const node = findA11yNode(action.role, action.name);
      if (!node) {
        throw new Error(`a11y key target missing role=${action.role} name=${action.name}`);
      }
      const event = keyEvent(action.key);
      dispatchElementEvent(a11yRoot, "keydown", {
        target: node,
        type: "keydown",
        ...event,
        preventDefault() {},
      });
      await delay();
      return;
    }
    case "a11y-text": {
      const node = findA11yNode(action.role, action.name);
      if (!node) {
        throw new Error(`a11y text target missing role=${action.role} name=${action.name}`);
      }
      const beforeTextCalls = calls.filter((call) => call[0] === "fillText").length;
      for (const char of Array.from(action.text)) {
        dispatchElementEvent(a11yRoot, "input", {
          target: node,
          type: "input",
          data: char,
          preventDefault() {},
        });
      }
      await delay();
      const afterTextCalls = calls.filter((call) => call[0] === "fillText").length;
      if (afterTextCalls <= beforeTextCalls) {
        throw new Error("a11y text input did not trigger additional text rendering");
      }
      return;
    }
    case "wheel": {
      const beforeDrawCalls = calls.length;
      dispatchCanvasEvent("wheel", action.x, action.y, { deltaY: action.deltaY });
      await delay();
      if (calls.length <= beforeDrawCalls) {
        throw new Error("wheel event did not trigger additional rendering");
      }
      return;
    }
    case "text": {
      const beforeTextCalls = calls.filter((call) => call[0] === "fillText").length;
      for (const char of Array.from(action.text)) {
        dispatchGlobalEvent("keydown", {
          key: char,
          ctrlKey: false,
          metaKey: false,
          altKey: false,
          shiftKey: false,
          preventDefault() {},
        });
      }
      await delay();
      const afterTextCalls = calls.filter((call) => call[0] === "fillText").length;
      if (afterTextCalls <= beforeTextCalls) {
        throw new Error("text input did not trigger additional text rendering");
      }
      return;
    }
    case "compose": {
      const beforeTextCalls = calls.filter((call) => call[0] === "fillText").length;
      dispatchGlobalEvent("compositionstart", {
        data: "",
        preventDefault() {},
      });
      dispatchGlobalEvent("keydown", {
        key: "Process",
        isComposing: true,
        ctrlKey: false,
        metaKey: false,
        altKey: false,
        shiftKey: false,
        preventDefault() {},
      });
      dispatchGlobalEvent("compositionend", {
        data: action.text,
        preventDefault() {},
      });
      await delay();
      const afterTextCalls = calls.filter((call) => call[0] === "fillText").length;
      if (afterTextCalls <= beforeTextCalls) {
        throw new Error("composition input did not trigger additional text rendering");
      }
      return;
    }
    case "paste": {
      const beforeTextCalls = calls.filter((call) => call[0] === "fillText").length;
      dispatchGlobalEvent("paste", {
        clipboardData: {
          getData(kind) {
            if (kind !== "text/plain") {
              return "";
            }
            return action.text;
          },
        },
        preventDefault() {},
      });
      await delay();
      const afterTextCalls = calls.filter((call) => call[0] === "fillText").length;
      if (afterTextCalls <= beforeTextCalls) {
        throw new Error("paste input did not trigger additional text rendering");
      }
      return;
    }
    case "key": {
      const event = keyEvent(action.key);
      dispatchGlobalEvent("keydown", {
        ...event,
        preventDefault() {},
      });
      await delay();
      return;
    }
    default:
      throw new Error(`unknown action ${action.type}`);
  }
}

function delay() {
  return new Promise((resolve) => setTimeout(resolve, 50));
}

function parseOptions(args) {
  const out = { actions: [], expectFill: [], expectStroke: [], expectLineWidth: [], expectLatestTextYLT: [], expectAlpha: [], expectA11y: [], expectA11yChecked: [], expectA11yDisabled: [], expectA11yFocused: [], expectA11yNotFocused: [], expectRounded: false, expectClip: false, expectCheckmark: false, expectSelection: false, expectArc: false, expectSVGPath: false, dumpCalls: false };
  for (let index = 0; index < args.length; index++) {
    const arg = args[index];
    switch (arg) {
      case "--click": {
        const value = args[++index];
        if (!value) {
          throw new Error("--click requires x,y");
        }
        const [xRaw, yRaw] = value.split(",");
        const x = Number(xRaw);
        const y = Number(yRaw);
        if (!Number.isFinite(x) || !Number.isFinite(y)) {
          throw new Error(`invalid --click value ${value}`);
        }
        out.actions.push({ type: "click", x, y });
        break;
      }
      case "--drag": {
        const value = args[++index];
        if (!value) {
          throw new Error("--drag requires x1,y1,x2,y2");
        }
        const [x1Raw, y1Raw, x2Raw, y2Raw] = value.split(",");
        const x1 = Number(x1Raw);
        const y1 = Number(y1Raw);
        const x2 = Number(x2Raw);
        const y2 = Number(y2Raw);
        if (!Number.isFinite(x1) || !Number.isFinite(y1) || !Number.isFinite(x2) || !Number.isFinite(y2)) {
          throw new Error(`invalid --drag value ${value}`);
        }
        out.actions.push({ type: "drag", x1, y1, x2, y2 });
        break;
      }
      case "--a11y-click": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--a11y-click requires role,name");
        }
        const comma = value.indexOf(",");
        if (comma <= 0) {
          throw new Error(`invalid --a11y-click value ${value}`);
        }
        out.actions.push({ type: "a11y-click", role: value.slice(0, comma), name: value.slice(comma + 1) });
        break;
      }
      case "--a11y-focus": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--a11y-focus requires role,name");
        }
        const comma = value.indexOf(",");
        if (comma <= 0) {
          throw new Error(`invalid --a11y-focus value ${value}`);
        }
        out.actions.push({ type: "a11y-focus", role: value.slice(0, comma), name: value.slice(comma + 1) });
        break;
      }
      case "--a11y-key": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--a11y-key requires role,name,key");
        }
        const parsed = parseRoleNameText(value, "--a11y-key");
        out.actions.push({ type: "a11y-key", role: parsed.role, name: parsed.name, key: parsed.text });
        break;
      }
      case "--a11y-text": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--a11y-text requires role,name,text");
        }
        const parsed = parseRoleNameText(value, "--a11y-text");
        out.actions.push({ type: "a11y-text", role: parsed.role, name: parsed.name, text: parsed.text });
        break;
      }
      case "--wheel": {
        const value = args[++index];
        if (!value) {
          throw new Error("--wheel requires x,y,deltaY");
        }
        const [xRaw, yRaw, deltaYRaw] = value.split(",");
        const x = Number(xRaw);
        const y = Number(yRaw);
        const deltaY = Number(deltaYRaw);
        if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(deltaY)) {
          throw new Error(`invalid --wheel value ${value}`);
        }
        out.actions.push({ type: "wheel", x, y, deltaY });
        break;
      }
      case "--expect-text": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-text requires text");
        }
        out.expectText = value;
        break;
      }
      case "--expect-latest-text-y-lt": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-latest-text-y-lt requires text,y");
        }
        const comma = value.lastIndexOf(",");
        if (comma <= 0) {
          throw new Error(`invalid --expect-latest-text-y-lt value ${value}`);
        }
        const text = value.slice(0, comma);
        const y = Number(value.slice(comma + 1));
        if (!text || !Number.isFinite(y)) {
          throw new Error(`invalid --expect-latest-text-y-lt value ${value}`);
        }
        out.expectLatestTextYLT.push({ text, y });
        break;
      }
      case "--text": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--text requires text");
        }
        out.actions.push({ type: "text", text: value });
        break;
      }
      case "--compose": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--compose requires text");
        }
        out.actions.push({ type: "compose", text: value });
        break;
      }
      case "--paste": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--paste requires text");
        }
        out.actions.push({ type: "paste", text: value });
        break;
      }
      case "--key": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--key requires key");
        }
        out.actions.push({ type: "key", key: value });
        break;
      }
      case "--expect-fill": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-fill requires color");
        }
        out.expectFill.push(value);
        break;
      }
      case "--expect-stroke": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-stroke requires color");
        }
        out.expectStroke.push(value);
        break;
      }
      case "--expect-line-width": {
        const value = Number(args[++index]);
        if (!Number.isFinite(value)) {
          throw new Error("--expect-line-width requires a number");
        }
        out.expectLineWidth.push(value);
        break;
      }
      case "--expect-rounded":
        out.expectRounded = true;
        break;
      case "--expect-clip":
        out.expectClip = true;
        break;
      case "--expect-alpha": {
        const value = Number(args[++index]);
        if (!Number.isFinite(value)) {
          throw new Error("--expect-alpha requires a number");
        }
        out.expectAlpha.push(value);
        break;
      }
      case "--expect-checkmark":
        out.expectCheckmark = true;
        break;
      case "--expect-selection":
        out.expectSelection = true;
        break;
      case "--expect-arc":
        out.expectArc = true;
        break;
      case "--expect-svg-path":
        out.expectSVGPath = true;
        break;
      case "--expect-a11y": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-a11y requires role,name");
        }
        const comma = value.indexOf(",");
        if (comma <= 0) {
          throw new Error(`invalid --expect-a11y value ${value}`);
        }
        out.expectA11y.push({ role: value.slice(0, comma), name: value.slice(comma + 1) });
        break;
      }
      case "--expect-a11y-focused": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-a11y-focused requires role,name");
        }
        const comma = value.indexOf(",");
        if (comma <= 0) {
          throw new Error(`invalid --expect-a11y-focused value ${value}`);
        }
        out.expectA11yFocused.push({ role: value.slice(0, comma), name: value.slice(comma + 1) });
        break;
      }
      case "--expect-a11y-not-focused": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-a11y-not-focused requires role,name");
        }
        const comma = value.indexOf(",");
        if (comma <= 0) {
          throw new Error(`invalid --expect-a11y-not-focused value ${value}`);
        }
        out.expectA11yNotFocused.push({ role: value.slice(0, comma), name: value.slice(comma + 1) });
        break;
      }
      case "--expect-a11y-checked": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-a11y-checked requires role,name,value");
        }
        const parts = value.split(",");
        if (parts.length < 3) {
          throw new Error(`invalid --expect-a11y-checked value ${value}`);
        }
        out.expectA11yChecked.push({ role: parts[0], name: parts.slice(1, -1).join(","), value: parts[parts.length - 1] });
        break;
      }
      case "--expect-a11y-disabled": {
        const value = args[++index];
        if (value === undefined) {
          throw new Error("--expect-a11y-disabled requires role,name,value");
        }
        const parts = value.split(",");
        if (parts.length < 3) {
          throw new Error(`invalid --expect-a11y-disabled value ${value}`);
        }
        out.expectA11yDisabled.push({ role: parts[0], name: parts.slice(1, -1).join(","), value: parts[parts.length - 1] });
        break;
      }
      case "--dump-calls":
        out.dumpCalls = true;
        break;
      default:
        throw new Error(`unknown option ${arg}`);
    }
  }
  return out;
}

function makeElement(tagName) {
  return {
    tagName,
    attributes: {},
    children: [],
    listeners: {},
    parentElement: null,
    textContent: "",
    set innerHTML(value) {
      this._innerHTML = String(value);
      this.children = [];
    },
    get innerHTML() {
      return this._innerHTML || "";
    },
    setAttribute(name, value) {
      this.attributes[name] = String(value);
    },
    getAttribute(name) {
      return this.attributes[name] ?? null;
    },
    appendChild(child) {
      child.parentElement = this;
      this.children.push(child);
      return child;
    },
    addEventListener(type, fn) {
      const list = this.listeners[type] || [];
      list.push(fn);
      this.listeners[type] = list;
    },
    focus() {
      dispatchElementEvent(a11yRoot, "focusin", {
        target: this,
        type: "focusin",
        preventDefault() {},
        stopPropagation() {},
      });
    },
  };
}

function dispatchElementEvent(element, type, event) {
  const handlers = element.listeners[type] || [];
  if (handlers.length === 0) {
    throw new Error(`no element listener for ${type}`);
  }
  for (const handler of handlers) {
    handler(event);
  }
}

function findA11yNode(role, name) {
	const stack = [...a11yRoot.children];
  while (stack.length > 0) {
    const node = stack.shift();
    if (node.attributes.role === role && (node.attributes["aria-label"] || "") === name) {
      return node;
    }
    stack.push(...node.children);
  }
	return null;
}

function parseRoleNameText(value, flag) {
  const first = value.indexOf(",");
  const last = value.lastIndexOf(",");
  if (first <= 0 || last <= first) {
    throw new Error(`invalid ${flag} value ${value}`);
  }
  return {
    role: value.slice(0, first),
    name: value.slice(first + 1, last),
    text: value.slice(last + 1),
  };
}

function summarizeA11y(root) {
  const out = [];
  const stack = [...root.children];
  while (stack.length > 0) {
    const node = stack.shift();
    out.push({
      role: node.attributes.role || "",
      name: node.attributes["aria-label"] || "",
      checked: node.attributes["aria-checked"] || "",
      disabled: node.attributes["aria-disabled"] || "",
      focused: node.attributes["data-vugra-focused"] || "",
    });
    stack.push(...node.children);
  }
  return out;
}

function hasCheckmarkPath() {
  for (let index = 0; index < calls.length - 3; index++) {
    if (
      calls[index][0] === "moveTo" &&
      calls[index + 1][0] === "lineTo" &&
      calls[index + 2][0] === "lineTo" &&
      calls[index + 3][0] === "stroke"
    ) {
      return true;
    }
  }
  return false;
}

function hasSVGPath() {
  for (let index = 0; index < calls.length - 2; index++) {
    if (
      calls[index][0] === "beginPath" &&
      calls[index + 1][0] === "moveTo" &&
      calls.slice(index + 2).some((call) => call[0] === "lineTo" || call[0] === "bezierCurveTo" || call[0] === "quadraticCurveTo")
    ) {
      return true;
    }
  }
  return false;
}

function findLatestTextCall(text) {
  for (let index = calls.length - 1; index >= 0; index--) {
    const call = calls[index];
    if (call[0] === "fillText" && call[1] === text) {
      return call;
    }
  }
  return null;
}

function dispatchGlobalEvent(type, event) {
  const handlers = listeners.get(`global:${type}`);
  if (!handlers || handlers.length === 0) {
    throw new Error(`no global listener for ${type}`);
  }
  for (const handler of handlers) {
    handler(event);
  }
}

function keyEvent(key) {
  if (key === "Mod+A") {
    return {
      key: "a",
      ctrlKey: true,
      metaKey: false,
      altKey: false,
      shiftKey: false,
    };
  }
  if (key === "Shift+Tab") {
    return {
      key: "Tab",
      ctrlKey: false,
      metaKey: false,
      altKey: false,
      shiftKey: true,
    };
  }
  if (key === "Home" || key === "End") {
    return {
      key,
      ctrlKey: false,
      metaKey: false,
      altKey: false,
      shiftKey: false,
    };
  }
  return {
    key,
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    shiftKey: false,
  };
}

function dispatchCanvasEvent(type, x, y, extra = {}) {
  const handlers = listeners.get(type);
  if (!handlers || handlers.length === 0) {
    throw new Error(`no listener for ${type}`);
  }
  const event = {
    type,
    clientX: x,
    clientY: y,
    movementX: 0,
    movementY: 0,
    shiftKey: false,
    ctrlKey: false,
    metaKey: false,
    altKey: false,
    deltaY: 0,
    preventDefault() {},
    ...extra,
  };
  for (const handler of handlers) {
    handler(event);
  }
}
