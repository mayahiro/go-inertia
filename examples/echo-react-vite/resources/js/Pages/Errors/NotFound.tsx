import { Link } from '@inertiajs/react'
import Layout from '../Layout'

export default function NotFound() {
  return (
    <Layout>
      <section className="panel">
        <div className="panel-body not-found">
          <p className="eyebrow">404</p>
          <h1>Page not found</h1>
          <p className="muted">The requested page does not exist in this example application.</p>
          <Link className="button button-link" href="/">
            Dashboard
          </Link>
        </div>
      </section>
    </Layout>
  )
}
