package app

// HostSort returns the saved hosts list sort order.
func (a *App) HostSort() string {
	return a.store.HostSort()
}

// SetHostSort persists the hosts list sort order.
func (a *App) SetHostSort(sort string) error {
	return a.store.SetHostSort(sort)
}
