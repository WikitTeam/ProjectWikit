import { t } from '~util/i18n'
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
        { title: t('user-actions.cancel'), onClick: onClose },
        { title: t('user-actions.confirm'), onClick: onOk, type: 'danger' },
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
      if (label) label.textContent = currentBlocked ? t('user-actions.unblock') : t('user-actions.block')
    } catch (err: any) {
      showErrorModal(err?.error || t('user-actions.failed'))
    }
  }

  blockLink.addEventListener('click', e => {
    e.preventDefault()
    e.stopPropagation()
    const text = currentBlocked
      ? t('user-actions.confirm-unblock')
      : t('user-actions.confirm-block')
    showConfirmModal(text, runToggle)
  })
}
