package handlers

// FindVerifiedParticipant экспортированная функция для поиска верифицированного участника.
func FindVerifiedParticipant(faculty, group, pass string) (*VerifiedParticipant, bool) {
	for _, vp := range verifiedParticipants {
		if vp.Faculty == faculty && vp.Group == group && vp.Pass == pass {
			return &vp, true
		}
	}
	return nil, false
}
