// fs.js: the file system Go's js/wasm runtime looks for on globalThis, kept in localStorage so that
// settings, high scores and saved games outlive the page.
//
// Go's syscall package on GOOS=js does its file I/O through a Node-style `fs` object, and
// wasm_exec.js installs one that fails every call with ENOSYS unless one is already there. So this
// loads first. It is a whole small file system rather than a map from a few names to strings,
// because the game writes the way it does on a disk -- a temporary file, an fsync, a rename over
// the old one -- and a save that is half written when the tab closes must not replace a whole one.
//
// The tree is one JSON value under one key, read once at load. What is kept after that is only
// what changed: when something is made durable -- a close after a write, an fsync, a rename, a
// removal, a new directory -- the paths written and removed since the last time are applied to the
// tree as it is stored now, and it is stored again. Not the whole tree as this page holds it,
// which is what would let a second tab, or a page that could not read one entry, put back a tree
// without the files somebody else kept since.
//
// A write the browser refuses is the call's own error, as a full or read-only disk would be on a
// native build, so the game says the save failed rather than that it was made.
//
// localStorage belongs to the page's origin, and on github.io that is every project of the same
// account, so the key is namespaced. Browsers also clear it: Safari after seven days without a
// visit, and any browser when the player clears site data. docs/IMPROVEMENTS.md 3.10 says what
// that costs and what would answer it.
(function () {
  "use strict";
  const KEY = "glidergo:fs:v1";
  const S_IFDIR = 0o040000, S_IFREG = 0o100000;
  const O = { O_WRONLY: 1, O_RDWR: 2, O_CREAT: 64, O_EXCL: 128, O_TRUNC: 512, O_APPEND: 1024 };
  const nodes = new Map(); // absolute path -> {dir, mode, mtime} or {data, mode, mtime}
  const fds = new Map();   // descriptor -> {path, pos, write, append, dirty}
  let nextFd = 3;

  // The paths written or made, and the paths removed, since the tree was last kept.
  const changed = new Set(), removed = new Set();
  function touched(p) { changed.add(p); removed.delete(p); }
  function gone(p) { removed.add(p); changed.delete(p); }

  // The last reason the browser gave for not keeping the tree, "" while it is keeping it, and
  // whether that reason was a full store.
  const state = { failed: "", full: false };

  // What the program prints. Both streams go to the console; glidergoLog keeps every line, and
  // glidergoErr the standard error's alone, for the page to show when the game stops badly.
  const decoder = { 1: new TextDecoder("utf-8"), 2: new TextDecoder("utf-8") };
  const partial = { 1: "", 2: "" };
  globalThis.glidergoLog = [];
  globalThis.glidergoErr = [];

  function err(code) { const e = new Error(code); e.code = code; return e; }

  function norm(p) {
    const out = [];
    for (const part of String(p).split("/")) {
      if (part === "" || part === ".") continue;
      if (part === "..") out.pop(); else out.push(part);
    }
    return "/" + out.join("/");
  }
  function parent(p) { const i = p.lastIndexOf("/"); return i <= 0 ? "/" : p.slice(0, i); }

  function toBase64(u8) {
    let s = "";
    for (let i = 0; i < u8.length; i += 0x8000) s += String.fromCharCode.apply(null, u8.subarray(i, i + 0x8000));
    return btoa(s);
  }
  function fromBase64(s) {
    const b = atob(s), u8 = new Uint8Array(b.length);
    for (let i = 0; i < b.length; i++) u8[i] = b.charCodeAt(i);
    return u8;
  }
  function encode(n) {
    return n.dir ? { dir: 1, mtime: n.mtime } : { data: toBase64(n.data), mode: n.mode, mtime: n.mtime };
  }
  function decode(n) {
    if (n && n.dir) return { dir: true, mode: S_IFDIR | 0o755, mtime: n.mtime };
    if (!n || typeof n.data !== "string") throw new Error("not a file or a directory");
    return { data: fromBase64(n.data), mode: n.mode, mtime: n.mtime };
  }
  function isTree(t) { return t !== null && typeof t === "object" && !Array.isArray(t); }

  function load() {
    nodes.set("/", { dir: true, mode: S_IFDIR | 0o755, mtime: Date.now() });
    let raw = null;
    try { raw = localStorage.getItem(KEY); } catch (e) { state.failed = String(e); return; }
    if (!raw) return;
    let tree = null;
    try { tree = JSON.parse(raw); } catch (e) { /* set aside below */ }
    if (!isTree(tree)) {
      // Not a tree at all. It is moved to a key of its own, where it can still be looked at,
      // and this session starts with no files; if even the move is refused, it stays where it
      // is, and nothing this session does can write over it (persist reads the stored tree
      // first, and fails).
      try {
        localStorage.setItem(KEY + ":unreadable", raw);
        localStorage.removeItem(KEY);
        console.warn("glidergo: the stored files could not be read; they are kept under " +
                     KEY + ":unreadable, and this session starts with none");
      } catch (e) {
        state.failed = String(e);
        console.warn("glidergo: the stored files could not be read, or moved aside", e);
      }
      return;
    }
    for (const [p, n] of Object.entries(tree)) {
      // One entry this cannot read is left out of this session and left as it is stored, since
      // only what changes is ever written back.
      try { nodes.set(p, decode(n)); } catch (e) {
        console.warn("glidergo: " + p + " could not be read, and is left as it is stored", e);
      }
    }
  }

  // persist applies what changed to the stored tree, and answers whether the browser kept it.
  function persist() {
    if (globalThis.glidergoKeepNothing) return true; // tools/webcheck's frame dumps, which are 6 MB
    if (changed.size === 0 && removed.size === 0) return true;
    try {
      const raw = localStorage.getItem(KEY);
      const tree = raw ? JSON.parse(raw) : {};
      if (!isTree(tree)) throw new Error("the stored files are not a tree, and are not written over");
      for (const p of removed) delete tree[p];
      for (const p of changed) {
        const n = nodes.get(p);
        if (n) tree[p] = encode(n); else delete tree[p];
      }
      localStorage.setItem(KEY, JSON.stringify(tree));
    } catch (e) {
      if (!state.failed) console.warn("glidergo: this browser would not keep the game's files", e);
      state.failed = String(e);
      state.full = !!e && (e.name === "QuotaExceededError" || e.name === "NS_ERROR_DOM_QUOTA_REACHED");
      return false;
    }
    changed.clear();
    removed.clear();
    state.failed = "";
    return true;
  }

  // kept is persist for a call that has to say whether it worked: a full store is a full disk to
  // Go, and any other refusal a read-only one, and the game's own message for either follows. What
  // was not kept is kept the next time a write succeeds.
  function kept() {
    if (!persist()) throw err(state.full ? "ENOSPC" : "EROFS");
  }

  function stat(n) {
    const size = n.dir ? 4096 : n.data.length;
    return {
      dev: 1, ino: 1, mode: n.mode, nlink: 1, uid: 0, gid: 0, rdev: 0, size, blksize: 4096,
      blocks: Math.ceil(size / 512), atimeMs: n.mtime, mtimeMs: n.mtime, ctimeMs: n.mtime,
      isDirectory() { return !!n.dir; },
    };
  }
  function lookup(path) { const n = nodes.get(norm(path)); if (!n) throw err("ENOENT"); return n; }
  function needDir(p) { const d = nodes.get(p); if (!d) throw err("ENOENT"); if (!d.dir) throw err("ENOTDIR"); }
  function children(p) {
    const pre = p === "/" ? "/" : p + "/", out = [];
    for (const k of nodes.keys()) {
      if (k !== p && k.startsWith(pre) && !k.slice(pre.length).includes("/")) out.push(k.slice(pre.length));
    }
    return out;
  }
  function grow(n, end) {
    if (end > n.data.length) { const d = new Uint8Array(end); d.set(n.data); n.data = d; }
  }

  function print(fd, u8) {
    partial[fd] += decoder[fd].decode(u8, { stream: true });
    let nl;
    while ((nl = partial[fd].indexOf("\n")) !== -1) {
      const line = partial[fd].slice(0, nl);
      partial[fd] = partial[fd].slice(nl + 1);
      globalThis.glidergoLog.push(line);
      if (fd === 2) globalThis.glidergoErr.push(line);
      (fd === 2 ? console.warn : console.log)(line);
    }
    return u8.length;
  }

  // Go's fsCall passes a callback last and accepts an answer made before it returns, so every
  // call is answered at once. A thrown error with no code is a bug in this file, not an errno,
  // and is let through to be seen.
  function call(f) {
    return function (...args) {
      const cb = args.pop();
      let r;
      try { r = f(...args); } catch (e) { if (!e.code) throw e; cb(e); return; }
      cb(null, r);
    };
  }

  globalThis.fs = {
    constants: O,
    writeSync(fd, buf) { if (fd === 1 || fd === 2) return print(fd, buf); throw err("EBADF"); },
    open: call((path, flags, mode) => {
      const p = norm(path);
      let n = nodes.get(p);
      const write = (flags & (O.O_WRONLY | O.O_RDWR)) !== 0;
      if (n && (flags & O.O_CREAT) && (flags & O.O_EXCL)) throw err("EEXIST");
      let made = false;
      if (!n) {
        if (!(flags & O.O_CREAT)) throw err("ENOENT");
        needDir(parent(p));
        n = { data: new Uint8Array(0), mode: S_IFREG | (mode & 0o777), mtime: Date.now() };
        nodes.set(p, n);
        made = true;
      }
      if (n.dir && write) throw err("EISDIR");
      if (!n.dir && (flags & O.O_TRUNC) && n.data.length > 0) {
        n.data = new Uint8Array(0);
        n.mtime = Date.now();
        made = true;
      }
      if (made) touched(p);
      const fd = nextFd++;
      fds.set(fd, { path: p, pos: 0, write, append: (flags & O.O_APPEND) !== 0, dirty: made });
      return fd;
    }),
    close: call((fd) => {
      const f = fds.get(fd); if (!f) throw err("EBADF");
      fds.delete(fd);
      if (f.dirty) kept();
    }),
    fstat: call((fd) => { const f = fds.get(fd); if (!f) throw err("EBADF"); return stat(lookup(f.path)); }),
    stat: call((path) => stat(lookup(path))),
    lstat: call((path) => stat(lookup(path))),
    read: call((fd, buf, off, len, pos) => {
      const f = fds.get(fd); if (!f) throw err("EBADF");
      const n = lookup(f.path); if (n.dir) throw err("EISDIR");
      const at = pos === null || pos === undefined ? f.pos : pos;
      const got = Math.max(0, Math.min(len, n.data.length - at));
      buf.set(n.data.subarray(at, at + got), off);
      if (pos === null || pos === undefined) f.pos += got;
      return got;
    }),
    write: call((fd, buf, off, len, pos) => {
      if (fd === 1 || fd === 2) return print(fd, buf.subarray(off, off + len));
      const f = fds.get(fd); if (!f || !f.write) throw err("EBADF");
      const n = lookup(f.path);
      let at = pos === null || pos === undefined ? f.pos : pos;
      if (f.append) at = n.data.length;
      grow(n, at + len);
      n.data.set(buf.subarray(off, off + len), at);
      n.mtime = Date.now();
      if (pos === null || pos === undefined) f.pos = at + len;
      f.dirty = true;
      touched(f.path);
      return len;
    }),
    fsync: call((fd) => { if (!fds.has(fd)) throw err("EBADF"); kept(); }),
    ftruncate: call((fd, len) => {
      const f = fds.get(fd); if (!f) throw err("EBADF");
      const n = lookup(f.path); n.data = n.data.slice(0, len); grow(n, len);
      f.dirty = true;
      touched(f.path);
    }),
    truncate: call((path, len) => {
      const p = norm(path); const n = lookup(p);
      n.data = n.data.slice(0, len); grow(n, len);
      touched(p); kept();
    }),
    mkdir: call((path, perm) => {
      const p = norm(path);
      if (nodes.has(p)) throw err("EEXIST");
      needDir(parent(p));
      nodes.set(p, { dir: true, mode: S_IFDIR | (perm & 0o777), mtime: Date.now() });
      touched(p); kept();
    }),
    readdir: call((path) => { const p = norm(path); needDir(p); return children(p); }),
    rename: call((from, to) => {
      const a = norm(from), b = norm(to);
      const n = lookup(a);
      needDir(parent(b));
      const there = nodes.get(b);
      if (there && there.dir && !n.dir) throw err("EISDIR");
      if (there && there.dir && children(b).length) throw err("ENOTEMPTY");
      const moves = [[a, b]];
      if (n.dir) for (const k of [...nodes.keys()]) if (k.startsWith(a + "/")) moves.push([k, b + k.slice(a.length)]);
      for (const [x, y] of moves) { const m = nodes.get(x); nodes.delete(x); nodes.set(y, m); gone(x); touched(y); }
      for (const f of fds.values()) for (const [x, y] of moves) if (f.path === x) f.path = y;
      kept();
    }),
    unlink: call((path) => {
      const p = norm(path); const n = lookup(p);
      if (n.dir) throw err("EISDIR");
      nodes.delete(p); gone(p); kept();
    }),
    rmdir: call((path) => {
      const p = norm(path); const n = lookup(p);
      if (!n.dir) throw err("ENOTDIR");
      if (children(p).length) throw err("ENOTEMPTY");
      nodes.delete(p); gone(p); kept();
    }),
    chmod: call((path, mode) => {
      const p = norm(path); const n = lookup(p);
      n.mode = (n.mode & ~0o777) | (mode & 0o777);
      touched(p); kept();
    }),
    fchmod: call((fd, mode) => {
      const f = fds.get(fd); if (!f) throw err("EBADF");
      const n = lookup(f.path); n.mode = (n.mode & ~0o777) | (mode & 0o777);
      f.dirty = true;
      touched(f.path);
    }),
    utimes: call((path, atime, mtime) => {
      const p = norm(path); lookup(p).mtime = mtime * 1000;
      touched(p); kept();
    }),
    chown: call(() => {}), fchown: call(() => {}), lchown: call(() => {}),
    link: call(() => { throw err("ENOSYS"); }),
    symlink: call(() => { throw err("ENOSYS"); }),
    readlink: call(() => { throw err("ENOSYS"); }),
  };

  // The rest of what wasm_exec.js would otherwise stub with ENOSYS. The working directory is the
  // root, so a relative path means what it would from there.
  globalThis.process = {
    getuid() { return -1; }, getgid() { return -1; }, geteuid() { return -1; }, getegid() { return -1; },
    getgroups() { throw err("ENOSYS"); }, pid: -1, ppid: -1,
    umask() { return 0o022; }, cwd() { return "/"; }, chdir() { throw err("ENOSYS"); },
  };

  load();

  // For the page and for tools/webcheck: the tree as this page holds it, a way to keep what has
  // changed now, the key, and why the browser last refused to keep it ("" while it is keeping it).
  globalThis.glidergoFS = { nodes, persist, key: KEY, state };
})();
