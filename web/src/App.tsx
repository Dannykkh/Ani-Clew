import { useState, useEffect } from 'react';
import { Sidebar } from './components/Sidebar';
import { ChatPage } from './pages/Chat';
import { SettingsPage } from './pages/Settings';
import { RoutesPage } from './pages/Routes';
import { CostsPage } from './pages/Costs';
import { KairosPage } from './pages/Kairos';
import { MemoryPage } from './pages/Memory';
import { TeamPage } from './pages/Team';
import { ExplorerPage } from './pages/Explorer';
import { fetchJSON } from './lib/api';

function App() {
  const [page, setPage] = useState('chat');
  const [status, setStatus] = useState<{ provider: string; model: string; router: boolean } | null>(null);
  const [, setLangTick] = useState(0);
  const [theme, setTheme] = useState<'dark' | 'light'>(() =>
    (localStorage.getItem('aniclew-theme') as 'dark' | 'light') || 'dark'
  );

  useEffect(() => {
    const load = async () => {
      try {
        const data = await fetchJSON<any>('/api/config');
        setStatus({ provider: data.provider, model: data.model, router: data.routerEnabled });
      } catch {
        setStatus(null);
      }
    };
    load();
    const interval = setInterval(load, 10000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('aniclew-theme', theme);
  }, [theme]);

  return (
    <>
      <Sidebar
        active={page}
        onNavigate={setPage}
        onLangChange={() => setLangTick((n) => n + 1)}
        onThemeToggle={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
        theme={theme}
        status={status}
      />
      <main className="flex-1 overflow-hidden">
        {page === 'chat' && <ChatPage />}
        {page === 'explorer' && <ExplorerPage />}
        {page === 'settings' && <SettingsPage />}
        {page === 'routes' && <RoutesPage />}
        {page === 'costs' && <CostsPage />}
        {page === 'kairos' && <KairosPage />}
        {page === 'memory' && <MemoryPage />}
        {page === 'team' && <TeamPage />}
      </main>
    </>
  );
}

export default App;
