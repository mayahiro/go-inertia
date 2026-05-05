import type { PageProps } from '@inertiajs/core'

export type AppProps = {
  name: string
}

export type User = {
  id: number
  name: string
  email: string
}

export interface ExamplePageProps extends PageProps {
  app: AppProps
  errors: Record<string, string | undefined>
  flash?: {
    success?: string
  }
}
