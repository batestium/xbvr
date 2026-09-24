import { spawn } from 'node:child_process';
import { mkdtemp, mkdir, symlink, rm } from 'node:fs/promises';
import { once } from 'node:events';

const dir = await mkdtemp('/state/smoke-');
await mkdir(`${dir}/bin`);
for (const tool of ['ffmpeg', 'ffprobe']) await symlink(`/usr/bin/${tool}`, `${dir}/bin/${tool}`);
const child = spawn(process.env.XBVR_SMOKE_BINARY || '/src/dist/xbvr', [], {
  cwd: '/src',
  env: { ...process.env, XBVR_APPDIR: dir, DATABASE_URL: `sqlite:${dir}/main.db`, XBVR_SEARCHDIR: `${dir}/search-v2` },
  stdio: ['ignore', 'pipe', 'pipe'],
});
let log = '';
child.stdout.on('data', data => { log += data; });
child.stderr.on('data', data => { log += data; });
try {
  let ready = false;
  for (let n = 0; n < 60; n++) {
    if (child.exitCode !== null) throw Error(`XBVR exited with ${child.exitCode}`);
    try {
      const response = await fetch('http://127.0.0.1:9999/ui/', { signal: AbortSignal.timeout(1000) });
      if (response.status === 200 && /<!doctype html/i.test(await response.text())) { ready = true; break; }
    } catch {}
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  if (!ready) throw Error('UI did not become ready in 30 seconds');
  if (log.includes('Error setting up xbvr_data')) throw Error('Bundled data failed to load');
  const response = await fetch('http://127.0.0.1:9999/api/scene/search?q=abcd%20503&fileId=1', { signal: AbortSignal.timeout(5000) });
  const result = await response.json();
  if (response.status !== 200 || result.results !== 0) throw Error(`Unexpected search response: ${JSON.stringify(result)}`);
  console.log('PASS: Linux binary starts, embedded UI returns HTTP 200, search API returns HTTP 200 with an empty disposable database.');
} catch (error) {
  console.error(log);
  throw error;
} finally {
  if (child.exitCode === null) {
    const stopped = once(child, 'exit');
    child.kill('SIGTERM');
    await stopped;
  }
  await rm(dir, { recursive: true });
}
