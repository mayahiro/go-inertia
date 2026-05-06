import { Deferred } from '@inertiajs/react'
import Layout from './Layout'

type DashboardProps = {
  stats: {
    users: number
    version: string
  }
  serverTime?: string
}

export default function Dashboard({ stats, serverTime }: DashboardProps) {
  return (
    <Layout>
      <div className="grid">
        <div className="stat">
          <span>Users</span>
          <strong>{stats.users}</strong>
        </div>
        <div className="stat">
          <span>Library</span>
          <strong>{stats.version}</strong>
        </div>
        <Deferred data="serverTime" fallback={<div className="stat"><span>Server time</span><strong>Loading</strong></div>}>
          <div className="stat">
            <span>Server time</span>
            <strong>{serverTime}</strong>
          </div>
        </Deferred>
      </div>
    </Layout>
  )
}
