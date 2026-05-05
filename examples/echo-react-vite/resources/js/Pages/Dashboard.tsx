import Layout from './Layout'

type DashboardProps = {
  stats: {
    users: number
    version: string
  }
}

export default function Dashboard({ stats }: DashboardProps) {
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
      </div>
    </Layout>
  )
}
