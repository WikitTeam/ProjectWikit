import * as React from 'react'
import { t } from '~util/i18n'

interface Props {
  id: string
  values?: Record<string, string | number>
  children?: Record<string, React.ReactNode>
}

const SLOT = /\{(\w+)\}/g

// A sentence that wraps part of itself in an element cannot be two keys: the
// order of the parts changes between languages, and a translator handed halves
// cannot see the sentence. The catalog keeps the whole sentence with {name}
// slots, and the slot is filled with an element here.
const Trans: React.FC<Props> = ({ id, values, children }) => {
  const text = t(id, values)
  const parts: React.ReactNode[] = []
  let last = 0
  for (const match of text.matchAll(SLOT)) {
    const name = match[1]
    const node = children?.[name]
    if (node === undefined) {
      continue
    }
    parts.push(text.slice(last, match.index))
    parts.push(<React.Fragment key={parts.length}>{node}</React.Fragment>)
    last = (match.index ?? 0) + match[0].length
  }
  parts.push(text.slice(last))
  return <>{parts}</>
}

export default Trans
