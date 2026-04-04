import { getLang } from '../lib/i18n';

interface NavItem {
  id: string;
  icon: string;
  label: string;
  labelKo: string;
  section: 'top' | 'bottom';
}

const items: NavItem[] = [
  { id: 'chat', icon: '💬', label: 'Chat', labelKo: '채팅', section: 'top' },
  { id: 'files', icon: '📁', label: 'Files', labelKo: '파일', section: 'top' },
  { id: 'routes', icon: '🔀', label: 'Routes', labelKo: '라우팅', section: 'top' },
  { id: 'costs', icon: '💰', label: 'Costs', labelKo: '비용', section: 'bottom' },
  { id: 'kairos', icon: '🤖', label: 'KAIROS', labelKo: 'KAIROS', section: 'bottom' },
  { id: 'settings', icon: '⚙️', label: 'Settings', labelKo: '설정', section: 'bottom' },
];

interface Props {
  active: string;
  onNavigate: (id: string) => void;
  onLangToggle: () => void;
  onThemeToggle: () => void;
  theme: 'dark' | 'light';
}

export function ActivityBar({ active, onNavigate, onLangToggle, onThemeToggle, theme }: Props) {
  const ko = getLang() === 'ko';
  const top = items.filter(i => i.section === 'top');
  const bottom = items.filter(i => i.section === 'bottom');

  return (
    <div className="w-12 bg-[var(--color-surface)] border-r border-[var(--color-border)] flex flex-col items-center py-2 shrink-0 h-screen sticky top-0">
      {/* Logo */}
      <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[var(--color-accent)] to-purple-400 flex items-center justify-center text-white text-xs font-bold mb-3">
        A
      </div>

      {/* Top items */}
      <div className="flex-1 flex flex-col items-center gap-1">
        {top.map((item) => (
          <button
            key={item.id}
            onClick={() => onNavigate(item.id)}
            title={ko ? item.labelKo : item.label}
            className={`w-10 h-10 rounded-lg flex items-center justify-center text-base transition-colors ${
              active === item.id
                ? 'bg-[var(--color-accent)]/15 text-[var(--color-accent)]'
                : 'text-[var(--color-text2)] hover:bg-[var(--color-surface2)] hover:text-[var(--color-text)]'
            }`}
          >
            {item.icon}
          </button>
        ))}
      </div>

      {/* Bottom items */}
      <div className="flex flex-col items-center gap-1 mb-2">
        {bottom.map((item) => (
          <button
            key={item.id}
            onClick={() => onNavigate(item.id)}
            title={ko ? item.labelKo : item.label}
            className={`w-10 h-10 rounded-lg flex items-center justify-center text-base transition-colors ${
              active === item.id
                ? 'bg-[var(--color-accent)]/15 text-[var(--color-accent)]'
                : 'text-[var(--color-text2)] hover:bg-[var(--color-surface2)] hover:text-[var(--color-text)]'
            }`}
          >
            {item.icon}
          </button>
        ))}
        <button onClick={onLangToggle} title="Language" className="w-10 h-10 rounded-lg flex items-center justify-center text-base text-[var(--color-text2)] hover:bg-[var(--color-surface2)]">
          🌐
        </button>
        <button onClick={onThemeToggle} title="Theme" className="w-10 h-10 rounded-lg flex items-center justify-center text-base text-[var(--color-text2)] hover:bg-[var(--color-surface2)]">
          {theme === 'dark' ? '☀️' : '🌙'}
        </button>
      </div>
    </div>
  );
}
