package routes

type Route []*Path

func (r Route) Copy() Route {
	newRoute := make(Route, len(r))
	for i, path := range r {
		newRoute[i] = path.Copy()
	}
	return newRoute
}
