import { Link, usePage } from '@inertiajs/react'
import type { PropsWithChildren } from 'react'
import type { ExamplePageProps } from '../types'

export default function Layout({ children }: PropsWithChildren) {
  const { app } = usePage<ExamplePageProps>().props

  return (
    <div className="layout">
      <header className="topbar">
        <div className="brand">{app.name}</div>
        <nav className="nav">
          <Link href="/">Dashboard</Link>
          <Link href="/users">Users</Link>
        </nav>
      </header>
      <main className="main">{children}</main>
    </div>
  )
}
