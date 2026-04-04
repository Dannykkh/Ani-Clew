const BASE = '';

export async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + url, init);
  return res.json();
}

export async function postJSON<T>(url: string, body: unknown): Promise<T> {
  return fetchJSON(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export async function putJSON<T>(url: string, body: unknown): Promise<T> {
  return fetchJSON(url, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

// SSE streaming for chat
export async function* streamChat(messages: unknown[], model?: string): AsyncGenerator<string> {
  const res = await fetch(BASE + '/v1/messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      model: model || 'default',
      messages,
      max_tokens: 8192,
      stream: true,
    }),
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
      if (!trimmed || trimmed.startsWith('event:')) continue;
      if (trimmed.startsWith('data: ')) {
        try {
          const event = JSON.parse(trimmed.slice(6));
          if (event.type === 'content_block_delta' && event.delta?.type === 'text_delta') {
            yield event.delta.text;
          }
          if (event.type === 'message_stop') return;
        } catch { /* skip */ }
      }
    }
  }
}

// Types
export interface ProviderInfo {
  name: string;
  displayName: string;
  models: { id: string; displayName: string }[];
}

export interface RouteRule {
  role: string;
  provider: string;
  model: string;
  fallback?: { provider: string; model: string };
}

export interface CostEntry {
  provider: string;
  model: string;
  requests: number;
  tokens: number;
  cost: number;
}
