import type { PageProps } from '@inertiajs/core'

export type AppProps = {
  name: string
}

export type ExampleFlashData = {
  success?: string
}

export type User = {
  id: number
  name: string
  email: string
}

export interface ExamplePageProps extends PageProps {
  app: AppProps
}

declare module '@inertiajs/core' {
  export interface InertiaConfig {
    errorValueType: string
    flashDataType: ExampleFlashData | undefined
    sharedPageProps: ExamplePageProps
  }
}
