import { useState, useEffect } from 'react';
import { fetchJSON, postJSON, putJSON } from '../lib/api';

export function KairosPage() {
  const [status, setStatus] = useState<any>(null);
  const [tasks, setTasks] = useState<any[]>([]);
  const [logs, setLogs] = useState<any[]>([]);
  const [gitStatus, setGitStatus] = useState<any>(null);
  const [newTask, setNewTask] = useState({ id: '', type: 'custom', description: '' });

  useEffect(() => {
    load();
    const interval = setInterval(load, 5000);
    return () => clearInterval(interval);
  }, []);

  async function load() {
    const [s, t, l, g] = await Promise.all([
      fetchJSON('/api/kairos'),
      fetchJSON('/api/kairos/tasks'),
      fetchJSON('/api/kairos/logs'),
      fetchJSON('/api/kairos/git').catch(() => null),
    ]);
    setStatus(s);
    setTasks(Array.isArray(t) ? t : []);
    setLogs(Array.isArray(l) ? l : []);
    setGitStatus(g);
  }

  async function start() { await postJSON('/api/kairos/start', {}); load(); }
  async function stop() { await postJSON('/api/kairos/stop', {}); load(); }
  async function setAutonomy(mode: string) { await putJSON('/api/kairos/autonomy', { mode }); load(); }

  async function addTask() {
    if (!newTask.id || !newTask.description) return;
    await postJSON('/api/kairos/tasks', newTask);
    setNewTask({ id: '', type: 'custom', description: '' });
    load();
  }

  return (
    <div className="p-6 w-full">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold">KAIROS Daemon</h1>
        {status?.enabled ? (
          <button onClick={stop} className="px-4 py-2 bg-[var(--color-red)] text-white rounded-lg text-sm">Stop</button>
        ) : (
          <button onClick={start} className="px-4 py-2 bg-[var(--color-green)] text-white rounded-lg text-sm">Start</button>
        )}
      </div>

      {/* Status */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-1">State</div>
          <div className="text-lg font-semibold">{status?.state || '—'}</div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-1">Autonomy</div>
          <div className="text-lg font-semibold">{status?.autonomy || '—'}</div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-1">Tasks</div>
          <div className="text-lg font-semibold">{status?.tasks || 0}</div>
        </div>
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-1">Tick Interval</div>
          <div className="text-lg font-semibold">{status?.tickInterval || '—'}</div>
        </div>
      </div>

      {/* Autonomy Mode */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4 mb-4">
        <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Autonomy Mode</div>
        <div className="flex gap-2">
          {['collaborative', 'autonomous', 'night'].map((mode) => (
            <button key={mode} onClick={() => setAutonomy(mode)}
              className={`px-4 py-2 rounded-lg text-sm capitalize ${
                status?.autonomy === mode
                  ? 'bg-[var(--color-accent)] text-white'
                  : 'bg-[var(--color-surface2)] text-[var(--color-text2)] hover:text-[var(--color-text)]'
              }`}>
              {mode}
            </button>
          ))}
        </div>
      </div>

      {/* Git Status */}
      {gitStatus && !gitStatus.error && (
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4 mb-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Git Status</div>
          <div className="flex items-center gap-3 mb-2">
            <span className="text-sm font-mono text-[var(--color-accent)]">{gitStatus.branch}</span>
            {gitStatus.aheadBy > 0 && (
              <span className="text-[10px] bg-yellow-500/20 text-yellow-400 px-1.5 py-0.5 rounded">
                {gitStatus.aheadBy} unpushed
              </span>
            )}
            {gitStatus.staged?.length === 0 && gitStatus.modified?.length === 0 && gitStatus.untracked?.length === 0 && (
              <span className="text-[10px] bg-green-500/20 text-green-400 px-1.5 py-0.5 rounded">Clean</span>
            )}
          </div>
          <div className="flex gap-4 text-xs">
            {gitStatus.staged?.length > 0 && (
              <div><span className="text-green-400">{gitStatus.staged.length} staged</span></div>
            )}
            {gitStatus.modified?.length > 0 && (
              <div><span className="text-yellow-400">{gitStatus.modified.length} modified</span>
                <span className="text-[var(--color-text2)] ml-1">({gitStatus.modified.slice(0, 3).join(', ')}{gitStatus.modified.length > 3 ? '...' : ''})</span>
              </div>
            )}
            {gitStatus.untracked?.length > 0 && (
              <div><span className="text-red-400">{gitStatus.untracked.length} untracked</span></div>
            )}
          </div>
          {gitStatus.lastCommit && (
            <div className="mt-2 text-[10px] text-[var(--color-text2)]">
              Last: {gitStatus.lastCommit}
            </div>
          )}
        </div>
      )}

      {/* Add Task */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4 mb-4">
        <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Add Background Task</div>
        <div className="flex gap-2">
          <input value={newTask.id} onChange={(e) => setNewTask({ ...newTask, id: e.target.value })}
            placeholder="Task ID" className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm text-[var(--color-text)] w-32" />
          <input value={newTask.description} onChange={(e) => setNewTask({ ...newTask, description: e.target.value })}
            placeholder="Description (e.g., Run tests every hour)" className="flex-1 bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm text-[var(--color-text)]" />
          <button onClick={addTask} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-sm">Add</button>
        </div>
      </div>

      {/* Tasks List */}
      {tasks.length > 0 && (
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4 mb-4">
          <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Active Tasks</div>
          {tasks.map((t: any) => (
            <div key={t.id} className="flex items-center justify-between py-2 border-b border-[var(--color-border)] last:border-0">
              <div>
                <span className="text-sm font-mono text-[var(--color-accent)]">{t.id}</span>
                <span className="text-sm ml-2">{t.description}</span>
              </div>
              <span className="text-xs text-[var(--color-text2)]">{t.type}</span>
            </div>
          ))}
        </div>
      )}

      {/* Logs */}
      <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-xl p-4 font-mono text-xs max-h-80 overflow-y-auto">
        <div className="text-[var(--color-text2)] uppercase mb-2 font-sans text-xs">Daemon Logs</div>
        {logs.length === 0 ? (
          <div className="text-[var(--color-text2)]">No logs yet</div>
        ) : (
          logs.map((l: any, i: number) => (
            <div key={i} className="flex gap-2 py-0.5">
              <span className="text-[var(--color-text2)] w-20 shrink-0">
                {new Date(l.time).toLocaleTimeString()}
              </span>
              <span className="text-[var(--color-accent)] w-24 shrink-0">{l.action}</span>
              <span className="text-[var(--color-text)]">{l.detail}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
