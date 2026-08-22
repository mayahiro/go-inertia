import { createInertiaApp } from '@inertiajs/react'
import type { ComponentType } from 'react'
import type { ExamplePageProps } from './types'
import './style.css'

createInertiaApp<ExamplePageProps>({
  id: 'inertia-app',
  dev: import.meta.env.DEV,
  serverHead: true,
  strictMode: true,
  resolve: name => {
    const pages = import.meta.glob<{ default: ComponentType<Record<string, unknown>> }>('./Pages/**/*.tsx', { eager: true })
    return pages[`./Pages/${name}.tsx`]
  },
})
