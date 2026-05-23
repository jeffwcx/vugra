import fs from "node:fs";
import http from "node:http";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { spawn } from "node:child_process";

const bundleDir = process.argv[2];
if (!bundleDir) {
  console.error("usage: node tools/wasm-browser-check/run.mjs <wasm-bundle-dir> [--click x,y] [--hover x,y] [--drag x1,y1,x2,y2] [--dblclick x,y] [--contextmenu x,y] [--a11y-click role,name] [--a11y-focus role,name] [--wheel x,y,deltaY] [--text text] [--compose text] [--paste text] [--key key] [--a11y-text role,name,text] [--a11y-key role,name,key] [--expect-title title] [--expect-canvas width,height] [--expect-text text] [--expect-a11y role,name] [--expect-a11y-focused role,name] [--expect-a11y-not-focused role,name] [--expect-a11y-checked role,name,value] [--expect-a11y-disabled role,name,value] [--expect-a11y-y-lt role,name,y] [--expect-pixel x,y,#rrggbb] [--screenshot out.png] [--chrome path]");
  process.exit(2);
}

const options = parseOptions(process.argv.slice(3));
const root = path.resolve(bundleDir);
for (const name of ["index.html", "app.wasm", "wasm_exec.js"]) {
  if (!fs.existsSync(path.join(root, name))) {
    throw new Error(`missing wasm bundle asset ${name} in ${root}`);
  }
}

const httpPort = await freePort();
const debugPort = await freePort();
const server = await startStaticServer(root, httpPort);
const userDataDir = fs.mkdtempSync(path.join(os.tmpdir(), "vugra-chrome-"));
const chrome = spawn(chromePath(options.chrome), [
  "--headless=new",
  "--disable-gpu",
  "--no-first-run",
  "--no-default-browser-check",
  `--remote-debugging-port=${debugPort}`,
  `--user-data-dir=${userDataDir}`,
  `http://127.0.0.1:${httpPort}/index.html`,
], { stdio: ["ignore", "pipe", "pipe"] });

try {
  const tab = await waitForTab(debugPort);
  const ws = await connectWebSocket(tab.webSocketDebuggerUrl);
  const cdp = createCDP(ws);
  await cdp.send("Runtime.enable");
  await cdp.send("Page.enable");
  await cdp.send("Page.bringToFront");
  await waitForRunning(cdp);
  await installHelpers(cdp);
  for (const action of options.actions) {
    await runAction(cdp, action);
  }
  await delay(100);
  const result = await pageState(cdp, options.expectPixel);
  if (!result.ok) {
    throw new Error(`browser check failed: ${JSON.stringify(result)}`);
  }
  if (options.expectText && !result.text.includes(options.expectText)) {
    throw new Error(`expected rendered text ${JSON.stringify(options.expectText)} in ${JSON.stringify(result.text)}`);
  }
  if (options.expectTitle && result.title !== options.expectTitle) {
    throw new Error(`expected title ${JSON.stringify(options.expectTitle)}, got ${JSON.stringify(result.title)}`);
  }
  if (options.expectCanvas && (result.canvas.width !== options.expectCanvas.width || result.canvas.height !== options.expectCanvas.height)) {
    throw new Error(`expected canvas ${JSON.stringify(options.expectCanvas)}, got ${JSON.stringify(result.canvas)}`);
  }
  for (const expectation of options.expectA11y) {
    if (!result.a11y.some((node) => node.role === expectation.role && node.name === expectation.name)) {
      throw new Error(`expected a11y node ${JSON.stringify(expectation)} in ${JSON.stringify(result.a11y)}`);
    }
  }
  for (const expectation of options.expectA11yFocused) {
    if (!result.a11y.some((node) => node.role === expectation.role && node.name === expectation.name && node.focused === "true")) {
      throw new Error(`expected focused a11y node ${JSON.stringify(expectation)} in ${JSON.stringify(result.a11y)}`);
    }
  }
  for (const expectation of options.expectA11yNotFocused) {
    if (result.a11y.some((node) => node.role === expectation.role && node.name === expectation.name && node.focused === "true")) {
      throw new Error(`expected a11y node not to be focused ${JSON.stringify(expectation)} in ${JSON.stringify(result.a11y)}`);
    }
  }
  for (const expectation of options.expectA11yChecked) {
    if (!result.a11y.some((node) => node.role === expectation.role && node.name === expectation.name && node.checked === expectation.value)) {
      throw new Error(`expected checked a11y node ${JSON.stringify(expectation)} in ${JSON.stringify(result.a11y)}`);
    }
  }
  for (const expectation of options.expectA11yDisabled) {
    if (!result.a11y.some((node) => node.role === expectation.role && node.name === expectation.name && node.disabled === expectation.value)) {
      throw new Error(`expected disabled a11y node ${JSON.stringify(expectation)} in ${JSON.stringify(result.a11y)}`);
    }
  }
  for (const expectation of options.expectA11yYLT) {
    const node = result.a11y.find((candidate) => candidate.role === expectation.role && candidate.name === expectation.name);
    if (!node) {
      throw new Error(`expected a11y y target ${JSON.stringify(expectation)} in ${JSON.stringify(result.a11y)}`);
    }
    if (!(node.y < expectation.y)) {
      throw new Error(`expected a11y node ${JSON.stringify(expectation)} y ${node.y} to be < ${expectation.y}`);
    }
  }
  for (const expectation of options.expectPixel) {
    const actual = result.pixels[`${expectation.x},${expectation.y}`];
    if (!actual) {
      throw new Error(`missing sampled pixel ${expectation.x},${expectation.y} in ${JSON.stringify(result.pixels)}`);
    }
    if (!sameColor(actual, expectation.color)) {
      throw new Error(`expected pixel ${expectation.x},${expectation.y} ${expectation.color.hex}, got ${actual.hex}`);
    }
  }
  if (options.screenshot) {
    const screenshot = await cdp.send("Page.captureScreenshot", { format: "png" });
    fs.writeFileSync(path.resolve(options.screenshot), Buffer.from(screenshot.data, "base64"));
  }
  console.log(JSON.stringify(result));
} finally {
  await stopProcess(chrome);
  server.close();
  rmRetry(userDataDir);
}

function parseOptions(args) {
  const out = { actions: [], expectA11y: [], expectA11yFocused: [], expectA11yNotFocused: [], expectA11yChecked: [], expectA11yDisabled: [], expectA11yYLT: [], expectPixel: [] };
  for (let index = 0; index < args.length; index++) {
    const arg = args[index];
    switch (arg) {
      case "--click":
        out.actions.push({ type: "click", ...parsePoint(args[++index], "--click") });
        break;
      case "--hover":
        out.actions.push({ type: "hover", ...parsePoint(args[++index], "--hover") });
        break;
      case "--drag":
        out.actions.push({ type: "drag", ...parseDrag(args[++index], "--drag") });
        break;
      case "--dblclick":
        out.actions.push({ type: "dblclick", ...parsePoint(args[++index], "--dblclick") });
        break;
      case "--contextmenu":
        out.actions.push({ type: "contextmenu", ...parsePoint(args[++index], "--contextmenu") });
        break;
      case "--a11y-click":
        out.actions.push({ type: "a11y-click", ...parseRoleName(args[++index], "--a11y-click") });
        break;
      case "--a11y-focus":
        out.actions.push({ type: "a11y-focus", ...parseRoleName(args[++index], "--a11y-focus") });
        break;
      case "--wheel":
        out.actions.push({ type: "wheel", ...parseWheel(args[++index], "--wheel") });
        break;
      case "--text": {
        const text = args[++index];
        if (text === undefined) throw new Error("--text requires text");
        out.actions.push({ type: "text", text });
        break;
      }
      case "--compose": {
        const text = args[++index];
        if (text === undefined) throw new Error("--compose requires text");
        out.actions.push({ type: "compose", text });
        break;
      }
      case "--paste": {
        const text = args[++index];
        if (text === undefined) throw new Error("--paste requires text");
        out.actions.push({ type: "paste", text });
        break;
      }
      case "--key": {
        const key = args[++index];
        if (key === undefined) throw new Error("--key requires key");
        out.actions.push({ type: "key", key });
        break;
      }
      case "--a11y-text":
        out.actions.push({ type: "a11y-text", ...parseRoleNameText(args[++index], "--a11y-text") });
        break;
      case "--a11y-key":
        out.actions.push({ type: "a11y-key", ...parseRoleNameText(args[++index], "--a11y-key") });
        break;
      case "--expect-text":
        out.expectText = args[++index];
        if (out.expectText === undefined) throw new Error("--expect-text requires text");
        break;
      case "--expect-title":
        out.expectTitle = args[++index];
        if (out.expectTitle === undefined) throw new Error("--expect-title requires title");
        break;
      case "--expect-canvas":
        out.expectCanvas = parseCanvasSize(args[++index], "--expect-canvas");
        break;
      case "--expect-a11y":
        out.expectA11y.push(parseRoleName(args[++index], "--expect-a11y"));
        break;
      case "--expect-a11y-focused":
        out.expectA11yFocused.push(parseRoleName(args[++index], "--expect-a11y-focused"));
        break;
      case "--expect-a11y-not-focused":
        out.expectA11yNotFocused.push(parseRoleName(args[++index], "--expect-a11y-not-focused"));
        break;
      case "--expect-a11y-checked":
        out.expectA11yChecked.push(parseRoleNameValue(args[++index], "--expect-a11y-checked"));
        break;
      case "--expect-a11y-disabled":
        out.expectA11yDisabled.push(parseRoleNameValue(args[++index], "--expect-a11y-disabled"));
        break;
      case "--expect-a11y-y-lt":
        out.expectA11yYLT.push(parseRoleNameNumber(args[++index], "--expect-a11y-y-lt"));
        break;
      case "--expect-pixel":
        out.expectPixel.push(parsePixel(args[++index], "--expect-pixel"));
        break;
      case "--screenshot":
        out.screenshot = args[++index];
        if (!out.screenshot) throw new Error("--screenshot requires a path");
        break;
      case "--chrome":
        out.chrome = args[++index];
        if (!out.chrome) throw new Error("--chrome requires a path");
        break;
      default:
        throw new Error(`unknown option ${arg}`);
    }
  }
  return out;
}

function parsePoint(value, flag) {
  if (!value) throw new Error(`${flag} requires x,y`);
  const [xRaw, yRaw] = value.split(",");
  const x = Number(xRaw);
  const y = Number(yRaw);
  if (!Number.isFinite(x) || !Number.isFinite(y)) {
    throw new Error(`invalid ${flag} value ${value}`);
  }
  return { x, y };
}

function parseWheel(value, flag) {
  if (!value) throw new Error(`${flag} requires x,y,deltaY`);
  const [xRaw, yRaw, deltaYRaw] = value.split(",");
  const x = Number(xRaw);
  const y = Number(yRaw);
  const deltaY = Number(deltaYRaw);
  if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(deltaY)) {
    throw new Error(`invalid ${flag} value ${value}`);
  }
  return { x, y, deltaY };
}

function parseDrag(value, flag) {
  if (!value) throw new Error(`${flag} requires x1,y1,x2,y2`);
  const [x1Raw, y1Raw, x2Raw, y2Raw] = value.split(",");
  const x1 = Number(x1Raw);
  const y1 = Number(y1Raw);
  const x2 = Number(x2Raw);
  const y2 = Number(y2Raw);
  if (!Number.isFinite(x1) || !Number.isFinite(y1) || !Number.isFinite(x2) || !Number.isFinite(y2)) {
    throw new Error(`invalid ${flag} value ${value}`);
  }
  return { x1, y1, x2, y2 };
}

function parseCanvasSize(value, flag) {
  if (!value) throw new Error(`${flag} requires width,height`);
  const [widthRaw, heightRaw] = value.split(",");
  const width = Number(widthRaw);
  const height = Number(heightRaw);
  if (!Number.isFinite(width) || !Number.isFinite(height)) {
    throw new Error(`invalid ${flag} value ${value}`);
  }
  return { width, height };
}

function parsePixel(value, flag) {
  if (!value) throw new Error(`${flag} requires x,y,#rrggbb`);
  const [xRaw, yRaw, colorRaw] = value.split(",");
  const x = Number(xRaw);
  const y = Number(yRaw);
  const color = parseHexColor(colorRaw);
  if (!Number.isFinite(x) || !Number.isFinite(y) || !color) {
    throw new Error(`invalid ${flag} value ${value}`);
  }
  return { x, y, color };
}

function parseHexColor(value) {
  if (!/^#[0-9a-fA-F]{6}$/.test(value || "")) {
    return null;
  }
  return {
    hex: value.toLowerCase(),
    r: Number.parseInt(value.slice(1, 3), 16),
    g: Number.parseInt(value.slice(3, 5), 16),
    b: Number.parseInt(value.slice(5, 7), 16),
  };
}

function sameColor(actual, expected) {
  return actual.r === expected.r && actual.g === expected.g && actual.b === expected.b && actual.a === 255;
}

function parseRoleName(value, flag) {
  if (value === undefined) throw new Error(`${flag} requires role,name`);
  const comma = value.indexOf(",");
  if (comma <= 0) throw new Error(`invalid ${flag} value ${value}`);
  return { role: value.slice(0, comma), name: value.slice(comma + 1) };
}

function parseRoleNameValue(value, flag) {
  if (value === undefined) throw new Error(`${flag} requires role,name,value`);
  const first = value.indexOf(",");
  const last = value.lastIndexOf(",");
  if (first <= 0 || last < first) throw new Error(`invalid ${flag} value ${value}`);
  return {
    role: value.slice(0, first),
    name: value.slice(first + 1, last),
    value: value.slice(last + 1),
  };
}

function parseRoleNameNumber(value, flag) {
  if (value === undefined) throw new Error(`${flag} requires role,name,number`);
  const first = value.indexOf(",");
  const last = value.lastIndexOf(",");
  if (first <= 0 || last <= first) throw new Error(`invalid ${flag} value ${value}`);
  const number = Number(value.slice(last + 1));
  if (!Number.isFinite(number)) throw new Error(`invalid ${flag} number ${value}`);
  return {
    role: value.slice(0, first),
    name: value.slice(first + 1, last),
    y: number,
  };
}

function parseRoleNameText(value, flag) {
  if (value === undefined) throw new Error(`${flag} requires role,name,text`);
  const first = value.indexOf(",");
  const last = value.lastIndexOf(",");
  if (first <= 0 || last <= first) throw new Error(`invalid ${flag} value ${value}`);
  return {
    role: value.slice(0, first),
    name: value.slice(first + 1, last),
    text: value.slice(last + 1),
  };
}

function chromePath(explicit) {
  const candidates = [
    explicit,
    process.env.CHROME,
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
  ].filter(Boolean);
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) return candidate;
  }
  throw new Error("Chrome/Chromium executable not found; pass --chrome /path/to/chrome");
}

function startStaticServer(rootDir, port) {
  const server = http.createServer((req, res) => {
    const url = new URL(req.url || "/", `http://127.0.0.1:${port}`);
    const decoded = decodeURIComponent(url.pathname);
    const clean = path.normalize(decoded).replace(/^(\.\.[/\\])+/, "");
    let filePath = path.join(rootDir, clean);
    if (decoded === "/" || decoded.endsWith("/")) {
      filePath = path.join(rootDir, clean, "index.html");
    }
    if (!filePath.startsWith(rootDir)) {
      res.writeHead(403);
      res.end("forbidden");
      return;
    }
    fs.readFile(filePath, (err, data) => {
      if (err) {
        res.writeHead(404);
        res.end("not found");
        return;
      }
      res.writeHead(200, { "Content-Type": contentType(filePath) });
      res.end(data);
    });
  });
  return new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, "127.0.0.1", () => resolve(server));
  });
}

function contentType(filePath) {
  switch (path.extname(filePath)) {
    case ".html": return "text/html; charset=utf-8";
    case ".js": return "text/javascript; charset=utf-8";
    case ".wasm": return "application/wasm";
    case ".css": return "text/css; charset=utf-8";
    case ".png": return "image/png";
    default: return "application/octet-stream";
  }
}

function freePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const port = server.address().port;
      server.close(() => resolve(port));
    });
  });
}

async function waitForTab(port) {
  const deadline = Date.now() + 10000;
  while (Date.now() < deadline) {
    try {
      const tabs = await fetchJSON(`http://127.0.0.1:${port}/json`);
      const tab = tabs.find((entry) => entry.type === "page");
      if (tab?.webSocketDebuggerUrl) return tab;
    } catch {
    }
    await delay(100);
  }
  throw new Error("timed out waiting for Chrome DevTools tab");
}

function fetchJSON(url) {
  return new Promise((resolve, reject) => {
    http.get(url, (res) => {
      let body = "";
      res.setEncoding("utf8");
      res.on("data", (chunk) => { body += chunk; });
      res.on("end", () => {
        try {
          resolve(JSON.parse(body));
        } catch (err) {
          reject(err);
        }
      });
    }).on("error", reject);
  });
}

function connectWebSocket(urlString) {
  const url = new URL(urlString);
  const key = Buffer.from(String(Math.random())).toString("base64");
  const socket = net.connect(Number(url.port), url.hostname);
  return new Promise((resolve, reject) => {
    socket.once("connect", () => {
      socket.write([
        `GET ${url.pathname}${url.search} HTTP/1.1`,
        `Host: ${url.host}`,
        "Upgrade: websocket",
        "Connection: Upgrade",
        `Sec-WebSocket-Key: ${key}`,
        "Sec-WebSocket-Version: 13",
        "",
        "",
      ].join("\r\n"));
    });
    let buffer = Buffer.alloc(0);
    socket.on("data", function onHandshake(chunk) {
      buffer = Buffer.concat([buffer, chunk]);
      const headerEnd = buffer.indexOf("\r\n\r\n");
      if (headerEnd < 0) return;
      socket.off("data", onHandshake);
      const rest = buffer.subarray(headerEnd + 4);
      resolve({ socket, buffer: rest });
    });
    socket.once("error", reject);
  });
}

function createCDP(connection) {
  let nextID = 1;
  let buffer = connection.buffer || Buffer.alloc(0);
  const pending = new Map();
  connection.socket.on("data", (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    for (;;) {
      const frame = readFrame(buffer);
      if (!frame) break;
      buffer = frame.rest;
      const message = JSON.parse(frame.payload);
      if (message.id && pending.has(message.id)) {
        const { resolve, reject } = pending.get(message.id);
        pending.delete(message.id);
        if (message.error) reject(new Error(JSON.stringify(message.error)));
        else resolve(message.result || {});
      }
    }
  });
  return {
    send(method, params = {}) {
      const id = nextID++;
      const payload = JSON.stringify({ id, method, params });
      connection.socket.write(writeFrame(payload));
      return new Promise((resolve, reject) => pending.set(id, { resolve, reject }));
    },
  };
}

async function waitForRunning(cdp) {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    const status = await evaluate(cdp, `document.getElementById("status")?.textContent || ""`);
    if (status === "Running") return;
    await delay(100);
  }
  throw new Error("timed out waiting for Vugra wasm status Running");
}

async function installHelpers(cdp) {
  await evaluate(cdp, `(() => {
    window.findVugraA11yNode = (role, name) => Array.from(document.querySelectorAll("#vugra-a11y [role]")).find((candidate) =>
      (candidate.getAttribute("role") || "") === role &&
      (candidate.getAttribute("aria-label") || "") === name
    ) || null;
  })()`);
}

async function pageState(cdp, samplePoints = []) {
  return evaluate(cdp, `(() => {
    const status = document.getElementById("status")?.textContent || "";
    const title = document.title;
    const canvas = document.getElementById("vugra-canvas");
    if (!canvas) return { ok: false, status, error: "missing canvas" };
    const ctx = canvas.getContext("2d");
    const sample = ctx.getImageData(0, 0, canvas.width, canvas.height).data;
    let nonBackgroundPixels = 0;
    for (let i = 0; i < sample.length; i += 4) {
      const r = sample[i], g = sample[i + 1], b = sample[i + 2], a = sample[i + 3];
      if (a !== 255 || r !== 250 || g !== 250 || b !== 250) {
        nonBackgroundPixels++;
      }
    }
    const samplePoints = ${JSON.stringify(samplePoints.map(({ x, y }) => ({ x, y })))};
    const pixels = {};
    for (const point of samplePoints) {
      const data = ctx.getImageData(point.x, point.y, 1, 1).data;
      pixels[point.x + "," + point.y] = {
        r: data[0],
        g: data[1],
        b: data[2],
        a: data[3],
        hex: "#" + [data[0], data[1], data[2]].map((value) => value.toString(16).padStart(2, "0")).join(""),
      };
    }
    const a11y = Array.from(document.querySelectorAll("#vugra-a11y [role]")).map((node) => ({
      role: node.getAttribute("role") || "",
      name: node.getAttribute("aria-label") || "",
      focused: node.getAttribute("data-vugra-focused") || "",
      checked: node.getAttribute("aria-checked") || "",
      disabled: node.getAttribute("aria-disabled") || "",
      x: Number(node.getAttribute("data-vugra-x") || "0"),
      y: Number(node.getAttribute("data-vugra-y") || "0"),
      width: Number(node.getAttribute("data-vugra-width") || "0"),
      height: Number(node.getAttribute("data-vugra-height") || "0"),
    }));
    return {
      ok: status === "Running" && canvas.width > 0 && canvas.height > 0 && nonBackgroundPixels > 0,
      status,
      title,
      canvas: { width: canvas.width, height: canvas.height, cssWidth: canvas.style.width, cssHeight: canvas.style.height },
      nonBackgroundPixels,
      pixels,
      text: a11y.map((node) => node.name).filter(Boolean),
      a11y,
    };
  })()`);
}

async function runAction(cdp, action) {
  switch (action.type) {
    case "click":
      await clickCanvas(cdp, action.x, action.y);
      return;
    case "hover":
      await hoverCanvas(cdp, action.x, action.y);
      return;
    case "drag":
      await dragCanvas(cdp, action.x1, action.y1, action.x2, action.y2);
      return;
    case "dblclick":
      await doubleClickCanvas(cdp, action.x, action.y);
      return;
    case "contextmenu":
      await contextMenuCanvas(cdp, action.x, action.y);
      return;
    case "a11y-click":
      await a11yClick(cdp, action.role, action.name);
      return;
    case "a11y-focus":
      await a11yFocus(cdp, action.role, action.name);
      return;
    case "wheel":
      await wheelCanvas(cdp, action.x, action.y, action.deltaY);
      return;
    case "text":
      await typeText(cdp, action.text);
      return;
    case "compose":
      await composeText(cdp, action.text);
      return;
    case "paste":
      await pasteText(cdp, action.text);
      return;
    case "key":
      await pressKey(cdp, action.key);
      return;
    case "a11y-text":
      await a11yText(cdp, action.role, action.name, action.text);
      return;
    case "a11y-key":
      await a11yKey(cdp, action.role, action.name, action.text);
      return;
    default:
      throw new Error(`unknown action ${action.type}`);
  }
}

async function clickCanvas(cdp, x, y) {
  const point = await evaluate(cdp, `(() => {
    const rect = document.getElementById("vugra-canvas").getBoundingClientRect();
    return { x: rect.left + ${JSON.stringify(x)}, y: rect.top + ${JSON.stringify(y)} };
  })()`);
  await cdp.send("Input.dispatchMouseEvent", { type: "mousePressed", x: point.x, y: point.y, button: "left", clickCount: 1 });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseReleased", x: point.x, y: point.y, button: "left", clickCount: 1 });
  await delay(100);
}

async function hoverCanvas(cdp, x, y) {
  const point = await canvasPoint(cdp, x, y);
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseMoved", x: point.x, y: point.y, button: "none" });
  await delay(100);
}

async function dragCanvas(cdp, x1, y1, x2, y2) {
  const start = await canvasPoint(cdp, x1, y1);
  const end = await canvasPoint(cdp, x2, y2);
  await cdp.send("Input.dispatchMouseEvent", { type: "mousePressed", x: start.x, y: start.y, button: "left", clickCount: 1 });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseMoved", x: end.x, y: end.y, button: "left" });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseReleased", x: end.x, y: end.y, button: "left", clickCount: 1 });
  await delay(100);
}

async function doubleClickCanvas(cdp, x, y) {
  const point = await canvasPoint(cdp, x, y);
  await cdp.send("Input.dispatchMouseEvent", { type: "mousePressed", x: point.x, y: point.y, button: "left", clickCount: 1 });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseReleased", x: point.x, y: point.y, button: "left", clickCount: 1 });
  await cdp.send("Input.dispatchMouseEvent", { type: "mousePressed", x: point.x, y: point.y, button: "left", clickCount: 2 });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseReleased", x: point.x, y: point.y, button: "left", clickCount: 2 });
  await delay(100);
}

async function contextMenuCanvas(cdp, x, y) {
  const point = await canvasPoint(cdp, x, y);
  await cdp.send("Input.dispatchMouseEvent", { type: "mousePressed", x: point.x, y: point.y, button: "right", clickCount: 1 });
  await cdp.send("Input.dispatchMouseEvent", { type: "mouseReleased", x: point.x, y: point.y, button: "right", clickCount: 1 });
  await delay(100);
}

async function canvasPoint(cdp, x, y) {
  return evaluate(cdp, `(() => {
    const rect = document.getElementById("vugra-canvas").getBoundingClientRect();
    return { x: rect.left + ${JSON.stringify(x)}, y: rect.top + ${JSON.stringify(y)} };
  })()`);
}

async function wheelCanvas(cdp, x, y, deltaY) {
  const point = await canvasPoint(cdp, x, y);
  await cdp.send("Input.dispatchMouseEvent", {
    type: "mouseWheel",
    x: point.x,
    y: point.y,
    deltaX: 0,
    deltaY,
  });
  await delay(100);
}

async function a11yClick(cdp, role, name) {
  const clicked = await evaluate(cdp, `(() => {
    const node = findVugraA11yNode(${JSON.stringify(role)}, ${JSON.stringify(name)});
    if (!node) return false;
    node.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
    return true;
  })()`);
  if (!clicked) {
    throw new Error(`a11y click target missing role=${role} name=${name}`);
  }
  await delay(100);
}

async function a11yFocus(cdp, role, name) {
  const focused = await evaluate(cdp, `(() => {
    const node = findVugraA11yNode(${JSON.stringify(role)}, ${JSON.stringify(name)});
    if (!node) return false;
    node.focus();
    return true;
  })()`);
  if (!focused) {
    throw new Error(`a11y focus target missing role=${role} name=${name}`);
  }
  await delay(100);
}

async function typeText(cdp, text) {
  for (const char of Array.from(text)) {
    await cdp.send("Input.dispatchKeyEvent", {
      type: "keyDown",
      key: char,
      code: "",
      windowsVirtualKeyCode: char.toUpperCase().charCodeAt(0),
      nativeVirtualKeyCode: char.toUpperCase().charCodeAt(0),
      text: char,
      unmodifiedText: char,
    });
    await cdp.send("Input.dispatchKeyEvent", {
      type: "keyUp",
      key: char,
      code: "",
      windowsVirtualKeyCode: char.toUpperCase().charCodeAt(0),
      nativeVirtualKeyCode: char.toUpperCase().charCodeAt(0),
    });
  }
  await delay(100);
}

async function composeText(cdp, text) {
  const dispatched = await evaluate(cdp, `(() => {
    const text = ${JSON.stringify(text)};
    window.dispatchEvent(new CompositionEvent("compositionstart", { bubbles: true, cancelable: true, data: "" }));
    window.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, cancelable: true, key: "Process", isComposing: true }));
    window.dispatchEvent(new CompositionEvent("compositionend", { bubbles: true, cancelable: true, data: text }));
    return true;
  })()`);
  if (!dispatched) {
    throw new Error("composition dispatch failed");
  }
  await delay(100);
}

async function pasteText(cdp, text) {
  const dispatched = await evaluate(cdp, `(() => {
    const text = ${JSON.stringify(text)};
    const data = new DataTransfer();
    data.setData("text/plain", text);
    const event = new ClipboardEvent("paste", { bubbles: true, cancelable: true, clipboardData: data });
    window.dispatchEvent(event);
    return true;
  })()`);
  if (!dispatched) {
    throw new Error("paste dispatch failed");
  }
  await delay(100);
}

async function pressKey(cdp, key) {
  const event = keyEvent(key);
  await cdp.send("Input.dispatchKeyEvent", {
    type: "keyDown",
    key: event.key,
    code: event.code,
    windowsVirtualKeyCode: event.windowsVirtualKeyCode,
    nativeVirtualKeyCode: event.windowsVirtualKeyCode,
    modifiers: event.modifiers,
  });
  await cdp.send("Input.dispatchKeyEvent", {
    type: "keyUp",
    key: event.key,
    code: event.code,
    windowsVirtualKeyCode: event.windowsVirtualKeyCode,
    nativeVirtualKeyCode: event.windowsVirtualKeyCode,
    modifiers: event.modifiers,
  });
  await delay(100);
}

async function a11yText(cdp, role, name, text) {
  let currentName = name;
  for (const char of Array.from(text)) {
    const dispatched = await evaluate(cdp, `(() => {
      const node = findVugraA11yNode(${JSON.stringify(role)}, ${JSON.stringify(currentName)});
      if (!node) return false;
      node.dispatchEvent(new InputEvent("input", { bubbles: true, cancelable: true, data: ${JSON.stringify(char)} }));
      return true;
    })()`);
    if (!dispatched) {
      throw new Error(`a11y text target missing role=${role} name=${currentName}`);
    }
    currentName += char;
    await delay(100);
  }
}

async function a11yKey(cdp, role, name, key) {
  const event = keyEvent(key);
  const dispatched = await evaluate(cdp, `(() => {
    const node = findVugraA11yNode(${JSON.stringify(role)}, ${JSON.stringify(name)});
    if (!node) return false;
    node.dispatchEvent(new KeyboardEvent("keydown", {
      bubbles: true,
      cancelable: true,
      key: ${JSON.stringify(event.key)},
      code: ${JSON.stringify(event.code)},
      ctrlKey: ${(event.modifiers & 2) !== 0},
      shiftKey: ${(event.modifiers & 8) !== 0},
      altKey: ${(event.modifiers & 1) !== 0},
      metaKey: ${(event.modifiers & 4) !== 0}
    }));
    return true;
  })()`);
  if (!dispatched) {
    throw new Error(`a11y key target missing role=${role} name=${name}`);
  }
  await delay(100);
}

function keyEvent(key) {
  switch (key) {
    case "Backspace":
      return { key: "Backspace", code: "Backspace", windowsVirtualKeyCode: 8, modifiers: 0 };
    case "Delete":
      return { key: "Delete", code: "Delete", windowsVirtualKeyCode: 46, modifiers: 0 };
    case "Tab":
      return { key: "Tab", code: "Tab", windowsVirtualKeyCode: 9, modifiers: 0 };
    case "Shift+Tab":
      return { key: "Tab", code: "Tab", windowsVirtualKeyCode: 9, modifiers: 8 };
    case "Enter":
      return { key: "Enter", code: "Enter", windowsVirtualKeyCode: 13, modifiers: 0 };
    case "ArrowLeft":
      return { key: "ArrowLeft", code: "ArrowLeft", windowsVirtualKeyCode: 37, modifiers: 0 };
    case "ArrowRight":
      return { key: "ArrowRight", code: "ArrowRight", windowsVirtualKeyCode: 39, modifiers: 0 };
    case "Home":
      return { key: "Home", code: "Home", windowsVirtualKeyCode: 36, modifiers: 0 };
    case "End":
      return { key: "End", code: "End", windowsVirtualKeyCode: 35, modifiers: 0 };
    case "ArrowUp":
      return { key: "ArrowUp", code: "ArrowUp", windowsVirtualKeyCode: 38, modifiers: 0 };
    case "ArrowDown":
      return { key: "ArrowDown", code: "ArrowDown", windowsVirtualKeyCode: 40, modifiers: 0 };
    case "Escape":
      return { key: "Escape", code: "Escape", windowsVirtualKeyCode: 27, modifiers: 0 };
    case " ":
      return { key: " ", code: "Space", windowsVirtualKeyCode: 32, modifiers: 0 };
    case "Mod+A":
      return { key: "a", code: "KeyA", windowsVirtualKeyCode: 65, modifiers: 2 };
    default:
      if (key.length === 1) {
        return { key, code: "", windowsVirtualKeyCode: key.toUpperCase().charCodeAt(0), modifiers: 0 };
      }
      return { key, code: key, windowsVirtualKeyCode: 0, modifiers: 0 };
  }
}

async function evaluate(cdp, expression) {
  const result = await cdp.send("Runtime.evaluate", {
    expression,
    awaitPromise: true,
    returnByValue: true,
  });
  if (result.exceptionDetails) {
    throw new Error(JSON.stringify(result.exceptionDetails));
  }
  return result.result?.value;
}

function readFrame(buffer) {
  if (buffer.length < 2) return null;
  const first = buffer[0];
  const second = buffer[1];
  let length = second & 0x7f;
  let offset = 2;
  if (length === 126) {
    if (buffer.length < 4) return null;
    length = buffer.readUInt16BE(2);
    offset = 4;
  } else if (length === 127) {
    throw new Error("large websocket frames are not supported");
  }
  const masked = (second & 0x80) !== 0;
  const maskOffset = masked ? 4 : 0;
  if (buffer.length < offset + maskOffset + length) return null;
  let payload = buffer.subarray(offset + maskOffset, offset + maskOffset + length).toString("utf8");
  if ((first & 0x0f) === 8) throw new Error("websocket closed");
  return { payload, rest: buffer.subarray(offset + maskOffset + length) };
}

function writeFrame(payload) {
  const data = Buffer.from(payload);
  let header;
  if (data.length < 126) {
    header = Buffer.alloc(6);
    header[0] = 0x81;
    header[1] = 0x80 | data.length;
    header.writeUInt32BE(0, 2);
  } else {
    header = Buffer.alloc(8);
    header[0] = 0x81;
    header[1] = 0x80 | 126;
    header.writeUInt16BE(data.length, 2);
    header.writeUInt32BE(0, 4);
  }
  return Buffer.concat([header, data]);
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function stopProcess(child) {
  if (child.exitCode !== null || child.signalCode !== null) {
    return Promise.resolve();
  }
  return new Promise((resolve) => {
    child.once("exit", resolve);
    child.kill();
    setTimeout(resolve, 2000);
  });
}

function rmRetry(target) {
  for (let attempt = 0; attempt < 5; attempt++) {
    try {
      fs.rmSync(target, { recursive: true, force: true, maxRetries: 3, retryDelay: 100 });
      return;
    } catch {
    }
  }
}
