package handlers

const (
	// Состояния регистрации
	StateWaitingForFaculty  = "waiting_for_faculty"
	StateWaitingForGroup    = "waiting_for_group"
	StateWaitingForPass     = "waiting_for_pass"     // ввод регистрационного кода
	StateWaitingForPassword = "waiting_for_password" // установка нового пароля

	// Состояния входа (логин)
	LoginStateWaitingForRegCode  = "login_waiting_for_regcode"
	LoginStateWaitingForPassword = "login_waiting_for_password"
)

// VerifiedParticipant описывает верифицированного участника.
type VerifiedParticipant struct {
	FIO     string // ФИО
	Faculty string // Факультет
	Group   string // Группа
	Pass    string // Регистрационный код (например, "ST-456")
	Role    string // Роль: "student", "teacher", "admin"
}

// loginData хранит временные данные для входа.
type loginData struct {
	RegCode string
	MsgIDs  []int
}

// Список верифицированных участников (обычно обновляется через административный интерфейс).
var verifiedParticipants = map[string]VerifiedParticipant{
	"Иван Иванов":       {FIO: "Иван Иванов", Faculty: "Факультет Информатики", Group: "AA-25-07", Pass: "ST-456", Role: "student"},
	"Петр Петров":       {FIO: "Петр Петров", Faculty: "Факультет Механики", Group: "BB-10-07", Pass: "TR-345", Role: "teacher"},
	"Светлана Соколова": {FIO: "Светлана Соколова", Faculty: "Факультет Информатики", Group: "AA-25-08", Pass: "ST-457", Role: "student"},
	"Мария Смирнова":    {FIO: "Мария Смирнова", Faculty: "Факультет Физики", Group: "CC-15-01", Pass: "ST-459", Role: "student"},
	"Алексей Козлов":    {FIO: "Алексей Козлов", Faculty: "Факультет Механики", Group: "BB-10-08", Pass: "TR-346", Role: "teacher"},
	"Елена Васильева":   {FIO: "Елена Васильева", Faculty: "Факультет Физики", Group: "CC-15-02", Pass: "AD-314", Role: "admin"},
	"Сергей Иванов":     {FIO: "Сергей Иванов", Faculty: "Факультет Информатики", Group: "AA-25-07", Pass: "ST-458", Role: "student"},
	"Ольга Новикова":    {FIO: "Ольга Новикова", Faculty: "Факультет Механики", Group: "BB-10-07", Pass: "TR-347", Role: "teacher"},
	"Дмитрий Соколов":   {FIO: "Дмитрий Соколов", Faculty: "Факультет Физики", Group: "CC-15-01", Pass: "ST-460", Role: "student"},
	"Анна Кузнецова":    {FIO: "Анна Кузнецова", Faculty: "Факультет Экономики", Group: "EE-20-01", Pass: "ST-461", Role: "student"},
}

// Список факультетов и их групп.
var faculties = map[string][]string{
	"Факультет Информатики": {"AA-25-07", "AA-25-08", "AA-25-09"},
	"Факультет Механики":    {"BB-10-07", "BB-10-08"},
	"Факультет Физики":      {"CC-15-01", "CC-15-02"},
	"Факультет Экономики":   {"EE-20-01", "EE-20-02"},
}

// tempUserData хранит временные данные для регистрации.
type tempUserData struct {
	Faculty  string
	Group    string
	Verified *VerifiedParticipant
}

// Глобальные переменные для регистрации и входа.
var userStates = make(map[int64]string)
var userTempDataMap = make(map[int64]*tempUserData)

var loginStates = make(map[int64]string)
var loginTempDataMap = make(map[int64]*loginData)
