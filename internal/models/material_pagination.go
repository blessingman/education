package models

// MaterialPaginationState хранит состояние пагинации материалов для каждого пользователя
type MaterialPaginationState struct {
	CurrentPage  int    // текущая страница
	Filter       string // текущий фильтр (идентификатор курса)
	ItemsPerPage int    // количество элементов на странице
}
