package engine

import "slices"

// ParamKeys exposes only the compiled event's required source keys to its loader.
func (s Snapshot) ParamKeys(event string) []string {
	keys := []string{}
	for _, f := range s.events[event].factors {
		if value, ok := f.(interface{ ParamKey() string }); ok && !slices.Contains(keys, value.ParamKey()) {
			keys = append(keys, value.ParamKey())
		}
	}
	slices.Sort(keys)
	return keys
}
