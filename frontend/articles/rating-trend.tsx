import * as React from 'react'
import { useMemo, useRef, useState } from 'react'
import styled from 'styled-components'
import { t } from '~util/i18n'
import formatDate from '../util/date-format'

interface Props {
  points: Array<[number, number]>
  stars: boolean
}

const W = 600
const H = 120
const PAD = { top: 8, right: 44, bottom: 20, left: 4 }

const Styles = styled.figure`
  margin: 12px 0 4px;
  max-width: 600px;
  color: inherit;

  figcaption {
    font-size: 12px;
    opacity: 0.75;
    margin-bottom: 4px;
  }
  .w-trend-plot {
    position: relative;
  }
  svg {
    display: block;
    width: 100%;
    height: auto;
    overflow: visible;
  }
  .w-trend-axis {
    font-size: 11px;
    fill: currentColor;
    opacity: 0.6;
  }
  .w-trend-tip {
    position: absolute;
    top: 0;
    transform: translateX(-50%);
    padding: 2px 6px;
    font-size: 12px;
    white-space: nowrap;
    background: Canvas;
    color: CanvasText;
    border: 1px solid currentColor;
    pointer-events: none;
  }
  .w-trend-tip strong {
    margin-right: 6px;
  }
`

const curve = (pts: Array<[number, number]>) => {
  const p = pts.filter((q, i) => i === pts.length - 1 || pts[i + 1][0] > q[0])
  if (p.length < 2) return `M${p[0][0]},${p[0][1]}`
  const n = p.length
  const d: number[] = []
  for (let i = 0; i < n - 1; i++) d.push((p[i + 1][1] - p[i][1]) / (p[i + 1][0] - p[i][0]))
  const m: number[] = [d[0]]
  for (let i = 1; i < n - 1; i++) {
    if (d[i - 1] * d[i] <= 0) {
      m.push(0)
      continue
    }
    const h0 = p[i][0] - p[i - 1][0]
    const h1 = p[i + 1][0] - p[i][0]
    const mid = (d[i - 1] * h1 + d[i] * h0) / (h0 + h1)
    m.push(Math.sign(d[i]) * 2 * Math.min(Math.abs(d[i - 1]), Math.abs(d[i]), Math.abs(mid) / 2))
  }
  m.push(d[n - 2])
  let path = `M${p[0][0]},${p[0][1]}`
  for (let i = 0; i < n - 1; i++) {
    const h = (p[i + 1][0] - p[i][0]) / 3
    path += ` C${p[i][0] + h},${p[i][1] + m[i] * h} ${p[i + 1][0] - h},${p[i + 1][1] - m[i + 1] * h} ${p[i + 1][0]},${p[i + 1][1]}`
  }
  return path
}

const show = (value: number, stars: boolean) => (stars ? value.toFixed(1) : value > 0 ? `+${value}` : String(value))

const RatingTrend: React.FC<Props> = ({ points, stars }) => {
  const [hover, setHover] = useState<number | null>(null)
  const box = useRef<SVGSVGElement>(null)

  const geometry = useMemo(() => {
    if (points.length < 2) return null
    const now = Date.now()
    const series = [...points, [Math.max(now, points[points.length - 1][0]), points[points.length - 1][1]] as [number, number]]
    const t0 = series[0][0]
    const t1 = series[series.length - 1][0]
    const values = series.map(p => p[1])
    let lo = stars ? 0 : Math.min(0, ...values)
    let hi = stars ? 5 : Math.max(0, ...values)
    if (hi === lo) hi = lo + 1
    const x = (ts: number) => PAD.left + ((ts - t0) / Math.max(t1 - t0, 1)) * (W - PAD.left - PAD.right)
    const y = (v: number) => PAD.top + (1 - (v - lo) / (hi - lo)) * (H - PAD.top - PAD.bottom)
    const path = curve(series.map(([ts, v]) => [x(ts), y(v)] as [number, number]))
    return { series, x, y, lo, hi, path, t0, t1 }
  }, [points, stars])

  if (!geometry) return null
  const { series, x, y, lo, hi, path, t0, t1 } = geometry
  const current = series[series.length - 1][1]

  const onMove = (e: React.MouseEvent<SVGSVGElement>) => {
    const rect = box.current!.getBoundingClientRect()
    const at = ((e.clientX - rect.left) / rect.width) * W
    let found = 0
    for (let i = 0; i < series.length; i++) {
      if (x(series[i][0]) <= at) found = i
    }
    setHover(found)
  }

  return (
    <Styles className="w-rating-trend">
      <figcaption>{t('articles.rating.trend', { value: show(current, stars) })}</figcaption>
      <div className="w-trend-plot">
        <svg
          ref={box}
          viewBox={`0 0 ${W} ${H}`}
          role="img"
          aria-label={t('articles.rating.trend', { value: show(current, stars) })}
          onMouseMove={onMove}
          onMouseLeave={() => setHover(null)}
        >
          {!stars && <line x1={PAD.left} x2={W - PAD.right} y1={y(0)} y2={y(0)} stroke="currentColor" strokeOpacity={0.25} strokeDasharray="3 3" />}
          <path d={path} fill="none" stroke="currentColor" strokeWidth={2} strokeLinejoin="round" />
          <text className="w-trend-axis" x={W - PAD.right + 6} y={y(hi) + 4}>
            {show(hi, stars)}
          </text>
          <text className="w-trend-axis" x={W - PAD.right + 6} y={y(lo) + 4}>
            {show(lo, stars)}
          </text>
          <text className="w-trend-axis" x={PAD.left} y={H - 4}>
            {formatDate(new Date(t0))}
          </text>
          <text className="w-trend-axis" x={W - PAD.right} y={H - 4} textAnchor="end">
            {formatDate(new Date(t1))}
          </text>
          {hover !== null && (
            <>
              <line x1={x(series[hover][0])} x2={x(series[hover][0])} y1={PAD.top} y2={H - PAD.bottom} stroke="currentColor" strokeOpacity={0.35} />
              <circle cx={x(series[hover][0])} cy={y(series[hover][1])} r={4} fill="currentColor" stroke="Canvas" strokeWidth={2} />
            </>
          )}
        </svg>
        {hover !== null && (
          <div className="w-trend-tip" style={{ left: `${(x(series[hover][0]) / W) * 100}%` }}>
            <strong>{show(series[hover][1], stars)}</strong> <span>{formatDate(new Date(series[hover][0]))}</span>
          </div>
        )}
      </div>
    </Styles>
  )
}

export default RatingTrend
