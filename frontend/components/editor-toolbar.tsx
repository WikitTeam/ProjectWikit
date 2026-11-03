import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import styled from 'styled-components'
import { t } from '~util/i18n'
import { attachHovertip, hide } from '../util/hovertip'

export interface EditorTarget {
  value(): string
  selection(): [number, number]
  replace(start: number, end: number, text: string, selectFrom: number, selectTo: number): void
}

type Action =
  | { kind: 'wrap'; before: string; after: string }
  | { kind: 'lines'; prefix: (line: string) => string }
  | { kind: 'block'; open: () => string; close: string }
  | { kind: 'insert'; text: string }
  | { kind: 'redact' }

interface Tool {
  id: string
  icon?: string
  action: Action
}

const heading = (level: number): Action => ({
  kind: 'lines',
  prefix: line => '+'.repeat(level) + ' ' + line.replace(/^\++\s*/, ''),
})

const HEADINGS = [1, 2, 3, 4, 5, 6]

const GROUPS: Tool[][] = [
  [
    { id: 'bold', icon: 'fa-bold', action: { kind: 'wrap', before: '**', after: '**' } },
    { id: 'italic', icon: 'fa-italic', action: { kind: 'wrap', before: '//', after: '//' } },
    { id: 'underline', icon: 'fa-underline', action: { kind: 'wrap', before: '__', after: '__' } },
    { id: 'strikethrough', icon: 'fa-strikethrough', action: { kind: 'wrap', before: '--', after: '--' } },
    { id: 'blackbar', action: { kind: 'redact' } },
  ],
  [
    { id: 'bullist', icon: 'fa-list-ul', action: { kind: 'lines', prefix: line => '* ' + line } },
    { id: 'numlist', icon: 'fa-list-ol', action: { kind: 'lines', prefix: line => '# ' + line } },
    { id: 'quote', icon: 'fa-quote-left', action: { kind: 'lines', prefix: line => '> ' + line } },
  ],
  [
    { id: 'pagelink', icon: 'fa-link', action: { kind: 'wrap', before: '[[[', after: ']]]' } },
    { id: 'image', icon: 'fa-image', action: { kind: 'wrap', before: '[[image ', after: ']]' } },
    { id: 'table', icon: 'fa-table', action: { kind: 'insert', text: '||~  ||~  ||\n||  ||  ||' } },
  ],
  [
    {
      id: 'collapsible',
      icon: 'fa-caret-square-down',
      action: {
        kind: 'block',
        open: () => `[[collapsible show="${t('editor-toolbar.collapsible-show')}" hide="${t('editor-toolbar.collapsible-hide')}"]]`,
        close: '[[/collapsible]]',
      },
    },
    { id: 'code', icon: 'fa-code', action: { kind: 'block', open: () => '[[code]]', close: '[[/code]]' } },
    { id: 'footnote', icon: 'fa-asterisk', action: { kind: 'wrap', before: '[[footnote]]', after: '[[/footnote]]' } },
  ],
]

function lineBounds(value: string, start: number, end: number): [number, number] {
  const from = value.lastIndexOf('\n', start - 1) + 1
  let to = value.indexOf('\n', end > start && value[end - 1] === '\n' ? end - 1 : end)
  if (to < 0) to = value.length
  return [from, to]
}

// Blocks and inserted lines need a line of their own, or the parser reads them
// as part of the paragraph around them.
function onOwnLine(value: string, start: number, end: number, text: string): [string, number] {
  const lead = start > 0 && value[start - 1] !== '\n' ? '\n' : ''
  const trail = end < value.length && value[end] !== '\n' ? '\n' : ''
  return [lead + text + trail, lead.length]
}

interface Change {
  start: number
  end: number
  text: string
  from: number
  to: number
}

export function applyTool(action: Action, value: string, start: number, end: number): Change {
  const selected = value.slice(start, end)
  switch (action.kind) {
    case 'wrap': {
      const text = action.before + selected + action.after
      return { start, end, text, from: action.before.length, to: action.before.length + selected.length }
    }
    case 'lines': {
      const [from, to] = lineBounds(value, start, end)
      const text = value.slice(from, to).split('\n').map(action.prefix).join('\n')
      return { start: from, end: to, text, from: text.length, to: text.length }
    }
    case 'block': {
      const open = action.open()
      const [text, lead] = onOwnLine(value, start, end, open + '\n' + selected + '\n' + action.close)
      const at = lead + open.length + 1
      return { start, end, text, from: at, to: at + selected.length }
    }
    case 'insert': {
      const [text] = onOwnLine(value, start, end, action.text)
      return { start, end, text, from: text.length, to: text.length }
    }
    case 'redact': {
      const text = selected ? Array.from(selected, c => (c === '\n' ? c : '█')).join('') : '█'
      return { start, end, text, from: text.length, to: text.length }
    }
  }
}

// The id selector is what outranks both the old sprite rules in wikidot-base.css
// and whatever a site theme does to every button on the page.
const Styles = styled.div`
  &#wd-editor-toolbar-panel {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 2px;
    height: auto;
    margin: 0 0 6px;
    padding: 0;
    color: inherit;
  }
  &#wd-editor-toolbar-panel .w-tool {
    all: unset;
    box-sizing: border-box;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 2px;
    color: inherit;
    opacity: 0.78;
    font-size: 13px;
    line-height: 1;
    cursor: pointer;
  }
  &#wd-editor-toolbar-panel .w-tool:hover {
    opacity: 1;
    background: color-mix(in srgb, currentColor 10%, transparent);
  }
  &#wd-editor-toolbar-panel .w-tool:active,
  &#wd-editor-toolbar-panel .w-tool[aria-expanded='true'] {
    opacity: 1;
    background: color-mix(in srgb, currentColor 20%, transparent);
  }
  &#wd-editor-toolbar-panel .w-tool:focus-visible {
    outline: 2px solid currentColor;
    outline-offset: -2px;
    opacity: 1;
  }
  &#wd-editor-toolbar-panel .w-tool-heading {
    width: auto;
    padding: 0 6px;
    gap: 4px;
    font-weight: bold;
  }
  &#wd-editor-toolbar-panel .w-tool-heading .fa-caret-down {
    font-size: 10px;
  }
  &#wd-editor-toolbar-panel .w-tool-bar {
    display: block;
    width: 13px;
    height: 9px;
    background: currentColor;
  }
  &#wd-editor-toolbar-panel .w-tool-rule {
    width: 1px;
    height: 16px;
    margin: 0 5px;
    background: currentColor;
    opacity: 0.25;
  }
  &#wd-editor-toolbar-panel .w-tool-menu-wrap {
    position: relative;
    height: auto;
  }
  &#wd-editor-toolbar-panel .w-tool-menu {
    position: absolute;
    top: 100%;
    left: 0;
    z-index: 40;
    height: auto;
    min-width: 9em;
    margin-top: 2px;
    padding: 3px 0;
    border: 1px solid color-mix(in srgb, currentColor 30%, transparent);
    background: Canvas;
    color: CanvasText;
  }
  &#wd-editor-toolbar-panel .w-tool-menu .w-tool {
    display: flex;
    justify-content: flex-start;
    width: 100%;
    height: auto;
    padding: 3px 10px;
    border-radius: 0;
    opacity: 1;
    font-weight: bold;
    line-height: 1.25;
  }
`

const MENU_SIZES = ['170%', '145%', '125%', '110%', '100%', '90%']

function useHovertip(text: string) {
  const ref = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (!ref.current) return
    attachHovertip(ref.current, text)
    ref.current.style.cursor = 'pointer'
  }, [])
  return ref
}

const ToolButton: React.FC<{ tool: Tool; onUse: (action: Action) => void }> = ({ tool, onUse }) => {
  const label = t(`editor-toolbar.${tool.id}`)
  const ref = useHovertip(label)
  return (
    <button
      ref={ref}
      type="button"
      className={`w-tool w-tool-${tool.id}`}
      aria-label={label}
      onMouseDown={e => e.preventDefault()}
      onClick={() => onUse(tool.action)}
    >
      {tool.icon ? <i className={`fas ${tool.icon}`} aria-hidden="true" /> : <span className="w-tool-bar" aria-hidden="true" />}
    </button>
  )
}

const HeadingMenu: React.FC<{ onUse: (action: Action) => void }> = ({ onUse }) => {
  const [open, setOpen] = useState(false)
  const wrap = useRef<HTMLSpanElement>(null)
  const label = t('editor-toolbar.heading')
  const ref = useHovertip(label)

  useEffect(() => {
    if (!open) return
    const close = (e: Event) => {
      if (e instanceof KeyboardEvent ? e.key === 'Escape' : !wrap.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', close)
    document.addEventListener('keydown', close)
    return () => {
      document.removeEventListener('mousedown', close)
      document.removeEventListener('keydown', close)
    }
  }, [open])

  return (
    <span className="w-tool-menu-wrap" ref={wrap}>
      <button
        ref={ref}
        type="button"
        className="w-tool w-tool-heading"
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        onMouseDown={e => e.preventDefault()}
        onClick={() => {
          // The tip would sit on top of the first item in the menu.
          document.querySelectorAll<HTMLElement>('.hovertip.w-plain-hovertip').forEach(hide)
          setOpen(!open)
        }}
      >
        <i className="fas fa-heading" aria-hidden="true" />
        <i className="fas fa-caret-down" aria-hidden="true" />
      </button>
      {open && (
        <span className="w-tool-menu" role="menu" style={{ display: 'block' }}>
          {HEADINGS.map((level, i) => (
            <button
              key={level}
              type="button"
              role="menuitem"
              className="w-tool"
              style={{ fontSize: MENU_SIZES[i] }}
              onMouseDown={e => e.preventDefault()}
              onClick={() => {
                setOpen(false)
                onUse(heading(level))
              }}
            >
              {t('editor-toolbar.heading-level', { level })}
            </button>
          ))}
        </span>
      )}
    </span>
  )
}

const EditorToolbar: React.FC<{ target: () => EditorTarget | null; disabled?: boolean }> = ({ target, disabled }) => {
  const onUse = (action: Action) => {
    const editor = target()
    if (!editor || disabled) return
    const [start, end] = editor.selection()
    const change = applyTool(action, editor.value(), start, end)
    editor.replace(change.start, change.end, change.text, change.start + change.from, change.start + change.to)
  }

  return (
    <Styles id="wd-editor-toolbar-panel" className="wd-editor-toolbar-panel" role="toolbar" aria-label={t('editor-toolbar.label')}>
      <HeadingMenu onUse={onUse} />
      {GROUPS.map((group, i) => (
        <React.Fragment key={i}>
          <span className="w-tool-rule" aria-hidden="true" />
          {group.map(tool => (
            <ToolButton key={tool.id} tool={tool} onUse={onUse} />
          ))}
        </React.Fragment>
      ))}
    </Styles>
  )
}

export default EditorToolbar

export function textareaTarget(node: HTMLTextAreaElement | null, setValue: (value: string) => void): EditorTarget | null {
  if (!node) return null
  return {
    value: () => node.value,
    selection: () => [node.selectionStart, node.selectionEnd],
    replace: (start, end, text, selectFrom, selectTo) => {
      setValue(node.value.slice(0, start) + text + node.value.slice(end))
      // The new value lands on the next render, and the caret only sticks after it.
      setTimeout(() => {
        node.focus()
        node.setSelectionRange(selectFrom, selectTo)
      }, 0)
    },
  }
}

export function monacoTarget(instance: any, monaco: any): EditorTarget | null {
  const model = instance?.getModel()
  if (!model || !monaco) return null
  const range = (from: number, to: number) => {
    const a = model.getPositionAt(from)
    const b = model.getPositionAt(to)
    return new monaco.Range(a.lineNumber, a.column, b.lineNumber, b.column)
  }
  return {
    value: () => model.getValue(),
    selection: () => {
      const s = instance.getSelection()
      return [model.getOffsetAt(s.getStartPosition()), model.getOffsetAt(s.getEndPosition())]
    },
    replace: (start, end, text, selectFrom, selectTo) => {
      instance.executeEdits('toolbar', [{ range: range(start, end), text, forceMoveMarkers: true }])
      instance.setSelection(range(selectFrom, selectTo))
      instance.focus()
    },
  }
}
