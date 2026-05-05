import { router, usePage } from '@inertiajs/react'
import type { FormEvent } from 'react'
import type { ExamplePageProps, User } from '../../types'
import Layout from '../Layout'

type UsersIndexProps = {
  users: User[]
}

export default function UsersIndex({ users }: UsersIndexProps) {
  const { errors, flash } = usePage<ExamplePageProps>().props

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    router.post('/users', new FormData(event.currentTarget))
    event.currentTarget.reset()
  }

  return (
    <Layout>
      {flash?.success && <div className="flash">{flash.success}</div>}
      <section className="panel">
        <div className="panel-body">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
              </tr>
            </thead>
            <tbody>
              {users.map(user => (
                <tr key={user.id}>
                  <td>{user.name}</td>
                  <td>{user.email}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <form className="form" onSubmit={submit}>
            <label className="field">
              <span>Name</span>
              <input className="input" name="name" />
              {errors.name && <span className="error">{errors.name}</span>}
            </label>
            <label className="field">
              <span>Email</span>
              <input className="input" name="email" type="email" />
              {errors.email && <span className="error">{errors.email}</span>}
            </label>
            <button className="button" type="submit">Create</button>
          </form>
        </div>
      </section>
    </Layout>
  )
}
