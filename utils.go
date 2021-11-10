package go_boardgame_networking

func byteChanIsClosed(ch <-chan []byte) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}

func contains(items []string, item string) bool {
	for _, it := range items {
		if it == item {
			return true
		}
	}
	return false
}

func copyMap(m map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}
