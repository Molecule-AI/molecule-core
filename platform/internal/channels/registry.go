package channels

// Registry of all available channel adapters.
// To add a new platform: implement ChannelAdapter, register here.
var adapters = map[string]ChannelAdapter{
	"telegram": &TelegramAdapter{},
}

// GetAdapter returns the adapter for a channel type.
func GetAdapter(channelType string) (ChannelAdapter, bool) {
	a, ok := adapters[channelType]
	return a, ok
}

// ListAdapters returns metadata about all available adapters.
func ListAdapters() []map[string]string {
	result := make([]map[string]string, 0, len(adapters))
	for _, a := range adapters {
		result = append(result, map[string]string{
			"type":         a.Type(),
			"display_name": a.DisplayName(),
		})
	}
	return result
}
