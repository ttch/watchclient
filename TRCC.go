package main
/*
	the project remote Watcher clienter

	patcher by zhaonf(ttchgm@gmail.com)

	Inherited from mars.liu `s project cobweb

*/
import (
	"fmt"
	"net/http"
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"github.com/howeyc/fsnotify"
	"time"
	"bytes"
)

var conf = flag.String("c", "config.json", "gossamer config file path")
var logger chan string = make(chan string)
var count = 0
var l map[string]interface{} = make(map[string]interface{})


/* watch config file */
type Config struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Path        string      `json:"path"`
	DestPath    string      `json:"destpath"`
	Action      string      `json:"action"`
	Include     interface{} `json:"include"`
	Exclude     interface{} `json:"exclude"`
	ExcludeDirs interface{} `json:"exclude-dirs"`
}


/* command struct */
type Command struct {
	Command        string      `json:"Command"`
	Name           string      `json:"Name"`
	Action         string      `json:"Action"`
}

/*
	parse a config file
*/
func ParseConfig(watchIt interface{}) {
	flag.Parse()
	buf, err := ioutil.ReadFile(*conf)
	CheckErr(err)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               

	data := map[string]interface{}{}
	err = json.Unmarshal(buf, &data)

	if gossamer, ok := data["watch"]; ok {
		if configs, ok := gossamer.([]interface{}); ok {
			for _, config := range configs {
				buffer, err := json.Marshal(config)
				CheckErr(err)
				var conf Config
				err = json.Unmarshal(buffer, &conf)
				CheckErr(err)
				
				if f , ok := (watchIt).(func (conf Config));ok{
					f(conf)
				}	
			}
		} else {
			panic(fmt.Errorf("except path settings but got %v\n", configs))
		}
	} else {
		panic(fmt.Errorf("Which paths not found in conf[\"watch\"]\n"))
	}
	for {
		message := <-logger
		fmt.Println(message)
	}
}


/*
	watch a directory
*/
func watchDir(conf Config, path string) {
	watcher, err := fsnotify.NewWatcher()
	CheckErr(err)
	defer watcher.Close()
	logger <- fmt.Sprintf("[watch beginning]  .................")
	
	//watcher.WatchFlags(path, fsnotify.FSN_CREATE|fsnotify.FSN_MODIFY)

	err = watcher.Watch(path)
	if err != nil {
		panic(fmt.Errorf("watcher can't watch %v , got: %v \n count: %v", path, err, count))
	}

	count += 1
	subs, err := ioutil.ReadDir(path)
	CheckErr(err)
	for _, sub := range subs {
		if sub.IsDir() {
			matchExclude := false
			if conf.ExcludeDirs != nil {
				extds := conf.ExcludeDirs.([]interface{})
				for _, extd := range extds {
					extpath := extd.(string)
					matchExclude, err = filepath.Match(extpath, sub.Name())
					CheckErr(err)
					if matchExclude {
						break
					}
				}
			}

			subpath := filepath.Join(path, sub.Name())
			if !matchExclude {
				go watchDir(conf, subpath)
			}
		}
	}

	for {
		select {
		case ev := <-watcher.Event:
			OnNotify(conf, ev)
		case err := <-watcher.Error:
			panic(fmt.Errorf("[error] got error: %v when watch %v", err, conf.Name))
		}
	}
}

/*
	Notify Event
*/
func OnNotify(conf Config, event *fsnotify.FileEvent) {
	if conf.Include != nil {
		include := conf.Include.(string)
		ok, err := filepath.Match(include, filepath.Base(event.Name))
		CheckErr(err)
		if !ok {
			return
		}
	}
	if conf.Exclude != nil {

		exclude := conf.Exclude.(string)
		ok, err := filepath.Match(exclude, filepath.Base(event.Name))
		CheckErr(err)
		if ok {
			return
		}
	}


	logger <- fmt.Sprintf("[watch +] %v : %v", conf.Name, event)

	if event.IsModify() {
		if _,ok := l[event.Name] ; !ok{
			l[event.Name] = &Command{Command:conf.Action, Name:conf.DestPath+"/"+filepath.Base(event.Name), Action:"MODIFY"}
		}else{
			logger <- fmt.Sprintf("[skip] %v" , event.Name)
		}
	}
}


/*
	run command
*/
func RunCommand(key string ,command interface{}) {
	b, _ := json.Marshal(command)

	response,err := http.Post("http://192.168.99.100:11011/listen/touch","application/json;charset=utf-8",bytes.NewBuffer(b))

	if err != nil{
		logger <- fmt.Sprintf("[failed] send command : %v")
		return
	}else{
		time.Sleep(time.Millisecond * 5000)
		delete(l, key)
	}
    
    defer response.Body.Close()
    
	body,_ := ioutil.ReadAll(response.Body)

	logger <- fmt.Sprintf("[recv] server Result: %v ",string(body))

}

/*	checker zhaonf */

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}

/* init timer */

func timer_init(){
	tickChan := time.NewTicker(time.Millisecond * 5000).C
	for {
		select {
			case <- tickChan:
				
				for key,v:=range l{
					RunCommand(key,v)
				}
		}
    }
}

func main() {
	go timer_init()
	ParseConfig(func (conf Config){
		go watchDir(conf, conf.Path)
	})
}







