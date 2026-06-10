import { callModule } from '../api/modules'
import { APIError } from '../util/fetch-util'

interface NewPageCheckResponse {
  url: string
}

interface NewPageConfig {
  category: string
}

export function makeNewPageModule(node: HTMLElement) {
  const form = node.querySelector('form')
  if (!form) return

  const config: NewPageConfig = JSON.parse(node.dataset.config || '{}')
  const errorDiv = node.querySelector('.new-page-error') as HTMLElement | null

  form.addEventListener('submit', async e => {
    e.preventDefault()
    e.stopPropagation()

    const input = form.querySelector('input[name="new_fullname"]') as HTMLInputElement | null
    if (!input) return

    const newFullname = input.value.trim()
    if (!newFullname) return

    const submitBtn = form.querySelector('input[type="submit"]') as HTMLInputElement | null
    if (submitBtn) submitBtn.disabled = true
    if (errorDiv) errorDiv.style.display = 'none'

    try {
      const response = await callModule<NewPageCheckResponse>({
        module: 'newpage',
        method: 'check',
        params: {
          new_fullname: newFullname,
          category: config.category || '',
        },
      })
      window.location.href = response.url
    } catch (err) {
      if (errorDiv) {
        errorDiv.textContent = err instanceof APIError ? err.error : '创建页面时出错'
        errorDiv.style.display = 'block'
      }
      if (submitBtn) submitBtn.disabled = false
    }
  })
}
