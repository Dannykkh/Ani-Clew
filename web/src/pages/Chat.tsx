import { useState, useRef, useEffect, useCallback } from 'react';
import { t } from '../lib/i18n';
import { listSessions, getSession, saveSession, deleteSession, type SessionSummary, type SessionMessage } from '../lib/sessions';

interface ChatMessage {
  role: 'user' | 'assistant' | 'tool';
  content: string;
  toolName?: string;
  toolInput?: Record<string, unknown> | string;
  toolResult?: string;
  isError?: boolean;
  timestamp: Date;
}

export function ChatPage() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [streaming, setStreaming] = useState(false);
  const [status, setStatus] = useState('');
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [showHistory, setShowHistory] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Load session list
  useEffect(() => {
    listSessions().then(setSessions).catch(() => {});
  }, []);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, status]);

  // Auto-save session after each message
  const autoSave = useCallback(async (msgs: ChatMessage[], sid: string | null) => {
    if (msgs.length === 0) return;
    const sessionMsgs: SessionMessage[] = msgs.map((m) => ({
      role: m.role,
      content: m.content,
      toolName: m.toolName,
      toolInput: m.toolInput,
      toolResult: m.toolResult,
      isError: m.isError,
      timestamp: m.timestamp.toISOString(),
    }));
    const result = await saveSession({ id: sid || undefined, messages: sessionMsgs });
    if (result.id && !sid) {
      setSessionId(result.id);
    }
    listSessions().then(setSessions).catch(() => {});
  }, []);

  async function loadSession(id: string) {
    const sess = await getSession(id);
    const msgs: ChatMessage[] = (sess.messages || []).map((m) => ({
      ...m,
      timestamp: new Date(m.timestamp),
    }));
    setMessages(msgs);
    setSessionId(sess.id);
    setShowHistory(false);
  }

  function newChat() {
    setMessages([]);
    setSessionId(null);
    setShowHistory(false);
  }

  async function handleDelete(id: string) {
    await deleteSession(id);
    setSessions((prev) => prev.filter((s) => s.id !== id));
    if (sessionId === id) {
      newChat();
    }
  }

  async function send() {
    const text = input.trim();
    if (!text || streaming) return;

    setInput('');
    const userMsg: ChatMessage = { role: 'user', content: text, timestamp: new Date() };
    const newMsgs = [...messages, userMsg];
    setMessages([...newMsgs, { role: 'assistant', content: '', timestamp: new Date() }]);
    setStreaming(true);
    setStatus(t('chat.thinking'));

    const apiMessages = newMsgs
      .filter((m) => m.role === 'user' || m.role === 'assistant')
      .map((m) => ({ role: m.role, content: m.content }));

    try {
      const res = await fetch('/api/agent', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ messages: apiMessages }),
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
          const trimmed = line.trim();
          if (!trimmed.startsWith('data: ')) continue;
          try {
            handleAgentEvent(JSON.parse(trimmed.slice(6)));
          } catch { /* skip */ }
        }
      }
    } catch (err) {
      setMessages((prev) => {
        const updated = [...prev];
        const last = updated[updated.length - 1];
        if (last?.role === 'assistant') {
          updated[updated.length - 1] = { ...last, content: last.content + `\n\n[Error: ${err}]` };
        }
        return updated;
      });
    } finally {
      setStreaming(false);
      setStatus('');
      inputRef.current?.focus();
      // Auto-save after response complete
      setMessages((prev) => {
        autoSave(prev, sessionId);
        return prev;
      });
    }
  }

  function handleAgentEvent(event: { type: string; data: any }) {
    switch (event.type) {
      case 'text':
        setMessages((prev) => {
          const updated = [...prev];
          const last = updated[updated.length - 1];
          if (last?.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: last.content + event.data };
          }
          return updated;
        });
        break;
      case 'tool_start':
        setStatus(`${t('chat.executing')} ${event.data.name}`);
        break;
      case 'tool_input':
        setMessages((prev) => [...prev, {
          role: 'tool' as const, content: '', toolName: event.data.name,
          toolInput: event.data.input, timestamp: new Date(),
        }]);
        break;
      case 'tool_result':
        setMessages((prev) => {
          const updated = [...prev];
          for (let i = updated.length - 1; i >= 0; i--) {
            if (updated[i].role === 'tool' && updated[i].toolName === event.data.name && !updated[i].toolResult) {
              updated[i] = { ...updated[i], toolResult: event.data.result, isError: event.data.isError };
              break;
            }
          }
          return updated;
        });
        setStatus('');
        setMessages((prev) => [...prev, { role: 'assistant', content: '', timestamp: new Date() }]);
        break;
      case 'status':
        setStatus(event.data);
        break;
      case 'done':
      case 'stream_end':
        setStatus('');
        setMessages((prev) => {
          if (prev[prev.length - 1]?.role === 'assistant' && prev[prev.length - 1]?.content === '') {
            return prev.slice(0, -1);
          }
          return prev;
        });
        break;
      case 'error':
        setMessages((prev) => [...prev, { role: 'assistant', content: `Error: ${event.data}`, timestamp: new Date() }]);
        break;
    }
  }

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <div className="px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-surface)] flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setShowHistory(!showHistory)}
            className="px-2.5 py-1.5 rounded-lg text-xs text-[var(--color-text2)] hover:bg-[var(--color-surface2)] hover:text-[var(--color-text)] transition-colors"
          >
            {showHistory ? '✕' : '📋'} {t('session.history')}
          </button>
          <button
            onClick={newChat}
            className="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--color-accent)] text-white hover:bg-[var(--color-accent2)] transition-colors"
          >
            + {t('session.new')}
          </button>
          <h1 className="text-sm font-semibold">{t('chat.title')}</h1>
        </div>
        <div className="flex items-center gap-2 text-xs text-[var(--color-text2)]">
          {status && (
            <div className="flex items-center gap-1.5 text-[var(--color-accent)]">
              <div className="w-2 h-2 rounded-full bg-[var(--color-accent)] animate-pulse" />
              {status}
            </div>
          )}
          <span>{messages.filter((m) => m.role === 'user').length} {t('session.turns')}</span>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Session History Panel */}
        {showHistory && (
          <div className="w-64 border-r border-[var(--color-border)] bg-[var(--color-surface)] overflow-y-auto shrink-0">
            <div className="p-2 space-y-1">
              {sessions.length === 0 ? (
                <div className="text-xs text-[var(--color-text2)] text-center py-8">{t('session.noHistory')}</div>
              ) : (
                sessions.map((s) => (
                  <div
                    key={s.id}
                    className={`group px-3 py-2.5 rounded-lg cursor-pointer transition-colors ${
                      sessionId === s.id ? 'bg-[var(--color-surface2)]' : 'hover:bg-[var(--color-surface2)]'
                    }`}
                    onClick={() => loadSession(s.id)}
                  >
                    <div className="text-xs font-medium text-[var(--color-text)] truncate">{s.title}</div>
                    <div className="text-[10px] text-[var(--color-text2)] truncate mt-0.5">{s.preview}</div>
                    <div className="flex items-center justify-between mt-1">
                      <span className="text-[10px] text-[var(--color-text2)]">
                        {s.turns} {t('session.turns')} · {s.model}
                      </span>
                      <button
                        onClick={(e) => { e.stopPropagation(); handleDelete(s.id); }}
                        className="opacity-0 group-hover:opacity-100 text-[10px] text-[var(--color-red)] hover:underline"
                      >
                        {t('session.delete')}
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

        {/* Messages Area */}
        <div className="flex-1 flex flex-col overflow-hidden">
          <div className="flex-1 overflow-y-auto p-6 space-y-3">
            {messages.length === 0 && (
              <div className="flex flex-col items-center justify-center h-full text-[var(--color-text2)]">
                <div className="text-4xl mb-4">🤖</div>
                <div className="text-lg font-medium mb-2">{t('chat.welcome')}</div>
                <div className="text-sm">{t('chat.welcomeSub')}</div>
                <div className="text-xs mt-2 text-[var(--color-text2)]">{t('chat.tools')}</div>
              </div>
            )}

            {messages.map((msg, i) => {
              if (msg.role === 'user') {
                return (
                  <div key={i} className="flex justify-end">
                    <div className="max-w-[70%] bg-[var(--color-accent)] text-white rounded-xl rounded-br-sm px-4 py-3 text-sm">
                      <div className="whitespace-pre-wrap">{msg.content}</div>
                      <div className="text-[10px] text-white/50 mt-1">{msg.timestamp.toLocaleTimeString()}</div>
                    </div>
                  </div>
                );
              }
              if (msg.role === 'tool') {
                return (
                  <div key={i} className="mx-4">
                    <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-lg overflow-hidden">
                      <div className="flex items-center gap-2 px-3 py-2 bg-[var(--color-surface2)] border-b border-[var(--color-border)]">
                        <span className="text-xs font-semibold text-[var(--color-accent)]">{msg.toolName ?? ''}</span>
                        {msg.isError && <span className="text-[10px] text-[var(--color-red)] bg-red-500/10 px-1.5 py-0.5 rounded">ERROR</span>}
                      </div>
                      {msg.toolInput && (
                        <div className="px-3 py-2 border-b border-[var(--color-border)]">
                          <div className="text-[10px] text-[var(--color-text2)] uppercase mb-1">{t('chat.input')}</div>
                          <pre className="text-xs text-[var(--color-text)] overflow-x-auto whitespace-pre-wrap font-mono">
                            {typeof msg.toolInput === 'string' ? msg.toolInput : String(JSON.stringify(msg.toolInput, null, 2))}
                          </pre>
                        </div>
                      )}
                      {msg.toolResult ? (
                        <div className="px-3 py-2 max-h-60 overflow-y-auto">
                          <div className="text-[10px] text-[var(--color-text2)] uppercase mb-1">{t('chat.output')}</div>
                          <pre className={`text-xs overflow-x-auto whitespace-pre-wrap font-mono ${msg.isError ? 'text-[var(--color-red)]' : 'text-[var(--color-green)]'}`}>
                            {msg.toolResult}
                          </pre>
                        </div>
                      ) : (
                        <div className="px-3 py-2 flex items-center gap-2">
                          <div className="w-3 h-3 border-2 border-[var(--color-accent)] border-t-transparent rounded-full animate-spin" />
                          <span className="text-xs text-[var(--color-text2)]">{t('chat.executing')}</span>
                        </div>
                      )}
                    </div>
                  </div>
                );
              }
              if (msg.content === '' && streaming) {
                return (
                  <div key={i} className="flex justify-start">
                    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl rounded-bl-sm px-4 py-3">
                      <div className="flex gap-1">
                        <div className="w-2 h-2 bg-[var(--color-text2)] rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                        <div className="w-2 h-2 bg-[var(--color-text2)] rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                        <div className="w-2 h-2 bg-[var(--color-text2)] rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                      </div>
                    </div>
                  </div>
                );
              }
              if (msg.content === '') return null;
              return (
                <div key={i} className="flex justify-start">
                  <div className="max-w-[80%] bg-[var(--color-surface)] border border-[var(--color-border)] rounded-xl rounded-bl-sm px-4 py-3 text-sm">
                    <div className="chat-md whitespace-pre-wrap">{msg.content}</div>
                  </div>
                </div>
              );
            })}
            <div ref={bottomRef} />
          </div>

          {/* Input */}
          <div className="p-4 border-t border-[var(--color-border)] bg-[var(--color-surface)]">
            <div className="flex gap-3 items-end max-w-4xl mx-auto">
              <textarea
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); } }}
                placeholder={t('chat.placeholder')}
                rows={1}
                className="flex-1 bg-[var(--color-bg)] border border-[var(--color-border)] rounded-xl px-4 py-3 text-sm text-[var(--color-text)] resize-none focus:outline-none focus:border-[var(--color-accent)] placeholder:text-[var(--color-text2)]"
                style={{ minHeight: '44px', maxHeight: '200px' }}
                onInput={(e) => {
                  const el = e.currentTarget;
                  el.style.height = 'auto';
                  el.style.height = Math.min(el.scrollHeight, 200) + 'px';
                }}
              />
              <button
                onClick={send}
                disabled={streaming || !input.trim()}
                className="px-5 py-3 bg-[var(--color-accent)] text-white rounded-xl text-sm font-medium disabled:opacity-40 hover:bg-[var(--color-accent2)] transition-colors"
              >
                {streaming ? '...' : t('chat.send')}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
