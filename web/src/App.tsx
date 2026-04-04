import { useState, useEffect } from 'react';
import { ActivityBar } from './components/ActivityBar';
import { SidePanel } from './components/SidePanel';
import { ChatPage } from './pages/Chat';
import { RoutesPage } from './pages/Routes';
import { CostsPage } from './pages/Costs';
import { KairosPage } from './pages/Kairos';
import { SettingsPage } from './pages/Settings';
import { MemoryPage } from './pages/Memory';
import { TeamPage } from './pages/Team';
import { fetchJSON } from './lib/api';
import { setLang, getLang } from './lib/i18n';

function App() {
  const [page, setPage] = useState('chat');
  const [, setLangTick] = useState(0);
  const [theme, setTheme] = useState<'dark' | 'light'>(() =>
    (localStorage.getItem('aniclew-theme') as 'dark' | 'light') || 'dark'
  );
  const [status, setStatus] = useState<any>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const data = await fetchJSON<any>('/api/config');
        setStatus(data);
      } catch { setStatus(null); }
    };
    load();
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('aniclew-theme', theme);
  }, [theme]);

  // Side panel visible for chat and files mode
  const showSidePanel = page === 'chat' || page === 'files';
  const sidePanelMode = page === 'files' ? 'files' : 'chat';


  return (
    <div className="flex h-screen">
      {/* Activity Bar (always visible) */}
      <ActivityBar
        active={page}
        onNavigate={setPage}
        onLangToggle={() => {
          setLang(getLang() === 'ko' ? 'en' : 'ko');
          setLangTick(n => n + 1);
        }}
        onThemeToggle={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
        theme={theme}
      />

      {/* Side Panel (for chat + files) */}
      {showSidePanel && (
        <SidePanel
          visible={true}
          mode={sidePanelMode as 'files' | 'chat'}
          onNewChat={() => { /* handled in Chat */ }}
        />
      )}

      {/* Main Content */}
      <main className="flex-1 overflow-hidden">
        {page === 'chat' && <ChatPage />}
        {page === 'files' && <ChatPage />} {/* Files mode = chat with file panel */}
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
        <div className="ml-auto">AniClew v1.0</div>
      </div>
    </div>
  );
}

export default App;
