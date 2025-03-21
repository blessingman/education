package handlers

import "sync"

const (
	StateWaitingForRole     = "waiting_for_role" // Новое состояние для выбора роли
	StateWaitingForFaculty  = "waiting_for_faculty"
	StateWaitingForGroup    = "waiting_for_group"
	StateWaitingForPass     = "waiting_for_pass"
	StateWaitingForPassword = "waiting_for_password"

	// Состояния для преподавателей
	StateTeacherWaitingForPass     = "teacher_waiting_for_pass"
	StateTeacherWaitingForPassword = "teacher_waiting_for_password"

	LoginStateWaitingForRegCode  = "login_waiting_for_regcode"
	LoginStateWaitingForPassword = "login_waiting_for_password"
)

// loginData хранит временные данные логина
// tempUserData – временные данные при регистрации
type tempUserData struct {
	Faculty     string
	Group       string
	FoundUserID int64
	Role        string // Новое поле для хранения выбранной роли
	MsgIDs      []int  // Список MessageID для удаления сообщений
}

// loginData хранит временные данные логина
type loginData struct {
	RegCode string
	MsgIDs  []int // Список MessageID для удаления сообщений
}

// StateManager manages all the state data with mutex protection
type StateManager struct {
	mu            sync.RWMutex
	userStates    map[int64]string
	tempUserData  map[int64]*tempUserData
	loginStates   map[int64]string
	loginTempData map[int64]*loginData
}

// NewStateManager creates a new initialized state manager
func NewStateManager() *StateManager {
	return &StateManager{
		userStates:    make(map[int64]string),
		tempUserData:  make(map[int64]*tempUserData),
		loginStates:   make(map[int64]string),
		loginTempData: make(map[int64]*loginData),
	}
}

// userStates и прочие переменные для хранения состояний
var (
	userStates       = make(map[int64]string)
	userTempDataMap  = make(map[int64]*tempUserData)
	loginStates      = make(map[int64]string)
	loginTempDataMap = make(map[int64]*loginData)
)
