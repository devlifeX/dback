package notify

// HostNamer resolves profile and task IDs to human-readable names for notifications.
type HostNamer interface {
	HostName(profileID string) string
	TaskName(taskID string) string
}

type idNamer struct{}

func (idNamer) HostName(id string) string { return id }
func (idNamer) TaskName(id string) string { return id }

func resolveNamer(n HostNamer) HostNamer {
	if n != nil {
		return n
	}
	return idNamer{}
}
