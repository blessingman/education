package handlers

import (
	"education/internal/models"
	"sync"
	"time"
)

// Cache хранит данные, которые редко меняются (факультеты и группы)
type Cache struct {
	mu        sync.RWMutex
	Faculties []string
	Groups    map[string][]string // ключ – название факультета, значение – список групп
}

var globalCache = &Cache{
	Groups: make(map[string][]string),
}

// CacheEntry хранит расписание и время его обновления.
type CacheEntry struct {
	Schedules []models.Schedule
	UpdatedAt time.Time
}

// ScheduleCache – структура для кеширования расписания по ключам (например, группа или регистрационный код преподавателя)
var ScheduleCache = struct {
	sync.RWMutex
	entries map[string]CacheEntry
}{entries: make(map[string]CacheEntry)}

// CacheTTL задаёт время жизни кеша, например, 5 минут.
var CacheTTL = 5 * time.Minute

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

// GetCachedSchedule пытается получить расписание из кеша по ключу.
// Ключ может быть, например, user.Group для студентов или user.RegistrationCode для преподавателей.
func GetCachedSchedule(key string) (CacheEntry, bool) {
	ScheduleCache.RLock()
	defer ScheduleCache.RUnlock()

	entry, exists := ScheduleCache.entries[key]
	if !exists || time.Since(entry.UpdatedAt) > CacheTTL {
		return CacheEntry{}, false
	}
	return entry, true
}

// SetCachedSchedule сохраняет расписание в кеш по заданному ключу.
func SetCachedSchedule(key string, schedules []models.Schedule) {
	ScheduleCache.Lock()
	defer ScheduleCache.Unlock()

	ScheduleCache.entries[key] = CacheEntry{
		Schedules: schedules,
		UpdatedAt: time.Now(),
	}
}

func InvalidateScheduleCache(key string) {
	ScheduleCache.Lock()
	defer ScheduleCache.Unlock()
	delete(ScheduleCache.entries, key)
}
