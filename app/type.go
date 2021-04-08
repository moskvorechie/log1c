package app

import "time"

type Row struct {
	startedMain      bool
	dataStarted      bool
	dataOpenBrackets int

	DataSymbol string
	DataRow    string

	Date        time.Time
	Transaction string
	Row         int
	User        int
	PC          int
	App         int
	Conn        int
	Event       int
	Level       string
	Comment     string
	Meta        int
	Message     string
	Server      int
	Port1       int
	Port2       int
	Session     int
}

type Message struct {
	ID       string
	Allow    bool
	Level    string
	App      string
	NameDB   string
	Instance string

	ДатаВремя          time.Time
	СтатусТранзакции   string
	НомерТранзакции    string
	ПользовательИд     int64
	Пользователь       string
	Компьютер          string
	КомпьютерИд        int64
	Приложение         string
	ПриложениеИд       int64
	Соединение         string
	Событие            string
	СобытиеИд          int64
	Комментарий        string
	Метаданные         string
	МетаданныеИд       int64
	Данные             string
	Представление      string
	Сервер             string
	СерверИд           int64
	Порт1              string
	Порт2              string
	Сеанс              string
	СтатусТранзакцииИд string
	СыраяСтрока        string
	Folder             string
}
