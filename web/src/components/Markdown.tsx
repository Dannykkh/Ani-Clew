import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

function ThinkingBlock({ content }: { content: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="my-2 rounded-lg border border-[var(--color-border)] overflow-hidden">
      <button
        onClick={() => setOpen(!open)}
        className="w-full px-3 py-1.5 text-xs text-[var(--color-text2)] bg-[var(--color-surface2)] flex items-center gap-1.5 hover:bg-[var(--color-border)] transition-colors"
      >
        <span>{open ? '▾' : '▸'}</span>
        <span>Thinking</span>
        <span className="text-[9px] ml-auto">{content.length} chars</span>
      </button>
      {open && (
        <div className="px-3 py-2 text-xs text-[var(--color-text2)] whitespace-pre-wrap font-mono leading-relaxed max-h-60 overflow-y-auto bg-[var(--color-bg)]">
          {content}
        </div>
      )}
    </div>
  );
}

interface Props {
  content: string;
}

export function Markdown({ content }: Props) {
  // Extract <think>...</think> blocks and render separately
  const parts: { type: 'text' | 'thinking'; content: string }[] = [];
  const thinkRegex = /<think>([\s\S]*?)(<\/think>|$)/g;
  let lastIndex = 0;
  let match;
  while ((match = thinkRegex.exec(content)) !== null) {
    if (match.index > lastIndex) {
      parts.push({ type: 'text', content: content.slice(lastIndex, match.index) });
    }
    parts.push({ type: 'thinking', content: match[1] });
    lastIndex = match.index + match[0].length;
  }
  if (lastIndex < content.length) {
    parts.push({ type: 'text', content: content.slice(lastIndex) });
  }

  // If no thinking blocks, render normally
  if (parts.length <= 1 && parts[0]?.type === 'text') {
    return <MarkdownContent content={content} />;
  }

  return (
    <>
      {parts.map((part, i) =>
        part.type === 'thinking'
          ? <ThinkingBlock key={i} content={part.content} />
          : <MarkdownContent key={i} content={part.content} />
      )}
    </>
  );
}

function MarkdownContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        // Code blocks
        code({ className, children, ...props }) {
          const isInline = !className;
          if (isInline) {
            return (
              <code className="bg-[var(--color-bg)] px-1.5 py-0.5 rounded text-[13px] font-mono text-[var(--color-accent)]" {...props}>
                {children}
              </code>
            );
          }
          return (
            <code className={`block bg-[var(--color-bg)] p-3 rounded-lg text-[13px] font-mono overflow-x-auto my-2 leading-relaxed ${className || ''}`} {...props}>
              {children}
            </code>
          );
        },
        pre({ children }) {
          return <pre className="my-2">{children}</pre>;
        },
        // Headings
        h1: ({ children }) => <h1 className="text-lg font-bold mt-4 mb-2">{children}</h1>,
        h2: ({ children }) => <h2 className="text-base font-bold mt-3 mb-1.5">{children}</h2>,
        h3: ({ children }) => <h3 className="text-sm font-bold mt-2 mb-1">{children}</h3>,
        // Paragraphs
        p: ({ children }) => <p className="my-1.5 leading-relaxed">{children}</p>,
        // Lists
        ul: ({ children }) => <ul className="list-disc pl-5 my-1.5 space-y-0.5">{children}</ul>,
        ol: ({ children }) => <ol className="list-decimal pl-5 my-1.5 space-y-0.5">{children}</ol>,
        li: ({ children }) => <li className="leading-relaxed">{children}</li>,
        // Links
        a: ({ href, children }) => (
          <a href={href} target="_blank" rel="noopener noreferrer" className="text-[var(--color-accent)] underline hover:opacity-80">
            {children}
          </a>
        ),
        // Bold / Italic
        strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
        em: ({ children }) => <em className="italic">{children}</em>,
        // Blockquote
        blockquote: ({ children }) => (
          <blockquote className="border-l-3 border-[var(--color-accent)] pl-3 my-2 text-[var(--color-text2)] italic">
            {children}
          </blockquote>
        ),
        // Table
        table: ({ children }) => (
          <div className="overflow-x-auto my-2">
            <table className="w-full text-sm border-collapse">{children}</table>
          </div>
        ),
        thead: ({ children }) => <thead className="bg-[var(--color-surface2)]">{children}</thead>,
        th: ({ children }) => <th className="text-left px-2 py-1 border border-[var(--color-border)] text-xs font-semibold">{children}</th>,
        td: ({ children }) => <td className="px-2 py-1 border border-[var(--color-border)]">{children}</td>,
        // Horizontal rule
        hr: () => <hr className="border-[var(--color-border)] my-3" />,
      }}
    >
      {content}
    </ReactMarkdown>
  );
}
