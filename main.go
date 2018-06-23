package main

import (
	"encoding/json"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/go-yaml/yaml"
	"github.com/gorilla/sessions"
	"github.com/hunterlong/statup/comms"
	"github.com/hunterlong/statup/plugin"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	plg "plugin"
	"strconv"
	"strings"
)

var (
	configs    *Config
	core       *Core
	store      *sessions.CookieStore
	VERSION    string
	sqlBox     *rice.Box
	cssBox     *rice.Box
	jsBox      *rice.Box
	tmplBox    *rice.Box
	setupMode  bool
	allPlugins []plugin.PluginActions
)

const (
	pluginsRepo = "https://raw.githubusercontent.com/hunterlong/statup/master/plugins.json"
)

type Config struct {
	Connection string `yaml:"connection"`
	Host       string `yaml:"host"`
	Database   string `yaml:"database"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Port       string `yaml:"port"`
	Secret     string `yaml:"secret"`
}

type PluginRepos struct {
	Plugins []PluginJSON
}

type PluginJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Repo        string `json:"repo"`
	Author      string `json:"author"`
	Namespace   string `json:"namespace"`
}

func (c *Core) FetchPluginRepo() []PluginJSON {
	resp, err := http.Get(pluginsRepo)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var pk []PluginJSON
	json.Unmarshal(body, &pk)
	c.Repos = pk
	return pk
}

func DownloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

//func DownloadPlugin(name string) {
//	plugin := SelectPlugin(name)
//	var _, err = os.Stat("plugins/" + plugin.Namespace)
//	if err != nil {
//	}
//	if os.IsNotExist(err) {
//		var file, _ = os.Create("plugins/" + plugin.Namespace)
//		defer file.Close()
//	}
//	resp, err := http.Get("https://raw.githubusercontent.com/hunterlong/statup/master/plugins.json")
//	if err != nil {
//		panic(err)
//	}
//	defer resp.Body.Close()
//	body, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		panic(err)
//	}
//	file, err := os.OpenFile("plugins/"+plugin.Namespace, os.O_RDWR, 0644)
//	if err != nil {
//		panic(err)
//	}
//	defer file.Close()
//
//	_, err = file.Write(body)
//	if err != nil {
//		panic(err)
//	}
//	err = file.Sync()
//}

func main() {
	var err error
	fmt.Printf("Starting Statup v%v\n", VERSION)

	comms.NewMailer()

	RenderBoxes()
	configs, err = LoadConfig()
	if err != nil {
		fmt.Println("config.yml file not found - starting in setup mode")
		setupMode = true
		RunHTTPServer()
	}
	mainProcess()
}

func StringInt(s string) int64 {
	num, _ := strconv.Atoi(s)
	return int64(num)
}

func mainProcess() {
	var err error
	err = DbConnection(configs.Connection)
	if err != nil {
		throw(err)
	}
	core, err = SelectCore()
	if err != nil {
		fmt.Println("Core database was not found, Statup is not setup yet.")
		RunHTTPServer()
	}
	CheckServices()
	if !setupMode {
		LoadPlugins()
		RunHTTPServer()
	}
}

func throw(err error) {
	fmt.Println("ERROR: ", err)
	os.Exit(1)
}

func ForEachPlugin() {
	if len(core.Plugins) > 0 {
		//for _, p := range core.Plugins {
		//	p.OnShutdown()
		//}
	}
}

func LoadPlugins() {
	if _, err := os.Stat("./plugins"); os.IsNotExist(err) {
		os.Mkdir("./plugins", os.ModePerm)
	}

	ForEachPlugin()

	files, err := ioutil.ReadDir("./plugins")
	if err != nil {
		fmt.Printf("Plugins directory was not found. Error: %v\n", err)
		return
	}
	for _, f := range files {
		ext := strings.Split(f.Name(), ".")
		if len(ext) != 2 {
			continue
		}
		if ext[1] != "so" {
			continue
		}
		plug, err := plg.Open("plugins/" + f.Name())
		if err != nil {
			fmt.Printf("Plugin '%v' could not load correctly.\n", f.Name())
			continue
		}
		symPlugin, err := plug.Lookup("Plugin")

		var plugActions plugin.PluginActions
		plugActions, ok := symPlugin.(plugin.PluginActions)
		if !ok {
			fmt.Printf("Plugin '%v' could not load correctly, error: %v\n", f.Name(), "unexpected type from module symbol")
			continue
		}

		allPlugins = append(allPlugins, plugActions)
		core.Plugins = append(core.Plugins, plugActions.GetInfo())
	}

	OnLoad(dbSession)

	fmt.Printf("Loaded %v Plugins\n", len(allPlugins))

	ForEachPlugin()
}

func RenderBoxes() {
	sqlBox = rice.MustFindBox("sql")
	cssBox = rice.MustFindBox("html/css")
	jsBox = rice.MustFindBox("html/js")
	tmplBox = rice.MustFindBox("html/tmpl")
}

func LoadConfig() (*Config, error) {
	var config Config
	file, err := ioutil.ReadFile("config.yml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(file, &config)
	configs = &config
	return &config, err
}

func HashPassword(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes)
}
