package handlers

import "fmt"

// findVerifiedParticipant ищет участника по факультету, группе и введённому коду.
func FindVerifiedParticipant(faculty, group, pass string) (*VerifiedParticipant, bool) {
	for _, vp := range verifiedParticipants {
		if vp.Faculty == faculty && vp.Group == group && vp.Pass == pass {
			return &vp, true
		}
	}
	return nil, false
}

// debugPrint – вспомогательная функция для отладки.
func debugPrint(msg string) {
	fmt.Println(msg)
}
