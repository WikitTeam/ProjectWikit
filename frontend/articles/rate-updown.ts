import { ModuleRateResponse } from '../api/rate'
import { castVote, upDownVote } from './rate-notice'

function updateRating(element: HTMLElement, votesData: ModuleRateResponse) {
  let rating = String(votesData.rating)
  if (votesData.rating >= 0) {
    rating = '+' + rating
  }
  element.innerText = rating
}

export function makeUpDownRateModule(node: HTMLElement) {
  if ((node as any)._rate) {
    return
  }
  ;(node as any)._rate = true

  const pageId = node.dataset.pageId!

  const number: HTMLElement = node.querySelector('.number')!
  const rateup: HTMLElement = node.querySelector('.rateup a')!
  const ratedown: HTMLElement = node.querySelector('.ratedown a')!
  const cancel: HTMLElement = node.querySelector('.cancel a')!

  const callback = function (votesData: ModuleRateResponse) {
    updateRating(number, votesData)
    window.postMessage({ type: 'rate_updated' })
  }

  const click = (vote: number | null) => (e: MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    castVote(pageId, vote, upDownVote, callback)
  }

  rateup.addEventListener('click', click(1))
  ratedown.addEventListener('click', click(-1))
  cancel.addEventListener('click', click(null))
}
