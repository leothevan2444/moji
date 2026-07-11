import { createElement, Fragment, ReactNode } from "react";

function inlineMarkdown(value: string, keyPrefix: string): ReactNode[] {
  const pattern = /(\[([^\]]+)\]\(([^)]+)\)|\*\*([^*]+)\*\*|`([^`]+)`)/g;
  const nodes: ReactNode[] = [];
  let cursor = 0;
  let match: RegExpExecArray | null;
  let index = 0;

  while ((match = pattern.exec(value)) !== null) {
    if (match.index > cursor) nodes.push(value.slice(cursor, match.index));
    const key = `${keyPrefix}-${index++}`;
    if (match[2] && match[3]) {
      const external = /^https?:\/\//i.test(match[3]);
      nodes.push(<a key={key} href={match[3]} target={external ? "_blank" : undefined} rel={external ? "noreferrer" : undefined}>{match[2]}</a>);
    } else if (match[4]) {
      nodes.push(<strong key={key}>{match[4]}</strong>);
    } else if (match[5]) {
      nodes.push(<code key={key}>{match[5]}</code>);
    }
    cursor = pattern.lastIndex;
  }
  if (cursor < value.length) nodes.push(value.slice(cursor));
  return nodes;
}

/** A deliberately small, safe Markdown renderer for bundled help content. */
export function MarkdownBlock({ markdown }: { markdown: string }) {
  const nodes: ReactNode[] = [];
  const lines = markdown.replace(/\r\n/g, "\n").split("\n");
  let paragraph: string[] = [];
  let listItems: string[] = [];
  let listType: "ul" | "ol" = "ul";
  let codeLines: string[] = [];
  let inCode = false;

  const flushParagraph = () => {
    if (!paragraph.length) return;
    const value = paragraph.join(" ").trim();
    nodes.push(<p key={`p-${nodes.length}`}>{inlineMarkdown(value, `p-${nodes.length}`)}</p>);
    paragraph = [];
  };
  const flushList = () => {
    if (!listItems.length) return;
    const items = listItems.map((item, index) => <li key={index}>{inlineMarkdown(item, `li-${nodes.length}-${index}`)}</li>);
    nodes.push(listType === "ol" ? <ol key={`ol-${nodes.length}`}>{items}</ol> : <ul key={`ul-${nodes.length}`}>{items}</ul>);
    listItems = [];
  };
  const flushCode = () => {
    if (!codeLines.length) return;
    nodes.push(<pre key={`pre-${nodes.length}`}><code>{codeLines.join("\n")}</code></pre>);
    codeLines = [];
  };
  const flushText = () => { flushParagraph(); flushList(); };

  for (const line of lines) {
    if (line.trim().startsWith("```")) {
      if (inCode) flushCode(); else flushText();
      inCode = !inCode;
      continue;
    }
    if (inCode) { codeLines.push(line); continue; }
    if (!line.trim()) { flushText(); continue; }

    const heading = /^(#{1,6})\s+(.+)$/.exec(line);
    if (heading) {
      flushText();
      const level = Math.min(heading[1].length + 1, 6);
      nodes.push(createElement(`h${level}`, { key: `h-${nodes.length}` }, inlineMarkdown(heading[2], `h-${nodes.length}`)));
      continue;
    }
    if (/^>\s?/.test(line)) {
      flushText();
      const value = line.replace(/^>\s?/, "");
      nodes.push(<blockquote key={`quote-${nodes.length}`}>{inlineMarkdown(value, `quote-${nodes.length}`)}</blockquote>);
      continue;
    }
    const unordered = /^[-*]\s+(.+)$/.exec(line);
    const ordered = /^\d+\.\s+(.+)$/.exec(line);
    if (unordered || ordered) {
      flushParagraph();
      const nextType = ordered ? "ol" : "ul";
      if (listItems.length && listType !== nextType) flushList();
      listType = nextType;
      listItems.push((unordered?.[1] ?? ordered?.[1]) as string);
      continue;
    }
    paragraph.push(line.trim());
  }

  flushText();
  flushCode();
  return <Fragment>{nodes}</Fragment>;
}
