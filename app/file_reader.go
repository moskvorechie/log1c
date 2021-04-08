package app

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"github.com/moskvorechie/logs"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type FileReader struct {
	dir    *DirReader
	logger logs.Log
	exit   chan bool
	path   string
	hash   string
	pos    int64
}

func (f *FileReader) Run() {

	defer func() {
		if rc := recover(); rc != nil {
			f.logger.Error("Fatal stack: \n" + string(debug.Stack()))
			f.logger.FatalF("Recovered Fatal %v", rc)
		}
	}()

	// Set app to logger
	f.logger.SetCustomLogger(f.logger.Logger().With().Str("file_name", filepath.Base(f.path)).Logger())

	// Start & stop log
	f.logger.Debug("FileReader start")
	defer f.logger.Debug("FileReader stop")

	// File hash
	f.calcFileHash()

	// Start read file from end
	file, err := os.Open(f.path)
	if err != nil {
		f.logger.FatalError(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	if f.pos <= 0 {
		stat, err := os.Stat(f.path)
		if err != nil {
			f.logger.FatalError(err)
		}
		f.pos = stat.Size()
	}
	if f.dir.cfg.Section("main").Key("test").MustBool() {
		f.pos = 0
	}
	_, err = reader.Discard(int(f.pos))
	if err != nil {
		f.logger.FatalError(err)
	}

	// Read file
	var r Row
	var res []string
	for {
		select {
		case <-f.exit:
			return
		default:

			// Read row
			row, err := reader.ReadString('\n')
			exit := err == io.EOF
			if err != nil && err != io.EOF {
				f.logger.FatalError(err)
			}
			if len(row) < 0 || row == "" {
				if exit {
					return
				} else {
					continue
				}
			}

			f.logger.Debug(row)

			// Save pos
			f.pos += int64(len(row))

			// Parse row
			s := row[:1]
			if s == "{" && r.startedMain == false {
				r.startedMain = true
				r.dataOpenBrackets = 0
			}
			if r.startedMain == false {
				if exit {
					return
				} else {
					continue
				}
			}
			r.DataRow += row
			row = strings.TrimSpace(row)
			r.dataOpenBrackets += strings.Count(row, "{")
			r.dataOpenBrackets -= strings.Count(row, "}")
			if r.startedMain == true && r.dataOpenBrackets == 0 {
				res = f.dir.app.regex1.FindStringSubmatch(r.DataRow)
				if len(res) < 21 {
					f.logger.ErrorF("Regex find not work: %v", r.DataRow)
					r = Row{}
					if exit {
						return
					} else {
						continue
					}
				}
				r = Row{}
				m, err := f.prepareMessage(res)
				if err != nil {
					f.logger.FatalError(err)
				}

				f.logger.DebugF("Send row: %v", m)

				select {
				case <-f.exit:
					return
				default:
					if m.Allow {
						f.dir.app.mess <- m
					}
				}
			}
		}
	}
}

func (f *FileReader) calcFileHash() {
	h := sha256.New()
	h.Write([]byte(f.path))
	f.hash = fmt.Sprintf("%x", h.Sum(nil))
}

func (f *FileReader) prepareMessage(res []string) (m Message, err error) {

	// ID
	h := sha256.New()
	h.Write([]byte(f.hash + strconv.Itoa(int(f.pos))))
	m.ID = fmt.Sprintf("%x", h.Sum(nil))
	m.ДатаВремя, err = time.ParseInLocation("20060102150405", res[1], f.dir.app.loc)
	if err != nil {
		f.logger.LogError(err)
		return
	}

	m.СыраяСтрока = res[0]

	m.СтатусТранзакции = res[2]
	m.СтатусТранзакцииИд = res[2]
	switch res[2] {
	case "N":
		m.СтатусТранзакции = "Отсутствует"
	case "U":
		m.СтатусТранзакции = "Зафиксирована"
	case "R":
		m.СтатусТранзакции = "Не завершена"
	case "C":
		m.СтатусТранзакции = "Отменена"
	}

	m.НомерТранзакции = res[3] + "-" + res[4]

	m.Пользователь = res[5]
	if id, err := strconv.ParseInt(res[5], 10, 64); err == nil {
		m.ПользовательИд = id
		m.Пользователь = f.dir.meta.Users[id].Name
	}

	m.Компьютер = res[6]
	if id, err := strconv.ParseInt(res[6], 10, 64); err == nil {
		m.КомпьютерИд = id
		m.Компьютер = f.dir.meta.Computers[id].Name
	}

	m.Приложение = res[7]
	if id, err := strconv.ParseInt(res[7], 10, 64); err == nil {
		m.ПриложениеИд = id
		m.Приложение = f.dir.meta.Apps[id].Name
	}

	m.Событие = res[9]
	if id, err := strconv.ParseInt(res[9], 10, 64); err == nil {
		m.СобытиеИд = id
		m.Событие = f.dir.meta.Events[id].Name
	}

	m.Метаданные = res[12]
	if id, err := strconv.ParseInt(res[12], 10, 64); err == nil {
		m.МетаданныеИд = id
		m.Метаданные = f.dir.meta.Subs[id].Name
	}

	m.Сервер = res[15]
	if id, err := strconv.ParseInt(res[15], 10, 64); err == nil {
		m.СерверИд = id
		m.Сервер = f.dir.meta.Servers[id].Name
	}

	m.Соединение = res[8]
	m.Комментарий = res[11]
	m.Метаданные = res[12]
	m.Данные = res[13]
	m.Представление = res[14]
	m.Порт1 = res[16]
	m.Порт2 = res[17]
	m.Сеанс = res[18]

	switch res[10] {
	case "I":
		m.Level = "info"
	case "E":
		m.Level = "error"
	case "W":
		m.Level = "warning"
	case "N":
		m.Level = "debug"
	default:
		panic("no type")
	}
	m.NameDB = f.dir.name
	m.App = f.dir.app.name
	m.Folder = f.dir.path

	sendAllow := false
	cfgLevel := f.dir.cfg.Section("main").Key("msg_level").MustString("debug")
	if cfgLevel == "debug" {
		sendAllow = true
	}
	if cfgLevel == "info" && (m.Level == "info" || m.Level == "warning" || m.Level == "error") {
		sendAllow = true
	}
	if cfgLevel == "warning" && (m.Level == "warning" || m.Level == "error") {
		sendAllow = true
	}
	if cfgLevel == "error" && m.Level == "error" {
		sendAllow = true
	}
	m.Allow = sendAllow

	return
}
