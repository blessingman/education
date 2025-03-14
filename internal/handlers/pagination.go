package handlers

type PaginationState struct {
	Items      []string // сами названия курсов/групп
	Page       int      // номер текущей страницы (начинаем с 0)
	PageSize   int      // сколько элементов на страницу
	TotalPages int      // общее количество страниц
}

var teacherPages = make(map[int64]*PaginationState)
