import * as React from 'react'
import WikidotModal, {
  addUnmanagedModal,
  removeUnmanagedModal,
  showErrorModal,
} from '~util/wikidot-modal'
import { blockUser, unblockUser } from '~api/messages'

interface ContainerConfig {
  userId: number
  isBlocked: boolean
}

function showConfirmModal(text: string, onConfirm: () => void) {
  let uuid: string | null = null

  const onClose = () => {
    if (uuid) removeUnmanagedModal(uuid)
  }

  const onOk = () => {
    onClose()
    onConfirm()
  }

  uuid = addUnmanagedModal(
    <WikidotModal
      isError
      buttons={[
        { title: '取消', onClick: onClose },
        { title: '确定', onClick: onOk, type: 'danger' },
      ]}
    >
      <p>{text}</p>
    </WikidotModal>,
  )
}

export function attachUserActions() {
  const container = document.getElementById('user-actions-container')
  if (!container) return

  let cfg: ContainerConfig
  try {
    cfg = JSON.parse(container.dataset.config || '{}')
  } catch {
    return
  }

  const blockLink = container.querySelector<HTMLAnchorElement>('[data-action="toggle-block"]')
  if (!blockLink) return

  let currentBlocked = cfg.isBlocked

  const runToggle = async () => {
    try {
      if (currentBlocked) {
        await unblockUser(cfg.userId)
        currentBlocked = false
      } else {
        await blockUser(cfg.userId)
        currentBlocked = true
      }
      const label = container.querySelector('.block-label')
      if (label) label.textContent = currentBlocked ? '取消拉黑' : '拉黑'
    } catch (err: any) {
      showErrorModal(err?.error || '操作失败')
    }
  }

  blockLink.addEventListener('click', e => {
    e.preventDefault()
    e.stopPropagation()
    const text = currentBlocked
      ? '确定要取消对该用户的拉黑吗？'
      : '拉黑该用户后，对方将无法向你发送私信。确定拉黑吗？'
    showConfirmModal(text, runToggle)
  })
}
