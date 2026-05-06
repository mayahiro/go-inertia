package inertia

// OnceFunc loads a prop that the Inertia client remembers across visits.
type OnceFunc = PropFunc

// OnceProp marks a page prop as reusable after the client receives it once.
type OnceProp = Prop

// Once returns a prop that is resolved once and then reused by the client.
func Once(fn OnceFunc) OnceProp {
	return newProp(fn).Once()
}
