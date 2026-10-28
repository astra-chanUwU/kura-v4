import { createWriteStream } from "node:fs";
import { mkdir, readdir, rename, stat, unlink, writeFile } from "node:fs/promises";
import path from "node:path";
import { pipeline } from "node:stream/promises";
import { Readable } from "node:stream";

const IMAGE_EXTS = new Set([".jpg", ".jpeg", ".png", ".gif"]);
const API = "https://safebooru.org/index.php?page=dapi&s=post&q=index&json=1";
const USER_AGENT =
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36";
const DELAY_MS = 150;
const TIMEOUT_MS = 30_000;
const MIN_BYTES = 5 * 1024;
const MAX_TAGS = 30;

function parseArgs(argv) {
  let count = 600;
  let dir = ".local/seed/safebooru-anime";
  let tags = "";
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--count" && argv[i + 1]) count = Number(argv[++i]);
    else if (argv[i] === "--dir" && argv[i + 1]) dir = argv[++i];
    else if (argv[i] === "--tags" && argv[i + 1]) tags = argv[++i];
  }
  if (!Number.isFinite(count) || count < 0) throw new Error("Invalid --count");
  return { count, dir, tags };
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function fetchRetry(url, init = {}) {
  let last;
  for (let attempt = 0; attempt <= 2; attempt++) {
    try {
      const res = await fetch(url, {
        ...init,
        headers: { "User-Agent": USER_AGENT, ...init.headers },
        signal: AbortSignal.timeout(TIMEOUT_MS),
      });
      await sleep(DELAY_MS);
      return res;
    } catch (err) {
      last = err;
      if (attempt < 2) await sleep(DELAY_MS);
    }
  }
  throw last;
}

function absUrl(url) {
  if (!url) return "";
  if (url.startsWith("//")) return `https:${url}`;
  if (/^https?:\/\//i.test(url)) return url;
  return `https://safebooru.org/${url.replace(/^\//, "")}`;
}

function extFromUrl(url) {
  try {
    return path.extname(new URL(url).pathname).toLowerCase();
  } catch {
    return "";
  }
}

function idFromName(name) {
  const m = /^safebooru (\d+)\./i.exec(name);
  return m ? m[1] : null;
}

// Tags from --tags that the API actually returned. They are written
// first so the 30-tag sidecar cap cannot drop the search itself.
let pinnedTags = [];

function trimTags(raw) {
  const tags = raw.trim().split(/\s+/).filter(Boolean);
  const keep = [];
  const seen = new Set();
  for (const tag of pinnedTags) {
    if (!tags.includes(tag) || seen.has(tag)) continue;
    keep.push(tag);
    seen.add(tag);
  }
  for (const tag of tags) {
    if (seen.has(tag)) continue;
    if (keep.length >= MAX_TAGS) break;
    keep.push(tag);
    seen.add(tag);
  }
  return keep.join(" ");
}

async function listImageFiles(dir) {
  let names;
  try {
    names = await readdir(dir);
  } catch (err) {
    if (err.code === "ENOENT") {
      await mkdir(dir, { recursive: true });
      return [];
    }
    throw err;
  }
  return names.filter((name) => IMAGE_EXTS.has(path.extname(name).toLowerCase()));
}

async function sidecarPath(dir, id) {
  return path.join(dir, `safebooru ${id}.tags`);
}

async function writeSidecar(dir, id, tags) {
  const file = await sidecarPath(dir, id);
  await writeFile(file, `${trimTags(tags)}\n`, "utf8");
}

async function fetchPost(id) {
  const res = await fetchRetry(`${API}&id=${id}`);
  if (!res.ok) throw new Error(`post ${id}: HTTP ${res.status}`);
  const data = await res.json();
  const post = Array.isArray(data) ? data[0] : data;
  if (!post?.id) throw new Error(`post ${id}: not found`);
  return post;
}

async function fetchPage(pid, tags) {
  let url = `${API}&limit=100&pid=${pid}`;
  if (tags) url += `&tags=${encodeURIComponent(tags)}`;
  const res = await fetchRetry(url);
  if (!res.ok) throw new Error(`page ${pid}: HTTP ${res.status}`);
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

function pickUrl(post) {
  const useSample = post.sample === true || post.sample === 1 || post.sample === "1" || post.sample === "true";
  const url = absUrl(useSample ? post.sample_url : post.file_url);
  const ext = extFromUrl(url);
  if (!IMAGE_EXTS.has(ext)) return null;
  return { url, ext };
}

async function downloadImage(url, dest) {
  const tmp = `${dest}.part`;
  try {
    const res = await fetchRetry(url);
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const type = res.headers.get("content-type") || "";
    if (!type.startsWith("image/")) throw new Error(`content-type ${type}`);
    await pipeline(Readable.fromWeb(res.body), createWriteStream(tmp));
    const info = await stat(tmp);
    if (info.size <= MIN_BYTES) throw new Error(`size ${info.size}`);
    await rename(tmp, dest);
  } catch (err) {
    try {
      await unlink(tmp);
    } catch {}
    throw err;
  }
}

const { count: target, dir, tags } = parseArgs(process.argv.slice(2));
pinnedTags = tags.split(/\s+/).filter((tag) => tag && !tag.startsWith("-"));
await mkdir(dir, { recursive: true });

let downloaded = 0;
let backfilled = 0;
let failures = 0;

const images = await listImageFiles(dir);
const haveId = new Set();
for (const name of images) {
  const id = idFromName(name);
  if (id) haveId.add(id);
}
let total = images.length;

let progressAt = Math.floor(total / 25) * 25;
function tickProgress() {
  const mark = Math.floor(total / 25) * 25;
  if (mark > progressAt) {
    console.log(`${total}/${target} images`);
    progressAt = mark;
  }
}

for (const name of images) {
  const id = idFromName(name);
  if (!id) continue;
  try {
    await stat(await sidecarPath(dir, id));
  } catch {
    try {
      const post = await fetchPost(id);
      await writeSidecar(dir, id, post.tags || "");
      backfilled++;
    } catch (err) {
      failures++;
      console.error(`backfill ${id}: ${err.message}`);
    }
  }
}

if (total < target) {
  for (let pid = 0; total < target; pid++) {
    let page;
    try {
      page = await fetchPage(pid, tags);
    } catch (err) {
      failures++;
      console.error(err.message);
      break;
    }
    if (page.length === 0) {
      console.log(`API exhausted at page ${pid}: ${total} images (target ${target})`);
      break;
    }
    for (const post of page) {
      if (total >= target) break;
      const id = String(post.id);
      if (haveId.has(id)) continue;
      const picked = pickUrl(post);
      if (!picked) continue;
      const dest = path.join(dir, `safebooru ${id}${picked.ext}`);
      try {
        await downloadImage(picked.url, dest);
        await writeSidecar(dir, id, post.tags || "");
        haveId.add(id);
        total++;
        downloaded++;
        tickProgress();
      } catch (err) {
        failures++;
        console.error(`download ${id}: ${err.message}`);
      }
    }
  }
}

console.log(
  `summary: ${total} images, ${downloaded} downloaded this run, ${backfilled} sidecars backfilled, ${failures} failures`,
);
