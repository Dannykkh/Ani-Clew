import { useState, useEffect } from 'react';
import { fetchJSON } from '../lib/api';
import { getLang } from '../lib/i18n';
import { listSessions, type SessionSummary } from '../lib/sessions';

interface ProjectInfo {
  type: string;
  name: string;
  framework: string;
  fileCount: number;
  fileTree: string;
}

interface Props {
  visible: boolean;
  mode: 'files' | 'chat';
  onFileClick?: (path: string) => void;
  onSessionClick?: (id: string) => void;
  onNewChat?: () => void;
}

export function SidePanel({ visible, mode, onFileClick, onSessionClick, onNewChat }: Props) {
  const [project, setProject] = useState<ProjectInfo | null>(null);
  const [workspace, setWorkspace] = useState('');
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [browseEntries, setBrowseEntries] = useState<any[]>([]);
  const [browsePath, setBrowsePath] = useState('');
  const ko = getLang() === 'ko';

  useEffect(() => {
    fetchJSON<any>('/api/workspace').then((data) => {
      setWorkspace(data.path);
      setProject(data.project);
      loadBrowse(data.path);
    }).catch(() => {});
    listSessions().then(setSessions).catch(() => {});
  }, []);

  async function loadBrowse(path: string) {
    try {
      const data = await fetchJSON<any>(`/api/browse?path=${encodeURIComponent(path)}`);
      setBrowseEntries(data.entries || []);
      setBrowsePath(data.current);
    } catch {}
  }

  async function selectWorkspace(path: string) {
    await fetchJSON('/api/workspace', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    });
    setWorkspace(path);
    const projData = await fetchJSON<any>('/api/workspace');
    setProject(projData.project);
    loadBrowse(path);
  }

  if (!visible) return null;

  const projectTypes: Record<string, string> = {
    go: '🔵', node: '🟢', python: '🐍', rust: '🦀', java: '☕', dotnet: '🟣',
  };

  return (
    <div className="w-64 bg-[var(--color-surface)] border-r border-[var(--color-border)] flex flex-col h-screen overflow-hidden shrink-0">
      {/* Project Header */}
      <div className="p-3 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <span>{project ? (projectTypes[project.type] || '▸') : '▸'}</span>
          <div className="min-w-0">
            <div className="text-xs font-semibold truncate">{project?.name || 'No project'}</div>
            <div className="text-[10px] text-[var(--color-text2)] truncate">{workspace}</div>
          </div>
        </div>
        {project && (
          <div className="flex gap-1 mt-1.5">
            <span className="text-[9px] bg-[var(--color-accent)]/20 text-[var(--color-accent)] px-1 py-0.5 rounded">{project.type}</span>
            {project.framework && <span className="text-[9px] bg-[var(--color-green)]/20 text-[var(--color-green)] px-1 py-0.5 rounded">{project.framework}</span>}
            <span className="text-[9px] text-[var(--color-text2)]">{project.fileCount} files</span>
          </div>
        )}
      </div>

      {mode === 'files' ? (
        /* ── File Tree ── */
        <div className="flex-1 overflow-y-auto">
          {/* Browse path */}
          <div className="px-2 py-1.5 text-[10px] text-[var(--color-text2)] uppercase border-b border-[var(--color-border)] flex items-center justify-between">
            <span>{ko ? '파일' : 'Files'}</span>
            <button onClick={() => {
              const parent = browsePath.replace(/[/\\][^/\\]+$/, '');
              loadBrowse(parent);
            }} className="text-[10px] hover:text-[var(--color-text)]">↑</button>
          </div>
          <div className="py-1">
            {browseEntries.map((entry: any) => (
              <div
                key={entry.name}
                onClick={() => {
                  if (entry.isDir) {
                    loadBrowse(browsePath.replace(/\\/g, '/') + '/' + entry.name);
                  } else {
                    onFileClick?.(entry.name);
                  }
                }}
                className="flex items-center gap-1.5 px-3 py-1 text-xs cursor-pointer hover:bg-[var(--color-surface2)] transition-colors"
              >
                <span className="text-[10px]">{entry.isDir ? '▸' : '·'}</span>
                <span className={`truncate ${entry.isProject ? 'text-[var(--color-accent)]' : ''}`}>{entry.name}</span>
                {entry.isProject && (
                  <button
                    onClick={(e) => { e.stopPropagation(); selectWorkspace(browsePath.replace(/\\/g, '/') + '/' + entry.name); }}
                    className="ml-auto text-[9px] text-[var(--color-accent)] hover:underline shrink-0"
                  >
                    {ko ? '선택' : 'Set'}
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      ) : (
        /* ── Session History ── */
        <div className="flex-1 overflow-y-auto">
          <div className="px-2 py-1.5 text-[10px] text-[var(--color-text2)] uppercase border-b border-[var(--color-border)] flex items-center justify-between">
            <span>{ko ? '대화 기록' : 'History'}</span>
            <button onClick={onNewChat} className="text-[10px] bg-[var(--color-accent)] text-white px-1.5 py-0.5 rounded hover:opacity-80">
              + {ko ? '새 대화' : 'New'}
            </button>
          </div>
          <div className="py-1">
            {sessions.length === 0 ? (
              <div className="px-3 py-4 text-xs text-[var(--color-text2)] text-center">{ko ? '대화 기록 없음' : 'No history'}</div>
            ) : (
              sessions.map((s) => (
                <div
                  key={s.id}
                  onClick={() => onSessionClick?.(s.id)}
                  className="px-3 py-2 cursor-pointer hover:bg-[var(--color-surface2)] transition-colors"
                >
                  <div className="text-xs font-medium truncate">{s.title}</div>
                  <div className="text-[10px] text-[var(--color-text2)] truncate">{s.preview}</div>
                  <div className="text-[9px] text-[var(--color-text2)] mt-0.5">{s.turns} {ko ? '턴' : 'turns'} · {s.model}</div>
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
