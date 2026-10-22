import { spawn } from "node:child_process";
const [command, ...args] = process.argv.slice(2);
if (!command) throw new Error("Missing command");
const child = spawn(command, args, { stdio: "inherit", env: process.env });
child.on("error", (error) => { console.error(error.message); process.exitCode = 1; });
child.on("exit", (code) => { process.exitCode = code ?? 1; });
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => child.kill(signal));
