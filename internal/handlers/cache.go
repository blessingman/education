package handlers

import "sync"

// Cache хранит данные, которые редко меняются (факультеты и группы)
type Cache struct {
	mu        sync.RWMutex
	Faculties []string
	Groups    map[string][]string // ключ – название факультета, значение – список групп
}

var globalCache = &Cache{
	Groups: make(map[string][]string),
}

// GetFaculties возвращает список факультетов из кэша
func GetFaculties() []string {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	return globalCache.Faculties
}

// SetFaculties сохраняет список факультетов в кэш
func SetFaculties(fac []string) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.Faculties = fac
}

// GetGroups возвращает список групп для факультета из кэша
func GetGroups(faculty string) []string {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	return globalCache.Groups[faculty]
}

// SetGroups сохраняет список групп для факультета в кэш
func SetGroups(faculty string, groups []string) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	globalCache.Groups[faculty] = groups
}
