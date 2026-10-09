import * as React from 'react'
import { useEffect, useId, useRef, useState } from 'react'
import { suggestSearch, SuggestField, SuggestItem } from '../api/search-module'

interface Props {
  field: SuggestField
  value: string
  placeholder: string
  onChange: (value: string) => void
}

function lastToken(field: SuggestField, value: string): string {
  if (field !== 'tags') return value.trim()
  const parts = value.split(/[\s,]+/)
  return (parts[parts.length - 1] || '').replace(/^-/, '')
}

function withChoice(field: SuggestField, value: string, choice: string): string {
  if (field !== 'tags') return choice
  const match = value.match(/^([\s\S]*?)(-?)([^\s,]*)$/)
  if (!match) return choice + ' '
  return match[1] + match[2] + choice + ' '
}

const SearchSuggestInput: React.FC<Props> = ({ field, value, placeholder, onChange }) => {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<SuggestItem[]>([])
  const [active, setActive] = useState(-1)
  const seq = useRef(0)
  const listId = useId()

  useEffect(() => {
    if (!open) return
    const my = ++seq.current
    const timer = setTimeout(async () => {
      try {
        const found = await suggestSearch(field, lastToken(field, value))
        if (my === seq.current) {
          setItems(found)
          setActive(-1)
        }
      } catch {
        if (my === seq.current) setItems([])
      }
    }, 150)
    return () => clearTimeout(timer)
  }, [open, field, value])

  const choose = (item: SuggestItem) => {
    onChange(withChoice(field, value, item.value))
    setOpen(field === 'tags')
    setActive(-1)
  }

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape') {
      setOpen(false)
      return
    }
    if (!open || !items.length) {
      if (e.key === 'ArrowDown') setOpen(true)
      return
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActive(i => (i + 1) % items.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActive(i => (i <= 0 ? items.length - 1 : i - 1))
    } else if (e.key === 'Enter' && active >= 0) {
      e.preventDefault()
      choose(items[active])
    }
  }

  const shown = open && items.length > 0

  return (
    <div className="w-search-field">
      <input
        className="w-search-filter"
        type="text"
        value={value}
        placeholder={placeholder}
        role="combobox"
        aria-expanded={shown}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={shown && active >= 0 ? `${listId}-${active}` : undefined}
        autoComplete="off"
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onChange={e => {
          onChange(e.target.value)
          setOpen(true)
        }}
        onKeyDown={onKeyDown}
      />
      {shown && (
        <ul className="w-search-suggest" id={listId} role="listbox">
          {items.map((item, i) => (
            <li
              key={item.value}
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === active}
              className={i === active ? 'active' : undefined}
              onMouseDown={e => {
                e.preventDefault()
                choose(item)
              }}
              onMouseEnter={() => setActive(i)}
            >
              {item.label}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default SearchSuggestInput
