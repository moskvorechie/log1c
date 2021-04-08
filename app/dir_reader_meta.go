package app

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strconv"
)

type MetaUser struct {
	ID   int64
	Name string
}

type MetaPC struct {
	ID   int64
	Name string
}

type MetaApp struct {
	ID   int64
	Name string
}

type MetaEvent struct {
	ID   int64
	Name string
}

type MetaSub struct {
	ID   int64
	Name string
}

type MetaServer struct {
	ID   int64
	Name string
}

type Meta struct {
	Users     map[int64]MetaUser
	Computers map[int64]MetaPC
	Apps      map[int64]MetaApp
	Events    map[int64]MetaEvent
	Subs      map[int64]MetaSub
	Servers   map[int64]MetaServer
}

func (r *DirReader) parseMetadata() {

	r.meta = Meta{
		Users:     make(map[int64]MetaUser, 0),
		Computers: make(map[int64]MetaPC, 0),
		Apps:      make(map[int64]MetaApp, 0),
		Events:    make(map[int64]MetaEvent, 0),
		Subs:      make(map[int64]MetaSub, 0),
		Servers:   make(map[int64]MetaServer, 0),
	}

	file, err := os.Open( r.path + "/1Cv8.lgf")
	if err != nil {
		r.logger.FatalError(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	r1 := regexp.MustCompile(`{1,[a-z0-9-]+,"(.*)?",(\d+)}`)
	r2 := regexp.MustCompile(`{2,"?(.*)"?,(\d+)}`)
	r3 := regexp.MustCompile(`{3,"(.*)",(\d+)}`)
	r4 := regexp.MustCompile(`{4,"(.*)",(\d+)}`)
	r5 := regexp.MustCompile(`{5,[a-z0-9-]+,"(.*)?",(\d+)`)
	r6 := regexp.MustCompile(`{6,"(.*)",(\d+)}`)

	for {

		var (
			err error
			row string
		)

		select {
		case <-r.exit:
			return
		default:

			// Read row
			row, err = reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			s := row[:3]
			switch s {

			// Parse User
			case "{1,":
				res := r1.FindStringSubmatch(row)
				if len(res) == 0 {
					r.logger.ErrorF("1 metadata regex err: %+v", row)
					break
				}
				id, err := strconv.ParseInt(res[2], 10, 64)
				if err != nil {
					r.logger.Error(row)
					r.logger.FatalError(err)
				}
				user := MetaUser{
					ID:   id,
					Name: res[1],
				}
				r.meta.Users[user.ID] = user

			// Parse PC
			case "{2,":
				res := r2.FindStringSubmatch(row)
				if len(res) == 0 {
					r.logger.ErrorF("2 metadata regex err: %+v", row)
					break
				}
				id, err := strconv.ParseInt(res[2], 10, 64)
				if err != nil {
					r.logger.Error(row)
					r.logger.FatalError(err)
				}
				pc := MetaPC{
					ID:   id,
					Name: res[1],
				}
				r.meta.Computers[pc.ID] = pc

			// Parse App
			case "{3,":
				res := r3.FindStringSubmatch(row)
				if len(res) == 0 {
					r.logger.ErrorF("3 metadata regex err: %+v", row)
					break
				}
				id, err := strconv.ParseInt(res[2], 10, 64)
				if err != nil {
					r.logger.Error(row)
					r.logger.FatalError(err)
				}
				app := MetaApp{
					ID:   id,
					Name: res[1],
				}
				r.meta.Apps[app.ID] = app

			// Parse Events
			case "{4,":
				res := r4.FindStringSubmatch(row)
				if len(res) == 0 {
					r.logger.ErrorF("4 metadata regex err: %+v", row)
					break
				}
				id, err := strconv.ParseInt(res[2], 10, 64)
				if err != nil {
					r.logger.Error(row)
					r.logger.FatalError(err)
				}
				event := MetaEvent{
					ID:   id,
					Name: res[1],
				}
				r.meta.Events[event.ID] = event

			// Parse Sub
			case "{5,":
				res := r5.FindStringSubmatch(row)
				if len(res) == 0 {
					r.logger.ErrorF("5 metadata regex err: %+v", row)
					break
				}
				id, err := strconv.ParseInt(res[2], 10, 64)
				if err != nil {
					r.logger.Error(row)
					r.logger.FatalError(err)
				}
				sub := MetaSub{
					ID:   id,
					Name: res[1],
				}
				r.meta.Subs[sub.ID] = sub

			// Parse Servers
			case "{6,":
				res := r6.FindStringSubmatch(row)
				if len(res) == 0 {
					r.logger.ErrorF("6 metadata regex err: %+v", row)
					break
				}
				id, err := strconv.ParseInt(res[2], 10, 64)
				if err != nil {
					r.logger.Error(row)
					r.logger.FatalError(err)
				}
				server := MetaServer{
					ID:   id,
					Name: res[1],
				}
				r.meta.Servers[server.ID] = server
			}
		}

		if err == io.EOF {
			break
		}
	}

	r.logger.DebugF("%+v", r.meta)
}
