package app

import (
	"github.com/moskvorechie/logs"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type DirReader struct {
	wg     *sync.WaitGroup
	logger logs.Log
	cfg    *ini.File
	exit   chan bool
	name   string
	path   string
	app    *App
	meta   Meta
}

var (
	metricReadFileDur = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "log1c_watch_read_file_seconds",
		Help: "Как долго читался файл",
	},
		[]string{"server"},
	)
)

func init() {
	prometheus.MustRegister(metricReadFileDur)
}

func (r *DirReader) Run() {

	defer func() {
		if rc := recover(); rc != nil {
			r.logger.Error("Fatal stack: \n" + string(debug.Stack()))
			r.logger.FatalF("Recovered Fatal %v", rc)
		}
	}()

	defer r.wg.Done()

	// Set app to logger
	r.logger.SetCustomLogger(r.logger.Logger().With().Str("log_name", r.name).Logger())

	// Start & stop log
	r.logger.Info("DirReader start")
	defer r.logger.Info("DirReader stop")

	appName := r.cfg.Section("main").Key("app").String()

	// Parse metadata
	filePath, pos := r.prepare("", 0)

	// Read files forever
	for {

		// Check new file
		filePath, pos = r.prepare(filePath, pos)

		select {
		case <-time.After(10 * time.Second):

			// Metric read time >
			tReadDurStart := time.Now()

			// Run FileReader
			var fr FileReader
			if pos > fr.pos {
				fr.pos = pos
			}
			fr.dir = r
			fr.path = filePath
			fr.logger = r.logger
			fr.Run()
			pos = fr.pos

			// Metric read time <
			metricReadFileDur.WithLabelValues(appName).Set(time.Now().Sub(tReadDurStart).Seconds())

		case <-r.exit:
			return
		}
	}
}

// If exist new file we need set new position to end new file
func (r *DirReader) prepare(filePath string, pos int64) (string, int64) {

	// Parse metadata
	r.parseMetadata()

	// Get last file
	newFilePath, err := r.findNewestFile()
	if err != nil {
		r.logger.FatalError(err)
	}

	// Find file pos
	if newFilePath != filePath || pos <= 0 {
		pos, err = r.findFilePos(newFilePath)
		if err != nil {
			r.logger.FatalError(err)
		}
	}

	return newFilePath, pos
}

func (r *DirReader) findFilePos(filePath string) (pos int64, err error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return
	}
	pos = stat.Size()
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	return
}

func (r *DirReader) findNewestFile() (newestFile string, err error) {

	// Get one newest file
	files, err := ioutil.ReadDir(r.path)
	if err != nil {
		r.logger.LogError(err)
		return
	}

	// Hack for reset win cache
	for _, ff := range files {
		flags := os.O_RDONLY | os.O_SYNC
		f, _ := os.OpenFile(r.path+string(os.PathSeparator)+ff.Name(), flags, 0644)
		_ = f.Close()
	}

	// Get newest file by date
	var modTime1 time.Time
	var newestFile1 string
	for _, ff := range files {
		if !ff.Mode().IsRegular() || strings.ToLower(path.Ext(ff.Name())) != ".lgp" {
			continue
		}
		fileCreatedAt := ff.ModTime()
		if !fileCreatedAt.Before(modTime1) && fileCreatedAt.After(modTime1) {
			modTime1 = fileCreatedAt
			newestFile1 = r.path + string(os.PathSeparator) + ff.Name()
		}
	}

	// Get newest file by name
	var modTime2 time.Time
	var newestFile2 string
	for _, ff := range files {
		if !ff.Mode().IsRegular() || strings.ToLower(path.Ext(ff.Name())) != ".lgp" {
			continue
		}
		re, _ := regexp.Compile(`^(\d{8})`)
		name := re.FindString(ff.Name())
		if len(name) <= 0 {
			continue
		}
		fileCreatedAt, err := time.Parse("20060102", name)
		if err != nil {
			continue
		}
		if !fileCreatedAt.Before(modTime2) && fileCreatedAt.After(modTime2) {
			modTime2 = fileCreatedAt
			newestFile2 = r.path + string(os.PathSeparator) + ff.Name()
		}
	}

	// If file by date is newest than by name
	if modTime1.After(modTime2) {
		newestFile = newestFile1
	} else {
		newestFile = newestFile2
	}

	return
}
