import { Link } from '@inertiajs/react'
import Layout from '../Layout'

type ErrorProps = {
  status: number
}

export default function ErrorPage({ status }: ErrorProps) {
  return (
    <Layout>
      <section className="panel">
        <div className="panel-body not-found">
          <p className="eyebrow">{status}</p>
          <h1>Request failed</h1>
          <p className="muted">The request could not be completed</p>
          <Link className="button button-link" href="/">
            Dashboard
          </Link>
        </div>
      </section>
    </Layout>
  )
}
