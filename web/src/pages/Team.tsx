import { useState, useEffect } from 'react';
import { fetchJSON, postJSON } from '../lib/api';

export function TeamPage() {
  const [users, setUsers] = useState<any[]>([]);
  const [audit, setAudit] = useState<any[]>([]);
  const [newUser, setNewUser] = useState({ name: '', role: 'developer', budget: 50 });
  const [copied, setCopied] = useState('');
  const [teamTasks, setTeamTasks] = useState([{ id: 'task-1', name: '', description: '', files: '', dependsOn: '' }]);
  const [teamRunning, setTeamRunning] = useState(false);
  const [teamLog, setTeamLog] = useState<string[]>([]);

  useEffect(() => { load(); }, []);

  async function load() {
    const [u, a] = await Promise.all([
      fetchJSON('/api/gateway/users'),
      fetchJSON('/api/gateway/audit'),
    ]);
    setUsers(Array.isArray(u) ? u : []);
    setAudit(Array.isArray(a) ? a : []);
  }

  async function addUser() {
    if (!newUser.name) return;
    await postJSON('/api/gateway/users', newUser);
    setNewUser({ name: '', role: 'developer', budget: 50 });
    load();
  }

  function copyToken(token: string) {
    navigator.clipboard.writeText(token);
    setCopied(token);
    setTimeout(() => setCopied(''), 2000);
  }

  function addTaskRow() {
    const id = `task-${teamTasks.length + 1}`;
    setTeamTasks([...teamTasks, { id, name: '', description: '', files: '', dependsOn: '' }]);
  }

  async function runTeam() {
    setTeamRunning(true);
    setTeamLog([]);
    const tasks = teamTasks.filter(t => t.name && t.description).map(t => ({
      id: t.id,
      name: t.name,
      description: t.description,
      files: t.files ? t.files.split(',').map(f => f.trim()) : [],
      dependsOn: t.dependsOn ? t.dependsOn.split(',').map(d => d.trim()) : [],
    }));

    try {
      const res = await fetch('/api/team', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: 'web-team', tasks }),
      });

      const reader = res.body!.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const event = JSON.parse(line.slice(6));
              if (event.type === 'status' || event.type === 'text') {
                setTeamLog(prev => [...prev, typeof event.data === 'string' ? event.data : JSON.stringify(event.data)]);
              }
            } catch {}
          }
        }
      }
    } catch (err) {
      setTeamLog(prev => [...prev, `Error: ${err}`]);
    }
    setTeamRunning(false);
  }

  return (
    <div className="p-6 w-full overflow-y-auto h-full">
      <h1 className="text-xl font-semibold mb-6">Team</h1>

      {/* Agent Team Execution */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4 mb-6">
        <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Agent Team — Wave Execution</div>

        {teamTasks.map((task, i) => (
          <div key={task.id} className="flex gap-2 mb-2">
            <input value={task.name} onChange={(e) => { const t = [...teamTasks]; t[i].name = e.target.value; setTeamTasks(t); }}
              placeholder="Task name" className="w-32 bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)]" />
            <input value={task.description} onChange={(e) => { const t = [...teamTasks]; t[i].description = e.target.value; setTeamTasks(t); }}
              placeholder="Description (what to do)" className="flex-1 bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)]" />
            <input value={task.files} onChange={(e) => { const t = [...teamTasks]; t[i].files = e.target.value; setTeamTasks(t); }}
              placeholder="Files (comma)" className="w-36 bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)]" />
            <input value={task.dependsOn} onChange={(e) => { const t = [...teamTasks]; t[i].dependsOn = e.target.value; setTeamTasks(t); }}
              placeholder="Depends on" className="w-28 bg-[var(--color-bg)] border border-[var(--color-border)] rounded px-2 py-1.5 text-xs text-[var(--color-text)]" />
          </div>
        ))}

        <div className="flex gap-2 mt-3">
          <button onClick={addTaskRow} className="text-xs px-3 py-1.5 bg-[var(--color-surface2)] text-[var(--color-text)] rounded hover:bg-[var(--color-border)]">
            + Add Task
          </button>
          <button onClick={runTeam} disabled={teamRunning}
            className="text-xs px-4 py-1.5 bg-[var(--color-accent)] text-white rounded hover:opacity-80 disabled:opacity-40">
            {teamRunning ? 'Running...' : 'Execute Team'}
          </button>
        </div>

        {teamLog.length > 0 && (
          <div className="mt-3 bg-[var(--color-bg)] rounded p-3 font-mono text-xs max-h-40 overflow-y-auto">
            {teamLog.map((log, i) => (
              <div key={i} className="py-0.5 text-[var(--color-text2)]">{log}</div>
            ))}
          </div>
        )}
      </div>

      <h2 className="text-lg font-semibold mb-4">Team Gateway</h2>

      {/* Add User */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4 mb-6">
        <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Add Team Member</div>
        <div className="flex gap-2">
          <input value={newUser.name} onChange={(e) => setNewUser({ ...newUser, name: e.target.value })}
            placeholder="Name" className="w-40 bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm text-[var(--color-text)]" />
          <select value={newUser.role} onChange={(e) => setNewUser({ ...newUser, role: e.target.value })}
            className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm text-[var(--color-text)]">
            <option value="admin">Admin</option>
            <option value="developer">Developer</option>
            <option value="viewer">Viewer</option>
          </select>
          <input type="number" value={newUser.budget} onChange={(e) => setNewUser({ ...newUser, budget: +e.target.value })}
            placeholder="Monthly budget ($)" className="w-32 bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg px-3 py-2 text-sm text-[var(--color-text)]" />
          <button onClick={addUser} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-sm">Add</button>
        </div>
      </div>

      {/* Users */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl overflow-hidden mb-6">
        <table className="w-full">
          <thead>
            <tr className="bg-[var(--color-surface2)] text-[var(--color-text2)] text-xs uppercase">
              <th className="text-left px-4 py-2">Name</th>
              <th className="text-left px-4 py-2">Role</th>
              <th className="text-left px-4 py-2">Budget</th>
              <th className="text-left px-4 py-2">Spent</th>
              <th className="text-left px-4 py-2">Token</th>
            </tr>
          </thead>
          <tbody>
            {users.length === 0 ? (
              <tr><td colSpan={5} className="text-center py-6 text-[var(--color-text2)] text-sm">No users yet</td></tr>
            ) : (
              users.map((u: any) => (
                <tr key={u.id} className="border-t border-[var(--color-border)]">
                  <td className="px-4 py-2.5 text-sm font-medium">{u.name}</td>
                  <td className="px-4 py-2.5">
                    <span className={`text-xs px-2 py-0.5 rounded-full ${
                      u.role === 'admin' ? 'bg-orange-500/15 text-orange-400' :
                      u.role === 'developer' ? 'bg-blue-500/15 text-blue-400' : 'bg-gray-500/15 text-gray-400'
                    }`}>{u.role}</span>
                  </td>
                  <td className="px-4 py-2.5 text-sm">${u.monthlyBudget}</td>
                  <td className="px-4 py-2.5 text-sm">
                    <span className={u.currentSpend > u.monthlyBudget * 0.8 ? 'text-[var(--color-red)]' : ''}>
                      ${u.currentSpend.toFixed(2)}
                    </span>
                  </td>
                  <td className="px-4 py-2.5">
                    <button onClick={() => copyToken(u.token)}
                      className="text-xs font-mono bg-[var(--color-bg)] px-2 py-1 rounded border border-[var(--color-border)] hover:border-[var(--color-accent)]">
                      {copied === u.token ? 'Copied!' : u.token.slice(0, 12) + '...'}
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Audit Log */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl p-4">
        <div className="text-xs text-[var(--color-text2)] uppercase mb-3">Audit Log</div>
        <div className="max-h-60 overflow-y-auto font-mono text-xs space-y-1">
          {audit.length === 0 ? (
            <div className="text-[var(--color-text2)]">No audit entries yet</div>
          ) : (
            audit.map((a: any, i: number) => (
              <div key={i} className="flex gap-3 py-1 border-b border-[var(--color-border)] last:border-0">
                <span className="text-[var(--color-text2)] w-20">{new Date(a.time).toLocaleTimeString()}</span>
                <span className="text-[var(--color-accent)]">{a.userId}</span>
                <span>{a.provider}/{a.model}</span>
                <span className="text-[var(--color-green)]">${a.cost.toFixed(4)}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
