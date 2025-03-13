package handlers

import "strings"

// FindVerifiedParticipant экспортированная функция для поиска верифицированного участника.
func FindVerifiedParticipant(faculty, group, pass string) (*VerifiedParticipant, bool) {
	pass = strings.TrimSpace(pass) // убираем пробелы вокруг кода
	for _, vp := range verifiedParticipants {
		if vp.Faculty == faculty && vp.Group == group && vp.Pass == pass {
			return &vp, true
		}
	}
	return nil, false
}
