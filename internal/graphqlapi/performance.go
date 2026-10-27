package graphqlapi

func (r *Resolver) developerModeEnabled() bool {
	snapshot := r.RuntimeSettings
	if r.SettingsEditor != nil {
		snapshot = r.SettingsEditor.Snapshot()
	}
	return snapshot != nil && snapshot.System.DeveloperMode
}
