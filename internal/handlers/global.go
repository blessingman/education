package handlers

const (
	StateWaitingForFaculty  = "waiting_for_faculty"
	StateWaitingForGroup    = "waiting_for_group"
	StateWaitingForPass     = "waiting_for_pass"
	StateWaitingForPassword = "waiting_for_password"

	LoginStateWaitingForRegCode  = "login_waiting_for_regcode"
	LoginStateWaitingForPassword = "login_waiting_for_password"
)

// loginData хранит временные данные логина
type loginData struct {
	RegCode string
	MsgIDs  []int
}

// tempUserData – временные данные при регистрации
type tempUserData struct {
	Faculty     string
	Group       string
	FoundUserID int64
}

// userStates/… – состояния
var (
	userStates       = make(map[int64]string)
	userTempDataMap  = make(map[int64]*tempUserData)
	loginStates      = make(map[int64]string)
	loginTempDataMap = make(map[int64]*loginData)
)
