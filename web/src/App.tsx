import { useState, useEffect, useCallback } from 'react';
import { ActivityBar } from './components/ActivityBar';
import { SidePanel } from './components/SidePanel';
import { ChatPage } from './pages/Chat';
import { RoutesPage } from './pages/Routes';
import { CostsPage } from './pages/Costs';
import { KairosPage } from './pages/Kairos';
import { SettingsPage } from './pages/Settings';
import { MemoryPage } from './pages/Memory';
import { TeamPage } from './pages/Team';
import { fetchJSON, putJSON } from './lib/api';
import './lib/i18n';

interface ProjectInfo {
  path: string;
  name: string;
  type: string;
  framework: string;
  fileCount: number;
  active: boolean;
}

function App() {
  const [page, setPage] = useState('chat');
  const [theme, setTheme] = useState<'dark' | 'light'>(() =>
    (localStorage.getItem('aniclew-theme') as 'dark' | 'light') || 'dark'
  );
  const [status, setStatus] = useState<any>(null);
  const [loadSessionId, setLoadSessionId] = useState<string | null>(null);
  const [projects, setProjects] = useState<ProjectInfo[]>([]);
  const [showProjectPicker, setShowProjectPicker] = useState(false);

  const activeProject = projects.find(p => p.active);

  const loadProjects = useCallback(async () => {
    try {
      const data = await fetchJSON<ProjectInfo[]>('/api/projects');
      setProjects(data);
    } catch { setProjects([]); }
  }, []);

  useEffect(() => {
    const load = async () => {
      try {
        const data = await fetchJSON<any>('/api/config');
        setStatus(data);
      } catch { setStatus(null); }
    };
    load();
    loadProjects();
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, [loadProjects]);

  async function switchProject(path: string) {
    await putJSON('/api/workspace', { path });
    await loadProjects();
    setShowProjectPicker(false);
  }

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('aniclew-theme', theme);
  }, [theme]);

  // Side panel visible for chat and files mode
  const showSidePanel = page === 'chat' || page === 'files';
  const sidePanelMode = page === 'files' ? 'files' : 'chat';


  return (
    <div className="flex h-screen w-full">
      {/* Activity Bar (always visible) */}
      <ActivityBar
        active={page}
        onNavigate={setPage}
        onThemeToggle={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
        theme={theme}
      />

      {/* Side Panel (for chat + files) */}
      {showSidePanel && (
        <SidePanel
          visible={true}
          mode={sidePanelMode as 'files' | 'chat'}
          onNewChat={() => setLoadSessionId('__new__')}
          onSessionClick={(id) => setLoadSessionId(id)}
          onProjectSwitch={() => loadProjects()}
        />
      )}

      {/* Main Content */}
      <main className="flex-1 min-w-0 h-[calc(100vh-24px)] overflow-hidden flex">
        {page === 'chat' && <ChatPage loadSessionId={loadSessionId} onSessionLoaded={() => setLoadSessionId(null)} />}
        {page === 'files' && <ChatPage loadSessionId={loadSessionId} onSessionLoaded={() => setLoadSessionId(null)} />}
        {page === 'routes' && <RoutesPage />}
        {page === 'costs' && <CostsPage />}
        {page === 'kairos' && <KairosPage />}
        {page === 'settings' && (
          <div className="overflow-y-auto h-full">
            <SettingsPage />
            <div className="border-t border-[var(--color-border)]">
              <MemoryPage />
            </div>
            <div className="border-t border-[var(--color-border)]">
              <TeamPage />
            </div>
          </div>
        )}
      </main>

      {/* Status Bar */}
      <div className="fixed bottom-0 left-12 right-0 h-6 bg-[var(--color-surface)] border-t border-[var(--color-border)] flex items-center px-3 text-[10px] text-[var(--color-text2)] gap-4 z-50">
        <div className="flex items-center gap-1.5">
          <div className={`w-1.5 h-1.5 rounded-full ${status ? 'bg-[var(--color-green)]' : 'bg-[var(--color-red)]'}`} />
          {status ? `${status.provider} / ${status.model}` : 'Offline'}
        </div>
        {status?.routerEnabled && <span>Router ON</span>}

        {/* Project Picker in Status Bar */}
        <div className="relative">
          <button
            onClick={() => setShowProjectPicker(!showProjectPicker)}
            className="flex items-center gap-1 px-1.5 py-0.5 rounded hover:bg-[var(--color-surface2)] transition-colors"
          >
            <span>{activeProject ? `${activeProject.name}` : 'No Project'}</span>
            <span>{showProjectPicker ? '▴' : '▾'}</span>
          </button>
          {showProjectPicker && projects.length > 0 && (
            <div className="absolute bottom-6 left-0 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg shadow-lg min-w-48 max-h-60 overflow-y-auto z-50">
              {projects.map(p => (
                <button
                  key={p.path}
                  onClick={() => switchProject(p.path)}
                  className={`w-full text-left px-3 py-1.5 text-[11px] hover:bg-[var(--color-surface2)] transition-colors flex items-center gap-2 ${p.active ? 'text-[var(--color-accent)]' : ''}`}
                >
                  <span>{p.active ? '●' : '○'}</span>
                  <span className="truncate">{p.name}</span>
                  <span className="text-[9px] text-[var(--color-text2)] ml-auto">{p.type}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="ml-auto">AniClew v1.0</div>
      </div>
    </div>
  );
}

export default App;
