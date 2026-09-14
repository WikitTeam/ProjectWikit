import * as React from 'react'
import * as Styled from './OwnList.styles'

interface Props {
  page: number
  pages: number
  onPage: (to: number) => void
}

function pagesAround(current: number, total: number): Array<number | null> {
  const wanted = new Set<number>([1, total, current])
  for (let d = 1; d <= 2; d++) {
    if (current - d >= 1) wanted.add(current - d)
    if (current + d <= total) wanted.add(current + d)
  }
  const sorted = Array.from(wanted).sort((a, b) => a - b)
  const out: Array<number | null> = []
  sorted.forEach((page, i) => {
    if (i > 0 && page - sorted[i - 1] > 1) out.push(null)
    out.push(page)
  })
  return out
}

const Pager: React.FC<Props> = ({ page, pages, onPage }) => {
  if (pages <= 1) return null
  return (
    <Styled.Pager>
      {pagesAround(page, pages).map((one, i) =>
        one === null ? (
          <Styled.PagerDots key={i}>...</Styled.PagerDots>
        ) : (
          <Styled.PagerStep key={i} current={one === page} disabled={one === page} onClick={() => onPage(one)}>
            {one}
          </Styled.PagerStep>
        ),
      )}
    </Styled.Pager>
  )
}

export default Pager
