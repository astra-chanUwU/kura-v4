import { spawn } from "node:child_process";
import { createServer } from "node:net";

// Check both ports before starting either process, so a stale server is obvious.
async function assertFree(port) {
  const server = createServer();
  await new Promise((resolve, reject) => {
    server.once("error", () => reject(new Error(`Port ${port} is occupied. Stop the previous server first.`)));
    server.listen(port, "127.0.0.1", resolve);
  });
  await new Promise((resolve) => server.close(resolve));
}
// Ports come from .env (loaded by the caller), matching Vite's proxy.
const apiAddr = process.env.KURA_ADDR || "127.0.0.1:8094";
const apiPort = Number(apiAddr.split(":").pop());
const webPort = Number(new URL(process.env.KURA_PUBLIC_URL || "http://localhost:5174").port) || 5174;
await assertFree(apiPort);
await assertFree(webPort);
const children = [
  spawn("go", ["tool", "air"], { stdio: "inherit", detached: true, env: process.env }),
  spawn("pnpm", ["--filter", "@kura/frontend", "dev"], { stdio: "inherit", detached: true, env: process.env }),
];
let stopping = false;
function stop(code = 0) {
  if (stopping) return;
  stopping = true;
  process.exitCode = code;
  for (const child of children) {
    if (child.pid) { try { process.kill(-child.pid, "SIGTERM"); } catch {} }
  }
  const timer = setTimeout(() => {
    for (const child of children) {
      if (child.pid) { try { process.kill(-child.pid, "SIGKILL"); } catch {} }
    }
  }, 16000);
  timer.unref();
}
for (const child of children) {
  child.on("error", (error) => { console.error(error.message); stop(1); });
  child.on("exit", (code) => { if (!stopping) stop(code ?? 1); });
}
process.on("SIGINT", () => stop());
process.on("SIGTERM", () => stop());
