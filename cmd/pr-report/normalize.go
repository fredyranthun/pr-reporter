package main

func commentIndicator(comments, threads *int) *bool {
	if comments != nil && *comments > 0 || threads != nil && *threads > 0 {
		return ptr(true)
	}
	if comments != nil && threads != nil {
		return ptr(false)
	}
	return nil
}
