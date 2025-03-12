package tests

import (
	"testing"

	"education/internal/handlers"
)

func TestFindVerifiedParticipant_Success(t *testing.T) {
	// Используем данные, определённые в global.go
	faculty := "Факультет Информатики"
	group := "AA-25-07"
	pass := "ST-456"

	vp, found := handlers.FindVerifiedParticipant(faculty, group, pass)
	if !found {
		t.Fatalf("Expected to find verified participant, but not found")
	}
	if vp.FIO != "Иван Иванов" {
		t.Errorf("Expected FIO 'Иван Иванов', got '%s'", vp.FIO)
	}
}

func TestFindVerifiedParticipant_Fail(t *testing.T) {
	faculty := "Факультет Информатики"
	group := "AA-25-07"
	pass := "WRONG"
	_, found := handlers.FindVerifiedParticipant(faculty, group, pass)
	if found {
		t.Errorf("Expected not to find verified participant with wrong pass")
	}
}
