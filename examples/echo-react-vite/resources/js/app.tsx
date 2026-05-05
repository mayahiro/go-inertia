import { createInertiaApp } from '@inertiajs/react'
import type { ComponentType } from 'react'
import { createRoot } from 'react-dom/client'
import './style.css'

createInertiaApp({
  resolve: name => {
    const pages = import.meta.glob<{ default: ComponentType<Record<string, unknown>> }>('./Pages/**/*.tsx', { eager: true })
    return pages[`./Pages/${name}.tsx`]
  },
  setup({ el, App, props }) {
    if (!el) {
      return
    }
    createRoot(el).render(<App {...props} />)
  },
})
